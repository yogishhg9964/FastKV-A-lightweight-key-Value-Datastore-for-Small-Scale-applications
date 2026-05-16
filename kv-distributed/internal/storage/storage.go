package storage

import (
	"errors"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

const ShardCount = 256

// Shard manages a slice of the database keys space with its own lock
type Shard struct {
	mu      sync.RWMutex
	Buckets map[string]map[string][]byte
	TTL     map[string]map[string]TTLInfo
}

type TTLInfo struct {
	Expiry int64
}

type Storage struct {
	shards [ShardCount]*Shard
	wal    *WAL
	bytePool sync.Pool

	walWrites       uint64
	walBytesWritten uint64
	walLatencyNs    uint64
}

func (s *Storage) GetWALMetrics() (writes uint64, bytes uint64, latencyNs uint64) {
	return atomic.LoadUint64(&s.walWrites), atomic.LoadUint64(&s.walBytesWritten), atomic.LoadUint64(&s.walLatencyNs)
}

func NewStorage() *Storage {
	s := &Storage{
		bytePool: sync.Pool{
			New: func() interface{} {
				// Allocate a 1KB buffer by default
				b := make([]byte, 0, 1024)
				return &b
			},
		},
	}
	for i := 0; i < ShardCount; i++ {
		s.shards[i] = &Shard{
			Buckets: make(map[string]map[string][]byte),
			TTL:     make(map[string]map[string]TTLInfo),
		}
	}
	return s
}

// getShardIndex calculates the FNV-1a hash of the key to distribute evenly
func getShardIndex(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32() % ShardCount
}

func (s *Storage) getShard(key string) *Shard {
	return s.shards[getShardIndex(key)]
}

// SetPersistenceFile initializes the WAL
func (s *Storage) SetPersistenceFile(filename string) error {
	wal, err := NewWAL(filename)
	if err != nil {
		return err
	}
	s.wal = wal
	return nil
}

func (s *Storage) LoadFromFile() error {
	if s.wal == nil {
		return errors.New("persistence not initialized")
	}

	err := s.wal.Replay(func(op byte, bucket, key string, value []byte, ttl int64) {
		if op == OpPut {
			s.putInternal(bucket, key, value, ttl)
		} else if op == OpDel {
			s.deleteInternal(bucket, key)
		}
	})
	return err
}

func (s *Storage) SaveToFile() error {
	// With WAL, SaveToFile as a snapshot is not strictly needed for basic durability,
	// but kept for API compatibility. A real implementation would do compaction here.
	return nil
}

func (s *Storage) Put(bucket, key string, value []byte) error {
	if s.wal != nil {
		start := time.Now()
		shouldCompact, err := s.wal.Append(OpPut, bucket, key, value, 0)
		latency := time.Since(start).Nanoseconds()
		
		if err == nil {
			atomic.AddUint64(&s.walWrites, 1)
			atomic.AddUint64(&s.walBytesWritten, uint64(len(bucket)+len(key)+len(value)+21))
			atomic.AddUint64(&s.walLatencyNs, uint64(latency))
		}
		
		if err != nil {
			return err
		}
		if shouldCompact {
			go s.wal.Compact(s)
		}
	}
	s.putInternal(bucket, key, value, 0)
	return nil
}

func (s *Storage) PutWithTTL(bucket, key string, value []byte, ttlSeconds int64) error {
	ttl := int64(0)
	if ttlSeconds > 0 {
		ttl = time.Now().Unix() + ttlSeconds
	}
	
	if s.wal != nil {
		start := time.Now()
		shouldCompact, err := s.wal.Append(OpPut, bucket, key, value, ttl)
		latency := time.Since(start).Nanoseconds()
		
		if err == nil {
			atomic.AddUint64(&s.walWrites, 1)
			atomic.AddUint64(&s.walBytesWritten, uint64(len(bucket)+len(key)+len(value)+21))
			atomic.AddUint64(&s.walLatencyNs, uint64(latency))
		}
		
		if err != nil {
			return err
		}
		if shouldCompact {
			go s.wal.Compact(s)
		}
	}
	s.putInternal(bucket, key, value, ttl)
	return nil
}

// putInternal does the actual memory mutation without WAL logging
func (s *Storage) putInternal(bucket, key string, value []byte, ttl int64) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.Buckets[bucket]; !ok {
		shard.Buckets[bucket] = make(map[string][]byte)
	}

	// Make a copy of the value because the caller might modify the slice
	vCopy := make([]byte, len(value))
	copy(vCopy, value)
	shard.Buckets[bucket][key] = vCopy

	if ttl > 0 {
		if _, ok := shard.TTL[bucket]; !ok {
			shard.TTL[bucket] = make(map[string]TTLInfo)
		}
		shard.TTL[bucket][key] = TTLInfo{Expiry: ttl}
	}
}

