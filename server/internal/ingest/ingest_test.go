package ingest_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest"
)

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cm   *mockConfigManager
		proc *mockDProcessor

		wantErr bool
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

			if tc.proc == nil {
				tc.proc = newProcessor(map[string]error{})
			}

			s := ingest.New(t.Context(), tc.cm, tc.proc)
			runErr := run(t, s)
			time.Sleep(50 * time.Millisecond) // Allow some time for the service to start

			t.Cleanup(func() {
				gracefulShutdown(t, s, runErr)
			})

			if tc.wantErr {
				checkService(t, runErr, true, 3*time.Second)
				return
			}

			waitWorkersEqual(t, s, tc.cm.AllowList()...)
			// Ensure no errors are received
			checkService(t, runErr, false, 0)
		})
	}
}

// Tests the addition and removal of apps from the allow list
// and verifies that the service updates its workers accordingly.
func TestRunModifyAllowList(t *testing.T) {
	t.Parallel()

	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	s := ingest.New(t.Context(), cm, &mockDProcessor{})
	runErr := run(t, s)

	waitWorkersEqual(t, s, cm.AllowList()...)

	cm.SetAllowList(t, append(cm.AllowList(), "MultiMixed"), 3)
	waitWorkersEqual(t, s, cm.AllowList()...)

	cm.SetAllowList(t, []string{}, 3)
	waitWorkersEqual(t, s)

	gracefulShutdown(t, s, runErr)
}

func TestRunAfterQuitErrors(t *testing.T) {
	t.Parallel()

	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	s := ingest.New(t.Context(), cm, &mockDProcessor{})
	defer s.Quit(true)

	runErr := run(t, s)

	checkService(t, runErr, false, 1*time.Second)
	gracefulShutdown(t, s, runErr)

	runErr = make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()
	checkService(t, runErr, true, 3*time.Second)
}

func TestRunEarlyContextCancel(t *testing.T) {
	t.Parallel()
	cm := newConfigManager("MultiValid1", "MultiValid2", "MultiValid3")
	proc := newProcessor(map[string]error{
		"SingleValid": context.Canceled,
	})

	ctx, cancel := context.WithCancel(t.Context())
	s := ingest.New(ctx, cm, proc)
	runErr := run(t, s)

	// Ensure no errors are received before the context is canceled
	checkService(t, runErr, false, 50*time.Millisecond)

	cancel()

	// Ensure that the service exited within a reasonable time
	select {
	case err := <-runErr:
		require.NoError(t, err, "Expected nil error after context cancellation")
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
		require.Fail(t, "Service closed unexpectedly: %v", err)
	case <-time.After(duration):
		require.False(t, expectErr, "Service did not exit with an error within the expected duration")
	}
}

// gracefulShutdown is a helper function which simulates a graceful shutdown of the service.
// If the service does not shutdown within 8 seconds, it fails the test.
// If runErr receives an error during shutdown, it fails the test.
func gracefulShutdown(t *testing.T, s *ingest.Service, runErr chan error) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		s.Quit(false)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		require.Fail(t, "Service failed to shutdown gracefully within 8 seconds")
	}

	// Check for any errors during shutdown
	select {
	case err := <-runErr:
		require.NoError(t, err, "Service failed to shutdown gracefully")
	case <-time.After(2 * time.Second):
		require.Fail(t, "Service has not returned after 2 seconds")
	}
}

func waitWorkersEqual(t *testing.T, s *ingest.Service, workers ...string) {
	t.Helper()
	delay := 500 * time.Millisecond
	timeout := 8 * time.Second

	start := time.Now()
	for {
		got := s.WorkerNames()

		slices.Sort(got)
		slices.Sort(workers)

		if slices.Equal(workers, got) {
			return
		}
		require.LessOrEqual(t, time.Since(start), timeout, "Workers did not match within the timeout")
		time.Sleep(delay)
	}
}

type mockConfigManager struct {
	allowList []string

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
	m.mu.RLock() // Lock for reading
	defer m.mu.RUnlock()
	return m.allowList
}

func (m *mockConfigManager) SetAllowList(t *testing.T, newAllowList []string, sendReloadSignal uint) {
	t.Helper()

	m.mu.Lock() // Lock for writing
	defer m.mu.Unlock()
	m.allowList = newAllowList

	for range sendReloadSignal {
		require.NotNil(t, m.reloadCh, "Setup: Reload channel should not be nil")
		m.reloadCh <- struct{}{}
	}
}

// run is a helper function which runs the service in a separate goroutine
// and returns a channel to receive any errors that occur during the run.
//
// The channel is closed when the run is complete.
func run(t *testing.T, s *ingest.Service) chan error {
	t.Helper()

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
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
