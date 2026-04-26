package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"
)

func Middleware(store Store, processingTTL, resultTTL time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		if cached, exists, err := store.Get(ctx, key); err == nil && exists {
			if cached.Completed {
				body, _ := cached.BodyBytes()
				w.WriteHeader(cached.StatusCode)
				w.Write(body)
				return
			}
			http.Error(w, "Duplicate request in progress", http.StatusConflict)
			return
		} else if err != nil {
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}

		ok, err := store.StartProcessing(ctx, key, processingTTL)
		if err != nil {
			http.Error(w, "Storage error", http.StatusInternalServerError)
			return
		}
		if !ok {
			if cached, exists, err2 := store.Get(ctx, key); err2 == nil && exists && cached.Completed {
				body, _ := cached.BodyBytes()
				w.WriteHeader(cached.StatusCode)
				w.Write(body)
				return
			}
			http.Error(w, "Duplicate request in progress", http.StatusConflict)
			return
		}

		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		_ = store.Finish(context.Background(), key, rec.Code, rec.Body.Bytes(), resultTTL)

		for k, vals := range rec.Header() {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(rec.Code)
		w.Write(rec.Body.Bytes())
	})
}
