// Package middleware provides middleware functions for HTTP servers.
package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// IPLimiter is a middleware that limits the number of requests from a single IP address.
type IPLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

// New creates a new IPLimiter with the specified rate limit and burst size.
func New(r rate.Limit, b int) *IPLimiter {
	return &IPLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

func (l *IPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[ip] = limiter
	}
	return limiter
}

// RateLimitMiddleware is the middleware function that applies the rate limiting.
func (l *IPLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Unable to determine IP", http.StatusBadRequest)
			return
		}
		if !l.getLimiter(ip).Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
