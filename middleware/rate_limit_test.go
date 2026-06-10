package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitBlocksAfterThreshold(t *testing.T) {
	now := time.Now()
	limiter := newRateLimiter(1, time.Minute, func() time.Time { return now })
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.1:1234"

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d", second.Code)
	}
}

func TestRateLimitCleansExpiredEntries(t *testing.T) {
	now := time.Now()
	limiter := newRateLimiter(1, time.Minute, func() time.Time { return now })
	handler := limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got := limiter.entryCount(); got != 1 {
		t.Fatalf("entry count before expiry = %d", got)
	}

	now = now.Add(2 * time.Minute)
	req2 := httptest.NewRequest(http.MethodGet, "/other", nil)
	req2.RemoteAddr = "198.51.100.2:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if got := limiter.entryCount(); got != 1 {
		t.Fatalf("expired entry was not cleaned up, count = %d", got)
	}
}
