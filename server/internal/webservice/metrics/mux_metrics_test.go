package metrics_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

func TestNewMuxMiddleware(t *testing.T) {
	t.Parallel()

	// Ensure middleware is returned and no panic occurs.
	require.NotNil(t, metrics.NewMuxMiddleware(prometheus.NewRegistry()))
}

func TestMuxMiddlewareWrap(t *testing.T) {
	t.Parallel()

	const validPath = "/test-path"

	type request struct {
		method string
		path   string
		body   io.Reader
	}

	tests := map[string]struct {
		requests               []request
		withEndpointMiddleware bool // Whether to apply EndpointMiddleware to handlers registered with the mux.
	}{
		"No Requests": {},
		"Single GET Request": {
			requests: []request{
				{method: http.MethodGet, path: validPath, body: nil},
			},
		},
		"Single GET Request invalid path": {
			requests: []request{
				{method: http.MethodGet, path: "/invalid-path", body: nil},
			},
		},
		"Multiple Requests": {
			requests: []request{
				{method: http.MethodGet, path: validPath, body: nil},
				{method: http.MethodPost, path: "/invalid-path", body: nil},
				{method: http.MethodPut, path: validPath, body: nil},
				{method: http.MethodGet, path: "/invalid-path", body: nil},
				{method: http.MethodGet, path: validPath, body: nil},
			},
		},

		// With EndpointMiddleware
		"No Requests with Endpoint Middleware": {
			withEndpointMiddleware: true,
		},
		"Single GET Request with Endpoint Middleware": {
			requests: []request{
				{method: http.MethodGet, path: validPath, body: nil},
			},
			withEndpointMiddleware: true,
		},
		"Multiple Requests with Endpoint Middleware": {
			requests: []request{
				{method: http.MethodGet, path: validPath, body: nil},
				{method: http.MethodPost, path: "/invalid-path", body: nil},
				{method: http.MethodPut, path: validPath, body: nil},
				{method: http.MethodGet, path: "/invalid-path", body: nil},
				{method: http.MethodGet, path: validPath, body: nil},
			},
			withEndpointMiddleware: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			mw := metrics.NewMuxMiddleware(reg)

			mux := http.NewServeMux()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			})
			if tc.withEndpointMiddleware {
				handler = metrics.NewEndpointMiddleware(reg).Wrap(name, handler)
			}
			mux.Handle(validPath, handler)

			monitored := mw.Wrap(name, mux)

			for _, name := range endpointMetricNames {
				assert.Equal(t, 0, testutil.CollectAndCount(reg, name), "Expected no metrics to be collected before request")
			}

			for _, req := range tc.requests {
				status := http.StatusNotFound
				if req.path == validPath {
					status = http.StatusAccepted
				}
				sendRequest(t, monitored, req.method, req.path, req.body, status)
			}

			got := struct {
				HTTPMuxRequestsTotal string
				EndpointMetrics      map[string]string
			}{
				EndpointMetrics: make(map[string]string),
			}

			b, err := testutil.CollectAndFormat(reg, expfmt.TypeTextPlain, "http_mux_requests_total")
			require.NoError(t, err, "Failed to collect metrics for http_mux_requests_total", name)
			got.HTTPMuxRequestsTotal = string(b)

			if tc.withEndpointMiddleware {
				for _, name := range deterministicEndpointMetrics {
					b, err := testutil.CollectAndFormat(reg, expfmt.TypeTextPlain, name)
					require.NoError(t, err, "Failed to collect metrics for http_mux_requests_total", name)
					got.EndpointMetrics[name] = string(b)
				}
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collected metrics do not match expected values")
		})
	}
}
