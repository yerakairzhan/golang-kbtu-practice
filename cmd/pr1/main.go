package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"practice9/internal/config"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func IsRetryable(resp *http.Response, err error) bool {
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return true
		}
		return true
	}
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	case 401, 404:
		return false
	default:
		return false
	}
}

func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	backoff := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt-1))
	if backoff > float64(cfg.MaxDelay) {
		backoff = float64(cfg.MaxDelay)
	}
	maxBackoff := time.Duration(backoff)
	if maxBackoff <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(maxBackoff)))
}

type PaymentClient struct {
	HTTP *http.Client
	Cfg  RetryConfig
}

func (c *PaymentClient) ExecutePayment(ctx context.Context, url string) error {
	var lastErr error

	for attempt := 1; attempt <= c.Cfg.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		resp, err := c.HTTP.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}

		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			var body struct {
				Status string `json:"status"`
			}
			if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr == nil && body.Status == "success" {
				log.Printf("Attempt %d: Success!", attempt)
				return nil
			}
			return fmt.Errorf("unexpected response body")
		}

		lastErr = summarizeFailure(resp, err)
		if attempt == c.Cfg.MaxRetries {
			break
		}

		if !IsRetryable(resp, err) {
			log.Printf("Attempt %d failed (non-retryable): %v", attempt, lastErr)
			return lastErr
		}

		wait := CalculateBackoff(attempt, c.Cfg)
		log.Printf("Attempt %d failed: waiting ~%v before next retry...", attempt, wait)

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("payment failed after %d attempts: %w", c.Cfg.MaxRetries, lastErr)
}

func summarizeFailure(resp *http.Response, err error) error {
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	return fmt.Errorf("status %d", resp.StatusCode)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count <= 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status":"unavailable"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"success"}`)
	}))
	defer srv.Close()

	cfg := RetryConfig{
		MaxRetries: config.GetEnvInt("MAX_RETRIES", 5),
		BaseDelay:  config.GetEnvDurationMs("BASE_DELAY_MS", 500),
		MaxDelay:   config.GetEnvDurationMs("MAX_DELAY_MS", 5000),
	}
	timeout := config.GetEnvDurationSeconds("PAYMENT_TIMEOUT_SECONDS", 10)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &PaymentClient{
		HTTP: &http.Client{Timeout: 5 * time.Second},
		Cfg:  cfg,
	}

	log.Printf("Starting ExecutePayment (timeout=%v, maxRetries=%d)...", timeout, cfg.MaxRetries)
	if err := client.ExecutePayment(ctx, srv.URL); err != nil {
		log.Printf("Final result: FAILED: %v", err)
		return
	}
	log.Printf("Final result: SUCCESS")
}
