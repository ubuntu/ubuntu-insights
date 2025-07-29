// Package metrics provides a Prometheus metrics HTTP server.
package metrics

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server is a struct that holds the HTTP server and its configuration.
type Server struct {
	reg prometheus.Gatherer

	addr       net.Addr
	httpServer *http.Server

	mu sync.RWMutex
}

// Config holds the configuration for the metrics server.
type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// New creates a new metrics manager with the provided registry and host/port.
func New(cfg Config, reg prometheus.Gatherer) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	return &Server{
		reg: reg,
		httpServer: &http.Server{
			Addr:         net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
			Handler:      mux,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
}

// ListenAndServe starts the HTTP server and listens for incoming requests.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.addr = listener.Addr()
	s.mu.Unlock()

	return s.httpServer.Serve(listener)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close stops the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// Addr returns the address the server is listening on.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.addr == nil {
		return ""
	}
	return s.addr.String()
}
