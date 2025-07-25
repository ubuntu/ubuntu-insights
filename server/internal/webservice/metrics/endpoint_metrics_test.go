package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

var endpointMetricNames = []string{
	"http_endpoint_requests_total",
	"http_endpoint_request_duration_seconds",
	"http_endpoint_request_size_bytes",
}

var deterministicEndpointMetrics = []string{
	"http_endpoint_requests_total",
	"http_endpoint_request_size_bytes",
}

func TestNewEndpointMiddleware(t *testing.T) {
	t.Parallel()

	// Ensure middleware is returned and no panic occurs.
	require.NotNil(t, metrics.NewEndpointMiddleware(prometheus.NewRegistry()))
}

func TestEndpointMiddlewareWrap(t *testing.T) {
	t.Parallel()

	type request struct {
		method string
		path   string
		body   io.Reader
	}

	tests := map[string]struct {
		requests    []request
		applyLabels bool
	}{
		"No Requests": {},
		"Single GET Request": {
			requests: []request{
				{method: http.MethodGet, path: "/test-get", body: nil},
			},
		},
		"Single GET Request with Labels": {
			requests: []request{
				{method: http.MethodGet, path: "/test-get", body: nil},
			},
			applyLabels: true,
		},
		"Multiple Requests": {
			requests: []request{
				{method: http.MethodGet, path: "/test-get", body: nil},
				{method: http.MethodPost, path: "/test-post", body: nil},
				{method: http.MethodPut, path: "/test-put", body: nil},
				{method: http.MethodGet, path: "/test-get", body: nil},
			},
		},
		"Multiple Requests with Labels": {
			requests: []request{
				{method: http.MethodGet, path: "/test-get", body: nil},
				{method: http.MethodPost, path: "/test-post", body: nil},
				{method: http.MethodPut, path: "/test-put", body: nil},
				{method: http.MethodGet, path: "/test-get", body: nil},
			},
			applyLabels: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reg := prometheus.NewRegistry()
			mw := metrics.NewEndpointMiddleware(reg)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			})
			if tc.applyLabels {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					metrics.ApplyLabels(r)
					w.WriteHeader(http.StatusAccepted)
				})
			}

			monitored := mw.Wrap(name, handler)

			for _, name := range endpointMetricNames {
				assert.Equal(t, 0, testutil.CollectAndCount(reg, name), "Expected no metrics to be collected before request")
			}

			for _, req := range tc.requests {
				sendRequest(t, monitored, req.method, req.path, req.body, http.StatusAccepted)
			}

			var got = map[string]string{}
			for _, name := range deterministicEndpointMetrics {
				b, err := testutil.CollectAndFormat(reg, expfmt.TypeTextPlain, name)
				require.NoError(t, err, "Failed to collect metrics for %s", name)
				got[name] = string(b)
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collected metrics do not match expected values")
		})
	}
}

func TestApplyLabels(t *testing.T) {
	t.Parallel()

	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/test-path"},
	}

	metrics.ApplyLabels(req)

	assert.Equal(t, "GET", req.Method, "Expected method to be GET")
	assert.Equal(t, "/test-path", req.URL.Path, "Expected path to be /test-path")

	// Check if the context has the label applied
	ctx := req.Context()
	labelValue := ctx.Value(metrics.LabelPath)
	assert.Equal(t, "/test-path", labelValue, "Expected context to have path label")
}

func TestHandlerApplyLabels(t *testing.T) {
	t.Parallel()

	handler := metrics.HandlerApplyLabels(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test/path", r.Context().Value(metrics.LabelPath), "Expected path label to be applied")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "Expected status code to be OK")
	assert.Equal(t, "/test/path", req.Context().Value(metrics.LabelPath), "Expected path label to be applied")
}

func sendRequest(t *testing.T, handler http.HandlerFunc, method, target string, body io.Reader, expectedCode int) {
	t.Helper()

	req := httptest.NewRequest(method, target, body)
	rec := httptest.NewRecorder()
	handler(rec, req)

	assert.Equal(t, expectedCode, rec.Code, "Expected status code %d, got %d", expectedCode, rec.Code)
}
