package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetRate(t *testing.T) {
	t.Run("successful scenario", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"base":"USD","target":"EUR","rate":0.85}`))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		rate, err := service.GetRate("USD", "EUR")

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if rate != 0.85 {
			t.Errorf("expected rate 0.85, got %f", rate)
		}
	})

	t.Run("API business error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"invalid currency pair"}`))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		_, err := service.GetRate("USD", "XXX")

		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid currency pair") {
			t.Errorf("expected error to contain 'invalid currency pair', got %v", err)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		_, err := service.GetRate("USD", "EUR")

		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "decode error") {
			t.Errorf("expected decode error, got %v", err)
		}
	})

	t.Run("slow response timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(6 * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"base":"USD","target":"EUR","rate":0.85}`))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		_, err := service.GetRate("USD", "EUR")

		if err == nil {
			t.Error("expected timeout error, got nil")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("expected network error, got %v", err)
		}
	})

	t.Run("server panic 500 internal server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		_, err := service.GetRate("USD", "EUR")

		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected status: 500") && !strings.Contains(err.Error(), "internal server error") {
			t.Errorf("expected 500 error, got %v", err)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(``))
		}))
		defer server.Close()

		service := NewExchangeService(server.URL)
		_, err := service.GetRate("USD", "EUR")

		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "decode error") {
			t.Errorf("expected decode error, got %v", err)
		}
	})
}
