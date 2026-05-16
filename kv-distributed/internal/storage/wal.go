package storage

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"os"
	"sync"
)

const (
	OpPut byte = 1
	OpDel byte = 2
)

const CompactionThreshold = 100000 // Trigger compaction after 100,000 operations

// WAL handles the append-only log for durability.
type WAL struct {
	mu         sync.Mutex
	file       *os.File
	filename   string
	opCount    int
	compacting bool
}

func NewWAL(filename string) (*WAL, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &WAL{
		file:     file,
		filename: filename,
	}, nil
}

// Append writes an operation to the log. Returns true if compaction should be triggered.
func (w *WAL) Append(op byte, bucket, key string, value []byte, ttl int64) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	bucketBytes := []byte(bucket)
	keyBytes := []byte(key)

	size := 1 + 2 + len(bucketBytes) + 2 + len(keyBytes) + 8 + 4 + len(value)
	buf := make([]byte, size)

	offset := 0
	buf[offset] = op
	offset++

	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(bucketBytes)))
	offset += 2
	copy(buf[offset:], bucketBytes)
	offset += len(bucketBytes)

	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(keyBytes)))
	offset += 2
	copy(buf[offset:], keyBytes)
	offset += len(keyBytes)

	binary.LittleEndian.PutUint64(buf[offset:], uint64(ttl))
	offset += 8

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(value)))
	offset += 4
	if len(value) > 0 {
		copy(buf[offset:], value)
	}

	_, err := w.file.Write(buf)
	if err != nil {
		return false, err
	}
	
	w.opCount++
	shouldCompact := false
	if w.opCount >= CompactionThreshold && !w.compacting {
		w.compacting = true
		shouldCompact = true
	}

	return shouldCompact, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

// Compact shrinks the WAL file by writing only current active state to a new file.
func (w *WAL) Compact(s *Storage) {
	log.Println("Starting background WAL Compaction...")
	
	tmpFilename := w.filename + ".tmp"
	tmpFile, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Printf("WAL Compaction failed to create temp file: %v", err)
		w.mu.Lock()
		w.compacting = false
		w.mu.Unlock()
		return
	}

	shards := s.GetShards()
	for i := 0; i < len(shards); i++ {
		shard := shards[i]
		shard.mu.RLock()
		for bucket, b := range shard.Buckets {
			for key, val := range b {
				if s.checkTTL(shard, bucket, key) {
					ttl := int64(0)
					if bucketTTL, ok := shard.TTL[bucket]; ok {
						if t, ok := bucketTTL[key]; ok {
							ttl = t.Expiry
						}
					}
					writeOpToTemp(tmpFile, OpPut, bucket, key, val, ttl)
				}
			}
		}
		shard.mu.RUnlock()
	}

	tmpFile.Sync()
	tmpFile.Close()

	w.mu.Lock()
	defer w.mu.Unlock()

	w.file.Close()
	os.Remove(w.filename)
	os.Rename(tmpFilename, w.filename)

	newFile, err := os.OpenFile(w.filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err == nil {
		w.file = newFile
	}
	
	w.opCount = 0
	w.compacting = false
	log.Println("WAL Compaction complete!")
}

func writeOpToTemp(file *os.File, op byte, bucket, key string, value []byte, ttl int64) {
	bucketBytes := []byte(bucket)
	keyBytes := []byte(key)
	size := 1 + 2 + len(bucketBytes) + 2 + len(keyBytes) + 8 + 4 + len(value)
	buf := make([]byte, size)

	offset := 0
	buf[offset] = op
	offset++

	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(bucketBytes)))
	offset += 2
	copy(buf[offset:], bucketBytes)
	offset += len(bucketBytes)

	binary.LittleEndian.PutUint16(buf[offset:], uint16(len(keyBytes)))
	offset += 2
	copy(buf[offset:], keyBytes)
	offset += len(keyBytes)

	binary.LittleEndian.PutUint64(buf[offset:], uint64(ttl))
	offset += 8

	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(value)))
	offset += 4
	if len(value) > 0 {
		copy(buf[offset:], value)
	}

	file.Write(buf)
}

// Replay reads the WAL and applies it to the storage shards.
func (w *WAL) Replay(apply func(op byte, bucket, key string, value []byte, ttl int64)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(w.file)

	for {
		op, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		var bucketLen uint16
		err = binary.Read(reader, binary.LittleEndian, &bucketLen)
		if err != nil {
			return err
		}

		bucketBytes := make([]byte, bucketLen)
		_, err = io.ReadFull(reader, bucketBytes)
		if err != nil {
			return err
		}

		var keyLen uint16
		err = binary.Read(reader, binary.LittleEndian, &keyLen)
		if err != nil {
			return err
		}

		keyBytes := make([]byte, keyLen)
		_, err = io.ReadFull(reader, keyBytes)
		if err != nil {
			return err
		}

		var ttl uint64
		err = binary.Read(reader, binary.LittleEndian, &ttl)
		if err != nil {
			return err
		}

		var valLen uint32
		err = binary.Read(reader, binary.LittleEndian, &valLen)
		if err != nil {
			return err
		}

		var value []byte
		if valLen > 0 {
			value = make([]byte, valLen)
			_, err = io.ReadFull(reader, value)
			if err != nil {
				return err
			}
		}

		apply(op, string(bucketBytes), string(keyBytes), value, int64(ttl))
		w.opCount++ // Track loaded operations
	}

	return nil
}
