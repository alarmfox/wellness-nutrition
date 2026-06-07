package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	entries := make(map[string]rateLimitEntry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
			now := time.Now()

			mu.Lock()
			entry := entries[key]
			if now.After(entry.resetAt) {
				entry = rateLimitEntry{resetAt: now.Add(window)}
			}
			entry.count++
			entries[key] = entry
			blocked := entry.count > limit
			mu.Unlock()

			if blocked {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return r.URL.Path + ":" + host
}
