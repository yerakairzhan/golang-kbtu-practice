package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func (s *RedisStore) Get(ctx context.Context, key string) (*CachedResponse, bool, error) {
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if val == "processing" {
		return &CachedResponse{Completed: false}, true, nil
	}
	cached, err := UnmarshalCached(val)
	if err != nil {
		return nil, true, err
	}
	return cached, true, nil
}

func (s *RedisStore) StartProcessing(ctx context.Context, key string, processingTTL time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, key, "processing", processingTTL).Result()
}

func (s *RedisStore) Finish(ctx context.Context, key string, status int, body []byte, resultTTL time.Duration) error {
	cached := NewCompleted(status, body)
	str, err := cached.Marshal()
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, str, resultTTL).Err()
}
