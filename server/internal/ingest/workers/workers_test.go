package workers_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/workers"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cm   *mockConfigManager
		proc *mockDProcessor

		skipMetricsCheck bool
		wantErr          bool
	}{
		"Empty allow list": {},
		"Single app no errors": {
			cm: newConfigManager("SingleValid"),
		},
		"Multi apps no errors": {
			cm: newConfigManager("MultiValid1", "MultiValid2", "MultiValid3"),
		},

		// Processor errors
		"Single app with context canceled": {
			cm: newConfigManager("SingleValid"),
			proc: newProcessor(map[string]error{
				"SingleValid": context.Canceled,
			}),
			skipMetricsCheck: true,
		},
		"Single app with error": {
			cm: newConfigManager("SingleValid"),
			proc: newProcessor(map[string]error{
				"SingleValid": errors.New("requested error"),
			}),
		},
		"Multi apps with context canceled": {
			cm: newConfigManager("MultiValid1", "MultiValid2", "MultiValid3"),
			proc: newProcessor(map[string]error{
				"MultiValid1": context.Canceled,
				"MultiValid2": context.Canceled,
			}),
			skipMetricsCheck: true,
		},
		"Multi apps with errors": {
			cm: newConfigManager("MultiValid1", "MultiValid2", "MultiValid3"),
			proc: newProcessor(map[string]error{
				"MultiValid1": errors.New("error for MultiValid1"),
				"MultiValid2": errors.New("error for MultiValid2"),
			}),
		},

		// Config manager errors
		"Exits on config manager reloadCh early close": {
			cm: &mockConfigManager{
				allowList:     []string{"SingleValid"},
				closeReloadCh: true,
			},
			wantErr: true,
		},
		"Exits on config manager watchErrCh early close": {
			cm: &mockConfigManager{
				allowList:     []string{"SingleValid"},
				closeWatchErr: true,
			},
			wantErr: true,
		},
		"Exits on config manager watch error": {
			cm: &mockConfigManager{
				allowList: []string{"SingleValid"},
				watchErr:  errors.New("watch error"),
			},
			wantErr: true,
		},
		"Does not exit on config manager delayed watch error": {
			cm: &mockConfigManager{
				allowList:       []string{"SingleValid"},
				delayedWatchErr: errors.New("delayed watch error"),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.cm == nil {
				tc.cm = newConfigManager()
			}

			// Apply the allowSet if not already set
			if tc.cm.allowSet == nil {
				tc.cm.allowSet = createSet(tc.cm.allowList...)
			}

			if tc.proc == nil {
				tc.proc = newProcessor(map[string]error{})
			}

			registry := prometheus.NewRegistry()
			s, err := workers.New(tc.cm, tc.proc, registry)
			require.NoError(t, err, "Setup: Failed to create worker pool")
			runErr := run(t.Context(), t, s)

			if tc.wantErr {
				checkService(t, runErr, true, 3*time.Second)
				return
			}

			var collector prometheus.Collector
			if !tc.skipMetricsCheck {
				collector = registry
			}
			waitWorkersEqual(t, s, collector, tc.cm.AllowList()...)
			// Ensure no errors are received
			checkService(t, runErr, false, 0)
		})
	}
}

// Tests the addition and removal of apps from the allow list
// and verifies that the service updates its workers accordingly.
func TestRunModifyAllowList(t *testing.T) {
	t.Parallel()

	cm := newConfigManager("SingleValid")
	registry := prometheus.NewRegistry()
	s, err := workers.New(cm, &mockDProcessor{}, registry)
	require.NoError(t, err, "Setup: Failed to create worker pool")
	run(t.Context(), t, s)

	waitWorkersEqual(t, s, registry, cm.AllowList()...)

	cm.setAllowList(t, append(cm.AllowList(), "MultiMixed"), 3)
	waitWorkersEqual(t, s, registry, cm.AllowList()...)

	cm.setAllowList(t, []string{}, 3)
	waitWorkersEqual(t, s, registry)
}

func TestRunEarlyContextCancel(t *testing.T) {
	t.Parallel()
	cm := newConfigManager("MultiValid1", "MultiValid2", "MultiValid3")
	proc := newProcessor(map[string]error{
		"SingleValid": context.Canceled,
	})

	ctx, cancel := context.WithCancel(t.Context())
	s, err := workers.New(cm, proc, prometheus.NewRegistry())
	require.NoError(t, err, "Setup: Failed to create worker pool")
	runErr := run(ctx, t, s)

	// Ensure no errors are received before the context is canceled
	checkService(t, runErr, false, 50*time.Millisecond)

	cancel()

	// Ensure that the service exited within a reasonable time
	select {
	case err := <-runErr:
		require.ErrorIs(t, err, ctx.Err(), "Expected context error after context cancellation")
	case <-time.After(3 * time.Second):
		require.Fail(t, "Service did not exit after context cancellation")
	}
}

