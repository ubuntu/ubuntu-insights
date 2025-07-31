// Package ingest is responsible for running the ingest-service in the background.
package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Service represents the ingest service that processes and uploads reports to the database.
type Service struct {
	workerPool    WorkerPool
	metricsServer MetricsServer

	// This context is used to interrupt any action.
	// It must be the parent of gracefulCtx.
	ctx    context.Context
	cancel context.CancelFunc

	// This context waits until the next blocking Recv to interrupt.
	gracefulCtx    context.Context
	gracefulCancel context.CancelFunc

	maxDegradedDuration time.Duration

	running chan struct{} // Channel to signal when the service is running.
}

// WorkerPool is an interface that defines the methods for a worker pool.
type WorkerPool interface {
	Run(ctx context.Context) error
}

// MetricsServer is an interface that defines the methods for a metrics server.
type MetricsServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
	Close() error
}

type options struct {
	maxDegradedDuration time.Duration
}

// Option is a function which tweaks the creation of the Service.
type Option func(*options)

var (
	// errServiceClosed is returned when the service is already closed.
	errServiceClosed = errors.New("service closed")

	// ErrTeardownTimeout is returned when the service takes too long to shut down.
	// A force Quit may be required to cleanup the service.
	ErrTeardownTimeout = errors.New("service teardown timed out")
)

// New creates a new ingest service with the provided config manager and processor.
func New(ctx context.Context, workerPool WorkerPool, metricsServer MetricsServer, args ...Option) *Service {
	ctx, cancel := context.WithCancel(ctx)
	gCtx, gCancel := context.WithCancel(ctx)

	opts := options{
		maxDegradedDuration: 2 * time.Minute, // Default degraded state duration
	}
	for _, arg := range args {
		arg(&opts)
	}

	running := make(chan struct{})
	close(running) // Close immediately to avoid blocking on the channel.
	return &Service{
		workerPool:    workerPool,
		metricsServer: metricsServer,

		ctx:            ctx,
		cancel:         cancel,
		gracefulCtx:    gCtx,
		gracefulCancel: gCancel,

		maxDegradedDuration: opts.maxDegradedDuration,

		running: running,
	}
}

// Run starts the ingest service.
//
// Returns once all sub-services have completed, or after an extended time being in a degraded state.
func (s *Service) Run() error {
	slog.Info("Ingest service started")

	select {
	case <-s.gracefulCtx.Done():
		return errServiceClosed
	default:
	}

	s.running = make(chan struct{})
	defer close(s.running)
	defer s.cancel() // Ensure we cancel the context when done, regardless of result.

	done := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { done <- s.runWorkers(); wg.Done() }()
	go func() { done <- s.runMetrics(); wg.Done() }()
	go func() { wg.Wait(); close(done) }() // Close done only after both goroutines have finished.

	// Ensure we don't get stuck in a degraded state if one of the services fails.
	err := <-done
	slog.Info("Waiting for ingest services to finish")

	select {
	case <-time.After(s.maxDegradedDuration):
		// We've waited for teardown for too long, give up even though errors may be lost.
		slog.Warn("Ingest service teardown timed out")
		err = errors.Join(err, ErrTeardownTimeout)
	case secondDone := <-done:
		err = errors.Join(err, secondDone)
	}

	return err
}

func (s *Service) runWorkers() error {
	slog.Info("Starting worker pool")
	defer s.gracefulCancel() // Request stop if workers fail.

	if err := s.workerPool.Run(s.gracefulCtx); err != nil && !errors.Is(err, s.gracefulCtx.Err()) {
		slog.Error("Worker pool encountered an error", "err", err)
		return fmt.Errorf("ingest workers error: %v", err)
	}
	slog.Info("Workers stopped")
	return nil
}

func (s *Service) runMetrics() error {
	slog.Info("Starting metrics server")
	defer s.gracefulCancel() // Request stop if metrics fail.

	metricsErrCh := make(chan error, 1)
	go func() {
		defer close(metricsErrCh)
		if err := s.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			metricsErrCh <- err
		}
	}()

	select {
	case <-s.ctx.Done():
		slog.Info("Closing metrics server", "reason", s.ctx.Err())
		s.metricsServer.Close()
		return nil
	case <-s.gracefulCtx.Done():
		slog.Info("Graceful shutdown initiated for metrics server")
		if err := s.metricsServer.Shutdown(s.ctx); err != nil {
			slog.Error("Metrics server graceful shutdown encountered error", "err", err)
			return fmt.Errorf("metrics server shutdown error: %v", err)
		}
	case err := <-metricsErrCh:
		// No need to shutdown or close, just propagate the error.
		if err != nil {
			slog.Error("Metrics server encountered error", "err", err)
			return fmt.Errorf("metrics server error: %v", err)
		}
	}
	slog.Info("Metrics server shut down gracefully")
	return nil
}

// Quit stops the ingest service.
// Blocks until the service has finished running.
func (s *Service) Quit(force bool) {
	slog.Info("Stopping Ingest service")

	if force {
		s.cancel()
		s.metricsServer.Close()
	} else {
		s.gracefulCancel()
	}

	<-s.running // Wait for the service to finish running.
}
