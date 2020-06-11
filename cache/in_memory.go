package cache

import (
	"github.com/apex/log"
	"sync"
	"time"
)

type Item struct {
	Content    []byte
	Expiration int64
}

func (item *Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type Storage struct {
	items map[string]Item
	mu    *sync.RWMutex
}

func NewStorage() *Storage {
	storage := &Storage{
		items: make(map[string]Item),
		mu:    &sync.RWMutex{},
	}

	go storage.RunCacheEvictionRoutine()

	return storage
}

func (s *Storage) RunCacheEvictionRoutine() {
	for {
		select {
		case <-time.After(time.Second * 60):
			for key, item := range s.items {
				if item.Expired() {
					delete(s.items, key)
					log.Debug("Evicted cache key: " + key)
				}
			}
		}
	}
}

func (s *Storage) Get(key string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[key]

	if !exists {
		return nil
	}

	if item.Expired() {
		delete(s.items, key)

		return nil
	}

	return item.Content
}

func (s *Storage) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[key]

	if !exists {
		return false
	}

	if item.Expired() {
		delete(s.items, key)

		return false
	}

	return true
}

func (s *Storage) Set(key string, content []byte, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = Item{
		Content:    content,
		Expiration: time.Now().Add(duration).UnixNano(),
	}
}