func (s *Storage) Get(bucket, key string) ([]byte, error) {
	shard := s.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if !s.checkTTL(shard, bucket, key) {
		return nil, errors.New("key expired")
	}

	if b, ok := shard.Buckets[bucket]; ok {
		if val, ok := b[key]; ok {
			return val, nil
		}
	}
	return nil, errors.New("key not found")
}

func (s *Storage) Delete(bucket, key string) error {
	if s.wal != nil {
		start := time.Now()
		shouldCompact, err := s.wal.Append(OpDel, bucket, key, nil, 0)
		latency := time.Since(start).Nanoseconds()
		
		if err == nil {
			atomic.AddUint64(&s.walWrites, 1)
			atomic.AddUint64(&s.walBytesWritten, uint64(len(bucket)+len(key)+21))
			atomic.AddUint64(&s.walLatencyNs, uint64(latency))
		}
		
		if err != nil {
			return err
		}
		if shouldCompact {
			go s.wal.Compact(s)
		}
	}
	return s.deleteInternal(bucket, key)
}

func (s *Storage) deleteInternal(bucket, key string) error {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if b, ok := shard.Buckets[bucket]; ok {
		if _, exists := b[key]; exists {
			delete(b, key)
			if ttlBucket, ok := shard.TTL[bucket]; ok {
				delete(ttlBucket, key)
			}
			return nil
		}
		return errors.New("key not found")
	}
	return errors.New("bucket not found")
}

func (s *Storage) Update(bucket, key string, value []byte) error {
	// Verify it exists first
	_, err := s.Get(bucket, key)
	if err != nil {
		return err
	}
	return s.Put(bucket, key, value)
}

func (s *Storage) List(bucket string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	
	for i := 0; i < ShardCount; i++ {
		shard := s.shards[i]
		shard.mu.RLock()
		if b, ok := shard.Buckets[bucket]; ok {
			for k, v := range b {
				if s.checkTTL(shard, bucket, k) {
					result[k] = v
				}
			}
		}
		shard.mu.RUnlock()
	}
	
	if len(result) == 0 {
		return nil, errors.New("bucket not found or empty")
	}
	return result, nil
}

// checkTTL is called under a read lock. It returns false if expired (and marks it to be deleted, 
// though actually deleting it under RLock is not safe, so we should really upgrade to Lock,
// but for simplicity in this read path we just return false and let auto cleanup or next write fix it.
// Wait, the original code called delete() inside RLock which is a race condition! Let's fix that.
func (s *Storage) checkTTL(shard *Shard, bucket, key string) bool {
	if bucketTTL, ok := shard.TTL[bucket]; ok {
		if t, ok := bucketTTL[key]; ok {
			if time.Now().Unix() > t.Expiry {
				// We don't delete here anymore to avoid panics on RWMutex.
				// Background cleanup or next write will delete it.
				return false
			}
		}
	}
	return true
}

// Export shards for indexer scan
func (s *Storage) GetShards() [ShardCount]*Shard {
	return s.shards
}

func (s *Storage) ScanShard(bucket string, shardIndex int) []string {
	if shardIndex < 0 || shardIndex >= ShardCount {
		return nil
	}
	shard := s.shards[shardIndex]
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	var keys []string
	if b, ok := shard.Buckets[bucket]; ok {
		for k := range b {
			if s.checkTTL(shard, bucket, k) {
				keys = append(keys, k)
			}
		}
	}
	return keys
}

func (s *Storage) CheckTTLAndClean(shard *Shard, bucket, key string) bool {
	// A safe cleanup function that acquires a write lock if needed
	if bucketTTL, ok := shard.TTL[bucket]; ok {
		if t, ok := bucketTTL[key]; ok {
			if time.Now().Unix() > t.Expiry {
				shard.mu.RUnlock() // unlock read
				shard.mu.Lock()    // lock write
				delete(shard.Buckets[bucket], key)
				delete(shard.TTL[bucket], key)
				shard.mu.Unlock()  // unlock write
				shard.mu.RLock()   // re-acquire read
				return false
			}
		}
	}
	return true
}
