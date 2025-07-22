// Package ingest is responsible for running the ingest-service in the background.
package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// Service represents the ingest service that processes and uploads reports to the database.
type Service struct {
	cm   dConfigManager
	proc dProcessor

	// This context is used to interrupt any action.
	// It must be the parent of gracefulCtx.
	ctx    context.Context
	cancel context.CancelFunc

	// This context waits until the next blocking Recv to interrupt.
	gracefulCtx    context.Context
	gracefulCancel context.CancelFunc

	mu       sync.Mutex
	workers  map[string]context.CancelFunc
	workerWG sync.WaitGroup
}

type dConfigManager interface {
	Watch(context.Context) (<-chan struct{}, <-chan error, error)
	AllowSet() map[string]struct{}
}

type dProcessor interface {
	Process(ctx context.Context, app string) error
}

// New creates a new ingest service with the provided config manager and processor.
func New(ctx context.Context, cm dConfigManager, proc dProcessor) *Service {
	ctx, cancel := context.WithCancel(ctx)
	gCtx, gCancel := context.WithCancel(ctx)

	return &Service{
		cm:   cm,
		proc: proc,

		ctx:            ctx,
		cancel:         cancel,
		gracefulCtx:    gCtx,
		gracefulCancel: gCancel,

		mu:       sync.Mutex{},
		workers:  make(map[string]context.CancelFunc),
		workerWG: sync.WaitGroup{},
	}
}

// Run starts the ingest service.
//
// It watches the configured location for new reports that have been saved to disk.
// It will then process, validate, and then upload the reports to the database.
func (s *Service) Run() error {
	slog.Info("Ingest service started")

	select {
	case <-s.gracefulCtx.Done():
		return errors.New("server is already shutting down")
	default:
	}

	reloadEventCh, cfgWatchErrCh, err := s.cm.Watch(s.gracefulCtx)
	if err != nil {
		return fmt.Errorf("failed to start watch configuration: %v", err)
	}

	// Initial sync
	s.syncWorkers()

	// Debounce timer for handling bursts of events
	debounceDuration := 5 * time.Second
	debounceTimer := time.NewTimer(debounceDuration)
	defer debounceTimer.Stop()

	for {
		select {
		case <-s.gracefulCtx.Done():
			slog.Info("Ingest service stopped")
			return nil

		case _, ok := <-reloadEventCh:
			if !ok {
				return fmt.Errorf("reloadEventCh closed unexpectedly")
			}
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(debounceDuration)

		case <-debounceTimer.C:
			// Timer expired, perform the resync
			slog.Info("Resyncing workers after configuration change")
			s.syncWorkers()
			slog.Debug("Completed resyncing workers")

		case err, ok := <-cfgWatchErrCh:
			if !ok {
				return fmt.Errorf("cfgWatchErrCh closed unexpectedly")
			}
			if err != nil {
				slog.Error("Configuration watcher error", "err", err)
			}
		}
	}
}

// syncWorkers diffs the allow‐list and starts/stops goroutines.
func (s *Service) syncWorkers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	want := s.cm.AllowSet()

	// stop removed
	for app, cancel := range s.workers {
		if _, ok := want[app]; !ok {
			cancel()
			delete(s.workers, app)
		}
	}
	// start added
	for app := range want {
		if _, ok := s.workers[app]; ok {
			continue
		}

		select {
		case <-s.gracefulCtx.Done():
			slog.Info("Graceful shutdown in progress, stopping app worker", "app", app)
			return // normal shutdown
		default:
		}
		ctx, cancel := context.WithCancel(s.gracefulCtx)
		s.workers[app] = cancel
		slog.Info("Starting app worker", "app", app)
		go s.appWorker(ctx, app)
	}
}

// appWorker watches & processes files for a single app until ctx is canceled.
func (s *Service) appWorker(ctx context.Context, app string) {
	s.workerWG.Add(1)
	defer s.workerWG.Done()

	baseBackoff := 5 * time.Second
	maxBackoff := 30 * time.Second
	backoff := baseBackoff

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// this will read/process/remove JSON files and call s.db.Upload(...)
			err := s.proc.Process(ctx, app)
			if err == nil {
				backoff = baseBackoff
				continue
			}
			if errors.Is(err, context.Canceled) {
				slog.Debug("App worker stopped", "app", app)
				return // normal shutdown
			}

			// #nosec:G404 We don't need cryptographic randomness.
			sleep := time.Duration(rand.Int63n(int64(backoff)))
			select {
			case <-time.After(sleep):
			case <-ctx.Done():
				slog.Debug("App worker stopped during backoff", "app", app)
				return // normal shutdown
			}

			backoff = min(backoff*2, maxBackoff)
		}
	}
}

// Quit stops the ingest service.
// If force is false, it will block until all workers close.
//
// Safe to call multiple times.
func (s *Service) Quit(force bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("Stopping Ingest service")
	if force {
		s.cancel()
	} else {
		s.gracefulCancel()
		s.workerWG.Wait()
	}
}
