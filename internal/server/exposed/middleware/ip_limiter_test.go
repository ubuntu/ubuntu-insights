package middleware_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/internal/server/exposed/middleware"
	"golang.org/x/time/rate"
)

func makeRequestWithIP(handler http.Handler, port string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = net.JoinHostPort("1.2.3.4", port)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	return rr
}

func TestLimiter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		limiter     *middleware.IPLimiter
		handlerFunc http.HandlerFunc
		port1       string
		port2       string

		status1 int
		status2 int
	}{
		"Under limit OK": {
			limiter: middleware.New(rate.Every(time.Second), 2),
			port1:   "8080",
			port2:   "8080",
		},
		"Blocks over limit": {
			limiter: middleware.New(rate.Every(time.Second), 1),
			port1:   "8081",
			port2:   "8081",
			status2: http.StatusTooManyRequests,
		},
		"Different ports have independent limits": {
			limiter: middleware.New(rate.Every(time.Second), 1),
			port1:   "8082",
			port2:   "8083",
			status2: http.StatusTooManyRequests,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.status1 == 0 {
				tc.status1 = http.StatusOK
			}
			if tc.status2 == 0 {
				tc.status2 = http.StatusOK
			}

			if tc.handlerFunc == nil {
				tc.handlerFunc = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}
			}

			handler := tc.limiter.RateLimitMiddleware(tc.handlerFunc)

			rr1 := makeRequestWithIP(handler, tc.port1)
			rr2 := makeRequestWithIP(handler, tc.port2)

			assert.Equal(t, tc.status1, rr1.Code)
			assert.Equal(t, tc.status2, rr2.Code)
		})
	}
}

func TestLimiter_InvalidRemoteAddr(t *testing.T) {
	t.Parallel()
	limiter := middleware.New(rate.Every(time.Second), 1)
	handler := limiter.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for bad IP")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "invalid-ip" // not in host:port format
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
