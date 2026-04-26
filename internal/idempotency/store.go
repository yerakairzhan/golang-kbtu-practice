package idempotency

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type CachedResponse struct {
	StatusCode int    `json:"status"`
	BodyBase64 string `json:"body_base64"`
	Completed  bool   `json:"completed"`
}

func NewCompleted(status int, body []byte) *CachedResponse {
	return &CachedResponse{
		StatusCode: status,
		BodyBase64: base64.StdEncoding.EncodeToString(body),
		Completed:  true,
	}
}

func (c *CachedResponse) BodyBytes() ([]byte, error) {
	if c.BodyBase64 == "" {
		return []byte{}, nil
	}
	return base64.StdEncoding.DecodeString(c.BodyBase64)
}

func (c *CachedResponse) Marshal() (string, error) {
	b, err := json.Marshal(c)
	return string(b), err
}

func UnmarshalCached(s string) (*CachedResponse, error) {
	var c CachedResponse
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, key string) (*CachedResponse, bool, error)
	StartProcessing(ctx context.Context, key string, processingTTL time.Duration) (bool, error)
	Finish(ctx context.Context, key string, status int, body []byte, resultTTL time.Duration) error
}
