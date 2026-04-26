package idempotency

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu   sync.Mutex
	data map[string]*CachedResponse
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]*CachedResponse)}
}

func (m *MemoryStore) Get(ctx context.Context, key string) (*CachedResponse, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	resp, ok := m.data[key]
	return resp, ok, nil
}

func (m *MemoryStore) StartProcessing(ctx context.Context, key string, processingTTL time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = &CachedResponse{Completed: false}
	return true, nil
}

func (m *MemoryStore) Finish(ctx context.Context, key string, status int, body []byte, resultTTL time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = NewCompleted(status, body)
	return nil
}
