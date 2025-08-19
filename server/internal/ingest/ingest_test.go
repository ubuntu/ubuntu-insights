package ingest_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest"
)

func TestRun(t *testing.T) {
	t.Parallel()

	const maxDegradedDuration = 800 * time.Millisecond

	tests := map[string]struct {
		workerPool    *mockWorkerPool
		metricsServer *mockMetricsServer

		cancelContextPreRun bool // Cancel context before running the service
		cancelContext       bool // Cancel context after early error check

		triggerWorkerPoolErrEarly    bool // Trigger an error in the worker pool before run
		triggerMetricsServerErrEarly bool // Trigger an error in the metrics server before run

		// Within 50ms of early service state check
		wantEarlyReturn bool // Return early without error
		wantEarlyErr    bool // Errors within 200ms

		// Within maxDegradedDuration + 100ms after early service state check
		wantLateReturn bool // Return after late duration without error
		wantLateErr    bool // Errors after lateDuration

		wantSpecificErr error // Specific error to check for
	}{
		"Default run blocks": {},

		// Context cancellation
		"Context cancel before run errors fast": {
			cancelContextPreRun: true,
			wantEarlyErr:        true,
			wantSpecificErr:     context.Canceled,
		},
		"Context cancel after run without blocked close returns without err": {
			cancelContext:  true,
			wantLateReturn: true,
		},
		"Context cancel after run with blocked close returns with err": {
			metricsServer: &mockMetricsServer{
				closeDelay: 2 * time.Second,
			},
			cancelContext:   true,
			wantLateErr:     true,
			wantSpecificErr: ingest.ErrTeardownTimeout,
		},

		// Worker Pool errors
		"WorkerPool Run errors early": {
			workerPool: &mockWorkerPool{
				runErr: errors.New("requested worker pool run error"),
			},
			triggerWorkerPoolErrEarly: true,
			wantEarlyErr:              true,
		},
		"WorkerPool Run errors late": {
			workerPool: &mockWorkerPool{
				runErr: errors.New("requested worker pool run error"),
			},
			wantLateErr: true,
		},

		// Metrics Server errors
		"MetricsServer ListenAndServe errors early": {
			metricsServer: &mockMetricsServer{
				listenAndServeErr: errors.New("requested metrics server listen and serve error"),
			},
			triggerMetricsServerErrEarly: true,
			wantEarlyErr:                 true,
		},
		"MetricsServer ListenAndServe errors late": {
			metricsServer: &mockMetricsServer{
				listenAndServeErr: errors.New("requested metrics server listen and serve error"),
			},
			wantLateErr: true,
		},

		// Degraded state
		"Teardown Timeout when worker pool fails and metrics shutdown hangs": {
			workerPool: &mockWorkerPool{
				runErr: errors.New("requested worker pool run error"),
			},
			metricsServer: &mockMetricsServer{
				shutdownDelay: 2 * time.Second,
			},
			wantLateErr:     true,
			wantSpecificErr: ingest.ErrTeardownTimeout,
		},
		"Teardown Timeout when metrics server fails and worker pool hangs": {
			workerPool: &mockWorkerPool{
				hang: true,
			},
			metricsServer: &mockMetricsServer{
				listenAndServeErr: errors.New("requested metrics server listen and serve error"),
			},
			wantLateErr:     true,
			wantSpecificErr: ingest.ErrTeardownTimeout,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Sanitize test case
			// Only one of wantEarlyReturn, wantLateReturn, wantEarlyErr, or wantLateErr should be true at most.
			wants := []bool{tc.wantEarlyErr, tc.wantLateErr, tc.wantEarlyReturn, tc.wantLateReturn}
			oneTrue := false
			for _, w := range wants {
				if w {
					require.False(t, oneTrue, "Setup: Only one of the wants flags should be true at most",
						"got: %v", wants)
					oneTrue = true
				}
			}
			if tc.workerPool == nil {
				tc.workerPool = &mockWorkerPool{}
			}
			if tc.metricsServer == nil {
				tc.metricsServer = &mockMetricsServer{}
			}

			tc.workerPool.initialize(t)
			tc.metricsServer.initialize(t)

			args := []ingest.Option{
				ingest.WithMaxDegradedDuration(maxDegradedDuration),
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			service := ingest.New(ctx, tc.workerPool, tc.metricsServer, args...)

			if tc.cancelContextPreRun {
				cancel()
			}

			if tc.triggerWorkerPoolErrEarly {
				tc.workerPool.triggerError()
			}
			if tc.triggerMetricsServerErrEarly {
				tc.metricsServer.triggerError()
			}

			errCh := runServiceAsync(t, service)

			select {
			case err := <-errCh:
				if !tc.wantEarlyErr {
					require.NoError(t, err, "Service should not have exited early with error")
					require.True(t, tc.wantEarlyReturn, "Service should not have exited early without error")
					return
				}
				require.Error(t, err, "Expected early error but got nil from early return")
				if tc.wantSpecificErr != nil {
					require.ErrorIs(t, err, tc.wantSpecificErr, "Expected specific error but got different error")
				}
				return
			case <-time.After(maxDegradedDuration + 100*time.Millisecond):
			}
			require.False(t, tc.wantEarlyErr, "Service should have exited early with error but did not")
			require.False(t, tc.wantEarlyReturn, "Service should have exited early without error but did not")

			if tc.cancelContext {
				cancel()
			}

			// WorkerPool and MetricsServer always be non-nil error, so this is dependent on if the return error was set.
			tc.workerPool.triggerError()
			tc.metricsServer.triggerError()

			select {
			case err := <-errCh:
				if !tc.wantLateErr {
					require.NoError(t, err, "Service should not have exited late with error")
					require.True(t, tc.wantLateReturn, "Service should not have exited late without error")
					return
				}
				require.Error(t, err, "Expected late error but got nil from late return")
				if tc.wantSpecificErr != nil {
					require.ErrorIs(t, err, tc.wantSpecificErr, "Expected specific error but got different error")
				}
				return
			case <-time.After(maxDegradedDuration + 100*time.Millisecond):
			}
			require.False(t, tc.wantLateErr, "Service should have exited late with error but did not")
			require.False(t, tc.wantLateReturn, "Service should have exited late without error but did not")
		})
	}
}

