// Package webservice provides an HTTP server that handles incoming requests for uploading data and retrieving version information.
package webservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/handlers"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

// Server is a struct that holds the HTTP server and its configuration.
type Server struct {
	httpServer    *http.Server
	metricsServer *http.Server
	cm            dConfigManager

	primaryAddr net.Addr
	metricsAddr net.Addr

	// This context is used to interrupt any action.
	// It must be the parent of gracefulCtx.
	ctx    context.Context
	cancel context.CancelFunc

	// This context waits until the next blocking Recv to interrupt.
	gracefulCtx    context.Context
	gracefulCancel context.CancelFunc

	mu sync.RWMutex
}

// StaticConfig holds the static configuration for the server.
type StaticConfig struct {
	ReportsDir string

	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	RequestTimeout time.Duration
	MaxHeaderBytes int
	MaxUploadBytes int

	ListenHost string
	ListenPort int

	MetricsHost string
	MetricsPort int
}

type dConfigManager interface {
	Load() error
	Watch(context.Context) (<-chan struct{}, <-chan error, error)
	IsAllowed(string) bool
}

// New creates a new Server instance with the given http.Server and config.ConfigManager.
func New(ctx context.Context, cm dConfigManager, sc StaticConfig) (*Server, error) {
	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	gCtx, gCancel := context.WithCancel(ctx)

	s := Server{
		cm:     cm,
		ctx:    ctx,
		cancel: cancel,

		gracefulCtx:    gCtx,
		gracefulCancel: gCancel}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", sc.ListenHost, sc.ListenPort),
		ReadTimeout:    sc.ReadTimeout,
		WriteTimeout:   sc.WriteTimeout,
		Handler:        http.TimeoutHandler(setupPrimaryMux(cm, sc, registry), sc.RequestTimeout, ""),
		MaxHeaderBytes: sc.MaxHeaderBytes,
	}

	s.metricsServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", sc.MetricsHost, sc.MetricsPort),
		ReadTimeout:  sc.ReadTimeout,
		WriteTimeout: sc.WriteTimeout,
		Handler:      setupMetricsMux(registry),
	}

	return &s, nil
}

func setupPrimaryMux(cm dConfigManager, sc StaticConfig, registry *prometheus.Registry) http.Handler {
	endpointMW := metrics.NewEndpointMiddleware(registry)
	muxMW := metrics.NewMuxMiddleware(registry)

	mux := http.NewServeMux()
	uploadHandler := handlers.NewUpload(cm, sc.ReportsDir, int64(sc.MaxUploadBytes))
	legacyUploadHandler := handlers.NewLegacyReport(cm, sc.ReportsDir, int64(sc.MaxUploadBytes))

	mux.Handle("POST /upload/{app}", endpointMW.Wrap("upload", uploadHandler))
	mux.Handle("POST /{distribution}/desktop/{version}", endpointMW.Wrap("legacy_upload", legacyUploadHandler))
	mux.Handle("GET /version", endpointMW.Wrap("version", http.HandlerFunc(handlers.VersionHandler)))

	return muxMW.Wrap("primary_mux", mux)
}

func setupMetricsMux(registry *prometheus.Registry) *http.ServeMux {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return metricsMux
}

// Run starts the HTTP server and listens for incoming requests.
func (s *Server) Run() error {
	primaryErr := make(chan error, 1)
	metricsErr := make(chan error, 1)

	go func() {
		defer close(primaryErr)
		primaryErr <- s.servePrimary()
	}()
	go func() {
		defer close(metricsErr)
		metricsErr <- s.serveMetrics()
	}()

	// One server shutting down will shut down the other.
	errPrimary := <-primaryErr
	errMetrics := <-metricsErr

	if errPrimary != nil {
		errPrimary = fmt.Errorf("primary server error: %w", errPrimary)
	}
	if errMetrics != nil {
		errMetrics = fmt.Errorf("metrics server error: %w", errMetrics)
	}

	return errors.Join(errPrimary, errMetrics)
}

// servePrimary starts the primary HTTP server and listens for incoming requests.
func (s *Server) servePrimary() error {
	slog.Info("Starting server", "addr", s.httpServer.Addr)

	// already asked to quit?
	select {
	case <-s.gracefulCtx.Done():
		return s.gracefulCtx.Err()
	default:
	}

	defer s.cancel()

	_, watchErr, err := s.cm.Watch(s.gracefulCtx)
	if err != nil {
		return fmt.Errorf("failed to start watching configuration: %v", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		l, err := net.Listen("tcp", s.httpServer.Addr)
		if err != nil {
			serverErr <- err
			return
		}
		s.mu.Lock()
		s.primaryAddr = l.Addr()
		s.mu.Unlock()
		// Listener lifecycle is managed by the server.
		if err := s.httpServer.Serve(l); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-s.gracefulCtx.Done():
		slog.Info("Graceful shutdown initiated")
		// use parent ctx so if you call s.cancel() elsewhere it unblocks Shutdown immediately
		if err := s.httpServer.Shutdown(s.ctx); err != nil {
			return fmt.Errorf("failed to gracefully shutdown server: %v", err)
		}
		slog.Info("Server shut down gracefully")
		return nil

	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("primary HTTP server encountered an error: %v", err)
		}
		return nil
	case err := <-watchErr:
		if err != nil {
			err = fmt.Errorf("config watcher encountered an unrecoverable error: %v", err)
		}
		errC := s.httpServer.Close()

		return errors.Join(err, errC)
	}
}

// serveMetrics starts the metrics HTTP server and listens for incoming requests.
func (s *Server) serveMetrics() error {
	slog.Info("Starting metrics server", "addr", s.metricsServer.Addr)

	defer s.cancel()

	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		l, err := net.Listen("tcp", s.metricsServer.Addr)
		if err != nil {
			serverErr <- err
			return
		}
		// Listener lifecycle is managed by the server.
		s.mu.Lock()
		s.metricsAddr = l.Addr()
		s.mu.Unlock()
		if err := s.metricsServer.Serve(l); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-s.gracefulCtx.Done():
		slog.Info("Graceful shutdown initiated for metrics server")
		if err := s.metricsServer.Shutdown(s.ctx); err != nil {
			return fmt.Errorf("failed to gracefully shutdown server: %v", err)
		}
		slog.Info("Metrics server shut down gracefully")
		return nil
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("metrics server encountered an error: %v", err)
		}
		return nil
	}
}

// Quit shuts down the HTTP server gracefully.
func (s *Server) Quit(force bool) {
	defer s.cancel()

	if force {
		s.httpServer.Close()
		s.metricsServer.Close()
		s.cancel()
	} else {
		s.gracefulCancel()
	}
	slog.Info("Server quit")
}
