package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"practice9/internal/config"
	"practice9/internal/idempotency"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PaymentResponse struct {
	Status        string `json:"status"`
	Amount        int    `json:"amount"`
	TransactionID string `json:"transaction_id"`
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Business logic: Processing started...")
	time.Sleep(2 * time.Second)

	resp := PaymentResponse{
		Status:        "paid",
		Amount:        1000,
		TransactionID: uuid.New().String(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)

	log.Printf("Business logic: Processing finished. tx=%s", resp.TransactionID)
}

func buildStore() idempotency.Store {
	useRedis := config.GetEnvBool("USE_REDIS", true)
	if !useRedis {
		log.Printf("Using MemoryStore (USE_REDIS=false)")
		return idempotency.NewMemoryStore()
	}

	addr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	pass := config.GetEnv("REDIS_PASSWORD", "")
	db := config.GetEnvInt("REDIS_DB", 0)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis ping failed (%v). Falling back to MemoryStore.", err)
		return idempotency.NewMemoryStore()
	}

	log.Printf("Using RedisStore at %s", addr)
	return idempotency.NewRedisStore(rdb)
}

func startServer(addr string, handler http.Handler) *http.Server {
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		log.Printf("Payment server listening on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	return srv
}

type result struct {
	code int
	body string
	dur  time.Duration
}

func main() {
	mode := config.GetEnv("MODE", "demo") // demo or server
	addr := config.GetEnv("PAY_SERVER_ADDR", "127.0.0.1:8090")

	processingTTL := config.GetEnvDurationSeconds("PROCESSING_TTL_SECONDS", 10)
	resultTTL := config.GetEnvDurationSeconds("RESULT_TTL_SECONDS", 3600)

	store := buildStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/pay", paymentHandler)
	handler := idempotency.Middleware(store, processingTTL, resultTTL, mux)

	srv := startServer(addr, handler)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if mode == "server" {
		select {}
	}

	baseURL := "http://" + addr + "/pay"
	key := uuid.New().String()
	log.Printf("Demo Idempotency-Key: %s", key)

	client := &http.Client{Timeout: 10 * time.Second}

	reqNoKey, _ := http.NewRequest(http.MethodPost, baseURL, nil)
	respNoKey, err := client.Do(reqNoKey)
	if err == nil && respNoKey != nil {
		log.Printf("Missing key response: %d (expected 400)", respNoKey.StatusCode)
		respNoKey.Body.Close()
	}

	var wg sync.WaitGroup
	results := make([]result, 8)

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := time.Now()

			req, _ := http.NewRequest(http.MethodPost, baseURL, nil)
			req.Header.Set("Idempotency-Key", key)

			resp, err := client.Do(req)
			if err != nil {
				results[i] = result{code: 0, body: err.Error(), dur: time.Since(start)}
				return
			}
			defer resp.Body.Close()

			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(resp.Body)
			results[i] = result{code: resp.StatusCode, body: buf.String(), dur: time.Since(start)}
		}(i)
	}

	wg.Wait()

	first200 := ""
	for i, r := range results {
		log.Printf("Concurrent req #%d -> code=%d dur=%v body=%s", i+1, r.code, r.dur, trim(r.body, 90))
		if r.code == 200 && first200 == "" {
			first200 = r.body
		}
	}

	time.Sleep(200 * time.Millisecond)
	start := time.Now()
	reqAgain, _ := http.NewRequest(http.MethodPost, baseURL, nil)
	reqAgain.Header.Set("Idempotency-Key", key)

	respAgain, err := client.Do(reqAgain)
	if err != nil {
		log.Fatalf("repeat request failed: %v", err)
	}
	defer respAgain.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(respAgain.Body)
	bodyAgain := buf.String()

	log.Printf("Repeat request -> code=%d dur=%v body=%s", respAgain.StatusCode, time.Since(start), trim(bodyAgain, 140))
	log.Printf("Repeat body equals first success body: %v", bodyAgain == first200)

	time.Sleep(200 * time.Millisecond)
}

func trim(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
