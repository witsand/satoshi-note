package main

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a per-IP token-bucket rate limiter with automatic stale-entry cleanup.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	burst    int
}

func newRateLimiter(r rate.Limit, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		burst:    burst,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if entry, ok := rl.limiters[ip]; ok {
		entry.lastSeen = time.Now()
		return entry.limiter
	}
	l := rate.NewLimiter(rl.r, rl.burst)
	rl.limiters[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
	return l
}

// cleanupLoop removes limiters that have been idle for 10 minutes.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware wraps a handler, rejecting requests that exceed the rate limit.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.getLimiter(clientIP(r)).Allow() {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"status": "ERROR",
				"reason": "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from the request.
//
// Header priority:
//  1. CF-Connecting-IP — set by Cloudflare; cannot be forged by clients when
//     running behind a Cloudflare Tunnel or proxied through Cloudflare.
//  2. X-Real-IP — set by nginx/caddy when configured with
//     `proxy_set_header X-Real-IP $remote_addr` (or equivalent).
//  3. RemoteAddr — used when running without a reverse proxy (local/dev).
//
// X-Forwarded-For is intentionally not used: its leftmost value is
// client-controlled and can be trivially spoofed to bypass rate limiting.
// Operators running behind a non-Cloudflare proxy that does not set X-Real-IP
// should configure it to do so.
func clientIP(r *http.Request) string {
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		return cf
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	// RemoteAddr is host:port — strip the port.
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i]
	}
	return addr
}
