// Package middleware holds generic, domain-agnostic HTTP middleware. Keeping it
// free of any domain imports (e.g. auth) avoids import cycles — domain-specific
// middleware like session auth lives with its own package.
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"candella-ecosystem/backend/pkg/response"
)

// RateLimiter is a simple in-memory, fixed-window limiter keyed by client IP.
// It protects sensitive endpoints (login, code verification, …) from brute force.
// In-process only — a multi-instance deployment should back this with Redis; this
// is intentionally the minimal, dependency-free version for now.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*window
	limit    int
	interval time.Duration
}

type window struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter allows `limit` requests per `interval` per client IP.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{clients: make(map[string]*window), limit: limit, interval: interval}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w, ok := rl.clients[key]
	if !ok || now.After(w.resetAt) {
		rl.clients[key] = &window{count: 1, resetAt: now.Add(rl.interval)}
		rl.purge(now) // opportunistic cleanup of stale entries
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}

// purge drops expired windows so the map doesn't grow unbounded. Caller holds mu.
func (rl *RateLimiter) purge(now time.Time) {
	for k, w := range rl.clients {
		if now.After(w.resetAt) {
			delete(rl.clients, k)
		}
	}
}

// Middleware returns the handler wrapper; over-limit requests get a 429.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			response.Error(w, http.StatusTooManyRequests, "rate_limited",
				"Too many requests. Please slow down and try again.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the host portion of RemoteAddr (chi's RealIP middleware has
// already normalised this to the true client IP behind proxies).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
