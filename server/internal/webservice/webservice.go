// Package webservice provides an HTTP server that handles incoming requests for uploading data and retrieving version information.
package webservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/handlers"
)

// Server is a struct that holds the HTTP server and its configuration.
type Server struct {
	httpServer *http.Server
	cm         dConfigManager

	// This context is used to interrupt any action.
	// It must be the parent of gracefulCtx.
	ctx    context.Context
	cancel context.CancelFunc

	// This context waits until the next blocking Recv to interrupt.
	gracefulCtx    context.Context
	gracefulCancel context.CancelFunc
}

// StaticConfig holds the static configuration for the server.
type StaticConfig struct {
	ConfigPath string
	ReportsDir string

	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	RequestTimeout time.Duration
	MaxHeaderBytes int
	MaxUploadBytes int

	ListenHost string
	ListenPort int
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

	uploadHandler := handlers.NewUpload(cm, sc.ReportsDir, int64(sc.MaxUploadBytes))
	legacyUploadHandler := handlers.NewLegacyReport(cm, sc.ReportsDir, int64(sc.MaxUploadBytes))
	mux := http.NewServeMux()
	mux.Handle("POST /upload/{app}", uploadHandler)
	mux.Handle("POST /{distribution}/desktop/{version}", legacyUploadHandler)
	mux.Handle("GET /version", http.HandlerFunc(handlers.VersionHandler))

	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", sc.ListenHost, sc.ListenPort),
		ReadTimeout:    sc.ReadTimeout,
		WriteTimeout:   sc.WriteTimeout,
		Handler:        http.TimeoutHandler(mux, sc.RequestTimeout, ""),
		MaxHeaderBytes: sc.MaxHeaderBytes,
	}

	return &s, nil
}

// Run starts the HTTP server and listens for incoming requests.
func (s *Server) Run() error {
	slog.Info("Starting server", "addr", s.httpServer.Addr)

	// already asked to quit?
	select {
	case <-s.gracefulCtx.Done():
		return errors.New("server is already shutting down")
	default:
	}

	_, watchErr, err := s.cm.Watch(s.gracefulCtx)
	if err != nil {
		return fmt.Errorf("failed to start watching configuration: %v", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-s.gracefulCtx.Done():
		slog.Info("Graceful shutdown initiated")
		// use parent ctx so if you call s.cancel() elsewhere it unblocks Shutdown immediately
		if err := s.httpServer.Shutdown(s.ctx); err != nil {
			slog.Error("Graceful shutdown failed", "err", err)
			return err
		}
		slog.Info("Server shut down gracefully")
		// now kill everything else (watchers, handlers, etc.)
		s.cancel()
		return nil

	case err := <-serverErr:
		if err != nil {
			slog.Error("Server encountered error", "err", err)
			s.cancel()
			return err
		}
		// unlikely: ListenAndServe returned nil
		s.cancel()
		return nil
	case err := <-watchErr:
		if err != nil {
			slog.Error("Config watcher encountered unrecoverable error", "err", err)
		}
		errC := s.httpServer.Close()
		s.cancel()

		return errors.Join(err, errC)
	}
}

// Quit shuts down the HTTP server gracefully.
func (s *Server) Quit(force bool) {
	defer s.cancel()

	if force {
		s.httpServer.Close()
		s.cancel()
	} else {
		s.gracefulCancel()
	}
	slog.Info("Server quit")
}
