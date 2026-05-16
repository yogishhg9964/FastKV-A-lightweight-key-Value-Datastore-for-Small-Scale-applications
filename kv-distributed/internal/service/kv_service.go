package service

import (
	"time"

	"kv-distributed/internal/datastructures"
	"kv-distributed/internal/indexing"
	"kv-distributed/internal/storage"
)

type KVService struct {
	storage     *storage.Storage
	indexer     *indexing.Indexer
	dataStructs *datastructures.DataStructuresService
	pubsub      *PubSub
	stopTTL     chan bool
}

func NewKVService(storage *storage.Storage, indexer *indexing.Indexer, dataStructs *datastructures.DataStructuresService) *KVService {
	service := &KVService{
		storage:     storage,
		indexer:     indexer,
		dataStructs: dataStructs,
		pubsub:      NewPubSub(),
		stopTTL:     make(chan bool),
	}

	go service.autoCleanupTTL()
	return service
}

func (s *KVService) Stop() {
	s.stopTTL <- true
}

func (s *KVService) autoCleanupTTL() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			// TTL is handled at storage access time
		case <-s.stopTTL:
			ticker.Stop()
			return
		}
	}
}

func (s *KVService) Put(bucket, key string, value []byte) {
	s.storage.Put(bucket, key, value)
}

func (s *KVService) PutWithTTL(bucket, key string, value []byte, ttlSeconds int64) {
	s.storage.PutWithTTL(bucket, key, value, ttlSeconds)
}

func (s *KVService) Get(bucket, key string) ([]byte, error) {
	return s.storage.Get(bucket, key)
}

func (s *KVService) Delete(bucket, key string) error {
	return s.storage.Delete(bucket, key)
}

func (s *KVService) Update(bucket, key string, value []byte) error {
	return s.storage.Update(bucket, key, value)
}

func (s *KVService) List(bucket string) (map[string][]byte, error) {
	return s.storage.List(bucket)
}

func (s *KVService) ListKeysByPrefix(bucket, prefix string) ([]string, error) {
	data, err := s.storage.List(bucket)
	if err != nil {
		return nil, err
	}
	return s.indexer.ListKeysByPrefix(data, prefix), nil
}

func (s *KVService) ListKeysInRange(bucket, start, end string) ([]string, error) {
	data, err := s.storage.List(bucket)
	if err != nil {
		return nil, err
	}
	return s.indexer.ListKeysInRange(data, start, end), nil
}

func (s *KVService) ScanKeys(bucket string, cursor int, count int) ([]string, int) {
	if count <= 0 {
		count = 10 // Default count
	}
	
	var keys []string
	currentShard := cursor
	
	for currentShard < 256 && len(keys) < count {
		shardKeys := s.storage.ScanShard(bucket, currentShard)
		keys = append(keys, shardKeys...)
		currentShard++
	}
	
	if currentShard >= 256 {
		return keys, 0
	}
	return keys, currentShard
}

func (s *KVService) SetAdd(setName, value string) {
	s.dataStructs.SetAdd(setName, value)
}

func (s *KVService) SetRemove(setName, value string) error {
	return s.dataStructs.SetRemove(setName, value)
}

func (s *KVService) SetList(setName string) ([]string, error) {
	return s.dataStructs.SetList(setName)
}

func (s *KVService) SortedListAdd(listName, value string) {
	s.dataStructs.SortedListAdd(listName, value)
}

func (s *KVService) SortedListGet(listName string) ([]string, error) {
	return s.dataStructs.SortedListGet(listName)
}

func (s *KVService) MapPut(mapName, key, value string) {
	s.dataStructs.MapPut(mapName, key, value)
}

func (s *KVService) MapGet(mapName, key string) (interface{}, error) {
	return s.dataStructs.MapGet(mapName, key)
}

func (s *KVService) QueuePush(name, value string) {
	s.dataStructs.QueuePush(name, value)
}

func (s *KVService) QueuePop(name string) (string, error) {
	return s.dataStructs.QueuePop(name)
}

func (s *KVService) QueuePeek(name string) (string, error) {
	return s.dataStructs.QueuePeek(name)
}

func (s *KVService) StackPush(name, value string) {
	s.dataStructs.StackPush(name, value)
}

func (s *KVService) StackPop(name string) (string, error) {
	return s.dataStructs.StackPop(name)
}

func (s *KVService) StackPeek(name string) (string, error) {
	return s.dataStructs.StackPeek(name)
}

func (s *KVService) SaveToFile(filename string) error {
	s.storage.SetPersistenceFile(filename)
	return s.storage.SaveToFile()
}

func (s *KVService) LoadFromFile(filename string) error {
	s.storage.SetPersistenceFile(filename)
	return s.storage.LoadFromFile()
}

func (s *KVService) Subscribe(channel string) chan string {
	return s.pubsub.Subscribe(channel)
}

func (s *KVService) Unsubscribe(channel string, ch chan string) {
	s.pubsub.Unsubscribe(channel, ch)
}

func (s *KVService) Publish(channel string, message string) {
	s.pubsub.Publish(channel, message)
}