func TestQuit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		workerPool    *mockWorkerPool
		metricsServer *mockMetricsServer

		triggerWorkerPoolErr    bool // Trigger an error in the worker pool after run
		triggerMetricsServerErr bool // Trigger an error in the metrics server after run

		force     bool
		earlyQuit bool

		wantHang bool
		wantErr  bool
	}{
		"Basic Quit completes": {},
		"Force Quit completes": {
			force: true,
		},

		"Force Quit does not hang on metrics server shutdown": {
			metricsServer: &mockMetricsServer{
				shutdownDelay: 2 * time.Second,
			},
			force: true,
		},
		"Force Quit hangs on metrics server close": {
			metricsServer: &mockMetricsServer{
				closeDelay: 2 * time.Second,
			},
			force:    true,
			wantHang: true,
		},
		"Quit hangs on metrics server shutdown": {
			metricsServer: &mockMetricsServer{
				shutdownDelay: 2 * time.Second,
			},
			wantHang: true,
		},
		"Quit does not hang on metrics server close": {
			metricsServer: &mockMetricsServer{
				closeDelay: 2 * time.Second,
			},
		},

		// Error conditions
		"Early Quit errors": {
			earlyQuit: true,
			wantErr:   true,
		},
		"Early Force Quit errors": {
			earlyQuit: true,
			force:     true,
			wantErr:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.workerPool == nil {
				tc.workerPool = &mockWorkerPool{}
			}
			if tc.metricsServer == nil {
				tc.metricsServer = &mockMetricsServer{}
			}

			tc.workerPool.initialize(t)
			tc.metricsServer.initialize(t)

			args := []ingest.Option{
				ingest.WithMaxDegradedDuration(1 * time.Second),
			}

			service := ingest.New(t.Context(), tc.workerPool, tc.metricsServer, args...)

			if tc.earlyQuit {
				timedQuit(t, service, tc.force, tc.wantHang)
				if tc.wantHang {
					return
				}
			}

			errCh := runServiceAsync(t, service)

			select {
			case err := <-errCh:
				if tc.earlyQuit {
					if tc.wantErr {
						require.Error(t, err, "Expected error on early Quit but got none")
						return
					}
					require.NoError(t, err, "Unexpected error on early Quit")
					return
				}
				require.Fail(t, "Service should not have exited early before Quit")
			case <-time.After(100 * time.Millisecond):
				if tc.earlyQuit {
					require.Fail(t, "Service should have early Quit but did not")
				}
			}

			if tc.triggerWorkerPoolErr {
				tc.workerPool.triggerError()
			}
			if tc.triggerMetricsServerErr {
				tc.metricsServer.triggerError()
			}
			time.Sleep(50 * time.Millisecond)

			timedQuit(t, service, tc.force, tc.wantHang)
			if tc.wantHang {
				return
			}
		})
	}
}

