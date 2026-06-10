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

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
	limit   int
	window  time.Duration
	now     func() time.Time
}

func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return newRateLimiter(limit, window, time.Now).middleware
}

func rateLimit(limit int, window time.Duration, nowFunc func() time.Time) func(http.Handler) http.Handler {
	return newRateLimiter(limit, window, nowFunc).middleware
}

func newRateLimiter(limit int, window time.Duration, nowFunc func() time.Time) *rateLimiter {
	return &rateLimiter{
		entries: make(map[string]rateLimitEntry),
		limit:   limit,
		window:  window,
		now:     nowFunc,
	}
}

func (l *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rateLimitKey(r)
		now := l.now()

		l.mu.Lock()
		for entryKey, entry := range l.entries {
			if now.After(entry.resetAt) {
				delete(l.entries, entryKey)
			}
		}

		entry := l.entries[key]
		if now.After(entry.resetAt) {
			entry = rateLimitEntry{resetAt: now.Add(l.window)}
		}
		entry.count++
		l.entries[key] = entry
		blocked := entry.count > l.limit
		l.mu.Unlock()

		if blocked {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *rateLimiter) entryCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

func rateLimitKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return r.URL.Path + ":" + host
}
