package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter is a small in-memory fixed-window limiter keyed by client IP.
// It is meant for a single instance (no shared state) and for low-volume
// endpoints such as /auth/login.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	count int
	start time.Time
}

// NewRateLimiter allows at most limit requests per key during each window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		buckets: make(map[string]*bucket),
	}
}

// Allow records one request for key and reports whether it is within the limit.
// When it is not, retryAfter tells how long the client should wait.
func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := rl.now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.Sub(rl.lastSweep) >= rl.window {
		for k, b := range rl.buckets {
			if now.Sub(b.start) >= rl.window {
				delete(rl.buckets, k)
			}
		}
		rl.lastSweep = now
	}

	b := rl.buckets[key]
	if b == nil || now.Sub(b.start) >= rl.window {
		b = &bucket{start: now}
		rl.buckets[key] = b
	}
	if b.count >= rl.limit {
		return false, rl.window - now.Sub(b.start)
	}
	b.count++
	return true, 0
}

// Handler rejects requests over the limit with 429 and a Retry-After header.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retryAfter := rl.Allow(clientIP(r))
		if !ok {
			seconds := int(retryAfter.Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeError(w, http.StatusTooManyRequests, "rate_limited",
				"Trop de tentatives, réessayez dans "+strconv.Itoa(seconds)+" s")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the host part of RemoteAddr (chi's RealIP middleware, when
// enabled, has already rewritten it from X-Forwarded-For / X-Real-IP).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