// runServiceAsync runs the ingest service in a goroutine and returns a channel to receive any errors.
func runServiceAsync(t *testing.T, service *ingest.Service) <-chan error {
	t.Helper()

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- service.Run()
	}()

	// Allow some time for things to process
	time.Sleep(50 * time.Millisecond)
	return errCh
}

func quitServiceAsync(t *testing.T, service *ingest.Service, force bool) <-chan struct{} {
	t.Helper()

	running := make(chan struct{})
	go func() {
		defer close(running)
		service.Quit(force)
	}()

	return running
}

// timedQuit runs the Quit method.
// If hang is not expected and quit times out, it will error.
// If hang is expected but it does not hang, it will error.
//
// Hang timeout is set to 500 milliseconds.
func timedQuit(t *testing.T, service *ingest.Service, force bool, hang bool) {
	t.Helper()

	quitRunning := quitServiceAsync(t, service, force)

	select {
	case <-quitRunning:
		require.False(t, hang, "Expected quit to hang but it did not")
	case <-time.After(500 * time.Millisecond):
		require.True(t, hang, "Expected quit to exit but it did not")
	}
}

type mockWorkerPool struct {
	hang   bool
	runErr error

	internalCtx    context.Context
	internalCancel context.CancelFunc
}

// initializes the mock worker pool.
func (p *mockWorkerPool) initialize(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	p.internalCtx = ctx
	p.internalCancel = cancel
}

// Run simulates the worker pool's Run method.
func (p *mockWorkerPool) Run(ctx context.Context) error {
	if p.hang {
		// If hang is true, ignore the ctx
		<-p.internalCtx.Done()
		return p.runErr
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.internalCtx.Done():
		return p.runErr
	}
}

// triggerError simulates an error condition in the worker pool.
// If runErr is set, it will cancel the internal context to simulate an error condition.
func (p *mockWorkerPool) triggerError() {
	if p.runErr != nil {
		p.internalCancel()
	}
}

type mockMetricsServer struct {
	shutdownSignal chan struct{}
	shutdownDelay  time.Duration
	shutdownErr    error
	shutdownOnce   sync.Once

	closeSignal chan struct{}
	closeDelay  time.Duration
	closeErr    error
	closeOnce   sync.Once

	internalCtx       context.Context
	internalCancel    context.CancelFunc
	listenAndServeErr error

	running chan struct{}
}

// initialize sets up the mock metrics server with the provided context.
func (m *mockMetricsServer) initialize(t *testing.T) {
	t.Helper()
	m.shutdownSignal = make(chan struct{})
	m.closeSignal = make(chan struct{})

	ctx, cancel := context.WithCancel(t.Context())
	m.internalCtx = ctx
	m.internalCancel = cancel
}

// ListenAndServe simulates the metrics server's ListenAndServe method.
func (m *mockMetricsServer) ListenAndServe() error {
	m.running = make(chan struct{})
	defer close(m.running)

	select {
	case <-m.internalCtx.Done():
	case <-m.shutdownSignal:
		return http.ErrServerClosed
	case <-m.closeSignal:
		return http.ErrServerClosed
	}
	return m.listenAndServeErr
}

// Shutdown simulates graceful shutdown of the metrics server.
func (m *mockMetricsServer) Shutdown(ctx context.Context) error {
	m.shutdownOnce.Do(func() {
		close(m.shutdownSignal)
	})

	select {
	case <-time.After(m.shutdownDelay):
	case <-ctx.Done():
		return ctx.Err()
	}

	return m.shutdownErr
}

// Close simulates closing the metrics server.
func (m *mockMetricsServer) Close() error {
	m.closeOnce.Do(func() {
		close(m.closeSignal)
	})

	time.Sleep(m.closeDelay)
	return m.closeErr
}

// triggerError simulates an error condition in the metrics server.
// If listenAndServeErr is set, it will cancel the internal context to simulate an error condition.
func (m *mockMetricsServer) triggerError() {
	if m.listenAndServeErr != nil {
		m.internalCancel()
	}
}
