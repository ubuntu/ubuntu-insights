package metrics_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/metrics"
)

func TestListenAndServe(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg     *metrics.Config
		wantErr bool
	}{
		"Default configuration": {},

		"Bad port": {
			cfg: &metrics.Config{
				Port: -1, // Invalid port
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.cfg = initConfig(t, tc.cfg)

			reg := prometheus.NewRegistry()
			server := metrics.New(*tc.cfg, reg)

			errCh := listenAndServeAsync(t, server)
			defer server.Close()

			select {
			case err := <-errCh:
				if tc.wantErr {
					require.Error(t, err, "Expected ListenAndServe to fail")
					return
				}
				require.Failf(t, "ListenAndServe returned unexpectedly", "Got possible error: %v", err)
			case <-time.After(500 * time.Millisecond):
				require.False(t, tc.wantErr, "Expected ListenAndServe to return an error but it did not")
			}

			addr := server.Addr()
			require.NotNil(t, addr, "Expected server address to be set")

			// Try to access the metrics endpoint
			statusCode, err := sendRequest(t, server)
			require.NoError(t, err, "Expected to successfully send request to metrics endpoint")
			require.Equal(t, http.StatusOK, statusCode, "Expected metrics endpoint to return 200 OK")
		})
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	cfg := initConfig(t, nil)

	reg := prometheus.NewRegistry()
	server := metrics.New(*cfg, reg)

	errCh := listenAndServeAsync(t, server)
	defer server.Close()

	// Ensure the server is running
	select {
	case err := <-errCh:
		require.Failf(t, "ListenAndServe returned unexpectedly", "Got possible error: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	statusCode, err := sendRequest(t, server)
	require.NoError(t, err, "Expected to successfully send request to metrics endpoint")
	require.Equal(t, http.StatusOK, statusCode, "Expected metrics endpoint to return 200 OK")

	err = server.Shutdown(t.Context())
	require.NoError(t, err, "Expected Shutdown to succeed")

	// Ensure the server is no longer running
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, http.ErrServerClosed, "Expected ListenAndServe to return ErrServerClosed after shutdown")
	default:
		require.Fail(t, "Expected ListenAndServe to return an error after shutdown")
	}

	_, err = sendRequest(t, server)
	require.Error(t, err, "Expected error when sending request after shutdown")
}

func TestClose(t *testing.T) {
	t.Parallel()

	cfg := initConfig(t, nil)

	reg := prometheus.NewRegistry()
	server := metrics.New(*cfg, reg)

	errCh := listenAndServeAsync(t, server)
	defer server.Close()

	// Ensure the server is running
	select {
	case err := <-errCh:
		require.Failf(t, "ListenAndServe returned unexpectedly", "Got possible error: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	err := server.Close()
	require.NoError(t, err, "Expected Close to succeed")

	// Ensure the server is no longer running
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, http.ErrServerClosed, "Expected ListenAndServe to return ErrServerClosed after close")
	default:
		require.Fail(t, "Expected ListenAndServe to return an error after close")
	}
}

func TestAddr(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg *metrics.Config
	}{
		"Default configuration": {},
		"Returns empty string if server fails to start": {
			cfg: &metrics.Config{
				Port: -1, // Invalid port
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.cfg = initConfig(t, tc.cfg)

			reg := prometheus.NewRegistry()
			server := metrics.New(*tc.cfg, reg)
			require.Empty(t, server.Addr(), "Expected Addr to be empty before ListenAndServe")

			errCh := listenAndServeAsync(t, server)
			defer server.Close()

			select {
			case <-errCh:
				require.Empty(t, server.Addr(), "Expected Addr to be empty if ListenAndServe fails")
				return
			case <-time.After(500 * time.Millisecond):
			}

			require.NotEmpty(t, server.Addr(), "Expected Addr to be set after ListenAndServe")
		})
	}
}

func initConfig(t *testing.T, cfg *metrics.Config) *metrics.Config {
	t.Helper()

	if cfg == nil {
		cfg = &metrics.Config{}
	}

	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	return cfg
}

func listenAndServeAsync(t *testing.T, server *metrics.Server) chan error {
	t.Helper()

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- server.ListenAndServe()
	}()
	return errCh
}

func sendRequest(t *testing.T, server *metrics.Server) (int, error) {
	t.Helper()

	resp, err := http.Get("http://" + server.Addr() + "/metrics")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
