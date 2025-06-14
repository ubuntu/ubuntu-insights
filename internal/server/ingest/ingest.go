// Package ingest is responsible for running the ingest-service in the background.
package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/processor"
)

// Service represents the ingest service that processes and uploads reports to the database.
type Service struct {
	cm         dConfigManager
	db         dbManager
	reportsDir string
	invalidDir string

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

// StaticConfig holds the static configuration for the service.
type StaticConfig struct {
	ReportsDir string
	InvalidDir string
}

type dbManager interface {
	Upload(ctx context.Context, app string, data *models.TargetModel) error
	UploadLegacy(ctx context.Context, distribution, version string, report *models.LegacyTargetModel) error
	Close() error
}

type dConfigManager interface {
	Load() error
	Watch(context.Context) (<-chan struct{}, <-chan error, error)
	AllowList() []string
}

type options struct {
	dbConnect func(ctx context.Context, cfg database.Config) (dbManager, error)
}

// Options is a function that modifies the options for the ingest service.
type Options func(*options)

// New creates a new ingest service with the provided database manager and connects to the database.
func New(ctx context.Context, cm dConfigManager, dbConfig database.Config, sc StaticConfig, args ...Options) (*Service, error) {
	opts := options{
		dbConnect: func(ctx context.Context, cfg database.Config) (dbManager, error) {
			return database.Connect(ctx, cfg)
		},
	}

	for _, opt := range args {
		opt(&opts)
	}

	if sc.ReportsDir == "" {
		return nil, fmt.Errorf("reportsDir must be set")
	}

	if sc.InvalidDir == "" {
		return nil, fmt.Errorf("invalidDir must be set")
	}

	if sc.InvalidDir == sc.ReportsDir {
		return nil, fmt.Errorf("invalidDir cannot be the same as reportsDir")
	}

	if err := os.MkdirAll(sc.InvalidDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create invalidDir: %v", err)
	}

	if err := os.MkdirAll(sc.ReportsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create reportsDir: %v", err)
	}

	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	dctx, dcancel := context.WithTimeout(ctx, 10*time.Second)
	defer dcancel()
	db, err := opts.dbConnect(dctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	gCtx, gCancel := context.WithCancel(ctx)

	return &Service{
		cm:         cm,
		db:         db,
		reportsDir: sc.ReportsDir,
		invalidDir: sc.InvalidDir,

		ctx:            ctx,
		cancel:         cancel,
		gracefulCtx:    gCtx,
		gracefulCancel: gCancel,

		mu:       sync.Mutex{},
		workers:  make(map[string]context.CancelFunc),
		workerWG: sync.WaitGroup{},
	}, nil
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
	allowed := s.cm.AllowList()
	s.mu.Lock()
	defer s.mu.Unlock()

	want := map[string]struct{}{}
	for _, app := range allowed {
		want[app] = struct{}{}
	}

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

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// this will read/process/remove JSON files and call s.db.Upload(...)
			p := processor.New(s.reportsDir, s.invalidDir, s.db)
			if err := p.Process(ctx, app); err != nil {
				if errors.Is(err, context.Canceled) {
					slog.Debug("App worker stopped", "app", app)
					return // normal shutdown
				}
				slog.Error("Failed to process files", "app", app, "err", err)
				return
			}
		}
	}
}

// Quit stops the ingest service and closes the database connection.
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

	if s.db != nil {
		s.db.Close()
	}
}
