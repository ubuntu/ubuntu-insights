// Package workers provides worker management for the ingest service.
package workers

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Pool is a struct that holds the worker management logic.
type Pool struct {
	cm   dConfigManager
	proc dProcessor

	mu       sync.Mutex
	workers  map[string]context.CancelFunc
	workerWG sync.WaitGroup

	metricsMu     sync.Mutex
	activeWorkers prometheus.Gauge
}

type dConfigManager interface {
	Watch(context.Context) (<-chan struct{}, <-chan error, error)
	AllowList() []string
	IsAllowed(string) bool
}

type dProcessor interface {
	Process(ctx context.Context, app string) error
}

// New creates a new worker pool instance with the provided config manager, processor, and Prometheus registerer.
func New(cm dConfigManager, proc dProcessor, reg prometheus.Registerer) (*Pool, error) {
	activeWorkers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ingest_active_workers",
		Help: "Number of active workers in the ingest service.",
	})
	if err := reg.Register(activeWorkers); err != nil {
		return nil, fmt.Errorf("failed to register active workers gauge: %v", err)
	}

	return &Pool{
		cm:            cm,
		proc:          proc,
		workers:       make(map[string]context.CancelFunc),
		activeWorkers: activeWorkers,
	}, nil
}

// Run orchestrates and manages the pool of workers.
//
// It watches the configured location for new reports that have been saved to disk.
// It will then process, validate, and then upload the reports to the database.
//
// This is blocking until an error occurs or the context is canceled and all workers are done.
//
// Always returns a non-nil error, which is either a context error or a processing error.
func (m *Pool) Run(ctx context.Context) error {
	slog.Info("Ingest service started")

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reloadEventCh, cfgWatchErrCh, err := m.cm.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start watch configuration: %v", err)
	}

	// Initial sync
	m.syncWorkers(ctx)

	// Debounce timer for handling bursts of events
	debounceDuration := 5 * time.Second
	debounceTimer := time.NewTimer(debounceDuration)
	defer debounceTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Context canceled, stopping worker pool")
			m.workerWG.Wait()
			return ctx.Err()

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
			m.syncWorkers(ctx)
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
func (m *Pool) syncWorkers(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// stop removed
	for app, cancel := range m.workers {
		if !m.cm.IsAllowed(app) {
			cancel()
			delete(m.workers, app)
		}
	}
	// start added
	for _, app := range m.cm.AllowList() {
		if _, ok := m.workers[app]; ok {
			continue
		}

		select {
		case <-ctx.Done():
			slog.Info("Context canceled, stopping worker sync")
			return // normal shutdown
		default:
		}
		appCtx, cancel := context.WithCancel(ctx)
		m.workers[app] = cancel
		slog.Info("Starting app worker", "app", app)
		m.workerWG.Add(1)
		go m.appWorker(appCtx, app)
	}
}

// appWorker watches & processes files for a single app until ctx is canceled.
func (m *Pool) appWorker(ctx context.Context, app string) {
	defer m.workerWG.Done()

	m.metricsMu.Lock()
	m.activeWorkers.Inc()
	m.metricsMu.Unlock()

	defer func() {
		m.metricsMu.Lock()
		m.activeWorkers.Dec()
		m.metricsMu.Unlock()
	}()

	baseBackoff := 5 * time.Second
	maxBackoff := 30 * time.Second
	backoff := baseBackoff

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// this will read/process/remove JSON files and call s.db.Upload(...)
			err := m.proc.Process(ctx, app)
			if err == nil {
				backoff = baseBackoff
				continue
			}

			// #nosec:G404 We don't need cryptographic randomness.
			sleep := time.Duration(rand.Int63n(int64(backoff)))
			select {
			case <-time.After(sleep):
			case <-ctx.Done():
				slog.Debug("App worker context canceled", "app", app)
				return // normal shutdown
			}

			backoff = min(backoff*2, maxBackoff)
		}
	}
}
