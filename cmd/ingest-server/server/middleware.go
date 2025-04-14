package server

import (
	"net/http"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
}

func newIPLimiter(r rate.Limit, b int) *ipLimiter {
	return &ipLimiter{}
}

func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	return nil
}

func (h Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		limiter := h.ipLimiter.getLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