// checkService is a helper function which waits a specified duration, unless an error signal is received.
func checkService(t *testing.T, runErr chan error, expectErr bool, duration time.Duration) {
	t.Helper()

	select {
	case err := <-runErr:
		if expectErr {
			require.Error(t, err, "Expected error but got nil")
			return
		}
		// Unexpected early close
		require.Fail(t, "Service closed unexpectedly", err)
	case <-time.After(duration):
		require.False(t, expectErr, "Service did not exit with an error within the expected duration")
	}
}

// waitWorkersEqual is a helper function which waits until the active workers in the service match the expected workers.
// It also checks the registry gauge if provided.
func waitWorkersEqual(t *testing.T, m *workers.Pool, registry prometheus.Collector, workers ...string) {
	t.Helper()
	delay := 500 * time.Millisecond
	timeout := 8 * time.Second

	start := time.Now()
	for {
		got := m.WorkerNames()

		slices.Sort(got)
		slices.Sort(workers)

		if slices.Equal(workers, got) {
			if registry == nil || len(workers) == int(testutil.ToFloat64(registry)) {
				return
			}
		}
		require.LessOrEqual(t, time.Since(start), timeout, "Workers did not match within the timeout. Wanted: %v, Got: %v", workers, got)
		time.Sleep(delay)
	}
}

type mockConfigManager struct {
	allowList []string
	allowSet  map[string]struct{}

	closeReloadCh   bool
	closeWatchErr   bool
	watchErr        error
	delayedWatchErr error

	reloadCh chan struct{}
	errCh    chan error

	mu sync.RWMutex // Mutex to protect access to the allowList
}

func newConfigManager(allowList ...string) *mockConfigManager {
	return &mockConfigManager{
		allowList: allowList,
		allowSet:  createSet(allowList...),
		reloadCh:  make(chan struct{}),
		errCh:     make(chan error),
	}
}

func (m *mockConfigManager) Watch(ctx context.Context) (<-chan struct{}, <-chan error, error) {
	if m.watchErr != nil {
		return nil, nil, m.watchErr
	}

	if m.reloadCh == nil {
		m.reloadCh = make(chan struct{})
	}

	if m.errCh == nil {
		m.errCh = make(chan error)
	}

	if m.closeReloadCh {
		close(m.reloadCh)
	}
	if m.closeWatchErr {
		close(m.errCh)
	} else if m.delayedWatchErr != nil {
		go func() {
			time.Sleep(2 * time.Second)
			m.errCh <- m.delayedWatchErr
		}()
	}
	return m.reloadCh, m.errCh, nil
}

func (m *mockConfigManager) AllowList() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	allowListCopy := make([]string, len(m.allowList))
	copy(allowListCopy, m.allowList)
	return allowListCopy
}

func (m *mockConfigManager) IsAllowed(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.allowSet[name]
	return ok
}

func (m *mockConfigManager) setAllowList(t *testing.T, newAllowList []string, sendReloadSignal uint) {
	t.Helper()

	m.mu.Lock() // Lock for writing
	defer m.mu.Unlock()
	m.allowList = newAllowList
	m.allowSet = createSet(newAllowList...)

	for range sendReloadSignal {
		require.NotNil(t, m.reloadCh, "Setup: Reload channel should not be nil")
		m.reloadCh <- struct{}{}
	}
}

func createSet(items ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

// run is a helper function which runs the worker manager in a separate goroutine
// and returns a channel to receive any errors that occur during the run.
//
// The channel is closed when the run is complete.
func run(ctx context.Context, t *testing.T, m *workers.Pool) chan error {
	t.Helper()

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := m.Run(ctx)
		if err != nil {
			runErr <- err
		}
	}()

	time.Sleep(50 * time.Millisecond) // Allow some time for the service to start
	return runErr
}

type mockDProcessor struct {
	processErrs map[string]error
}

func newProcessor(processErrs map[string]error) *mockDProcessor {
	return &mockDProcessor{processErrs: processErrs}
}

func (p *mockDProcessor) Process(ctx context.Context, app string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err, ok := p.processErrs[app]; ok {
		return err
	}
	return nil
}
