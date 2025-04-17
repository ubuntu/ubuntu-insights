package middleware_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/middleware"
	"golang.org/x/time/rate"
)

func makeRequestWithIP(handler http.Handler, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = net.JoinHostPort(ip, "12345")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	t.Parallel()
	limiter := middleware.New(rate.Every(time.Second), 2) // 2 requests burst
	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr1 := makeRequestWithIP(handler, "1.2.3.4")
	rr2 := makeRequestWithIP(handler, "1.2.3.4")

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatal("Expected both requests to succeed")
	}
}

func TestLimiter_BlocksRequestsOverLimit(t *testing.T) {
	t.Parallel()
	limiter := middleware.New(rate.Every(10*time.Second), 1) // Only 1 allowed every 10s
	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr1 := makeRequestWithIP(handler, "5.6.7.8")
	rr2 := makeRequestWithIP(handler, "5.6.7.8")

	if rr1.Code != http.StatusOK {
		t.Fatal("Expected first request to succeed")
	}
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatal("Expected second request to be rate-limited")
	}
}

func TestLimiter_SeparateLimitsForDifferentIPs(t *testing.T) {
	t.Parallel()
	limiter := middleware.New(rate.Every(10*time.Second), 1)
	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr1 := makeRequestWithIP(handler, "1.1.1.1")
	rr2 := makeRequestWithIP(handler, "2.2.2.2")

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatal("Expected both IPs to have separate rate limits")
	}
}

func TestLimiter_InvalidRemoteAddr(t *testing.T) {
	t.Parallel()
	limiter := middleware.New(rate.Every(time.Second), 1)
	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for bad IP")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "invalid-ip" // not in host:port format
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}
