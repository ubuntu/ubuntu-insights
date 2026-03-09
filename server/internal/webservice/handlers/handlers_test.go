package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice/metrics"
)

var deterministicHandlerMetricNames = []string{
	"http_endpoint_requests_total",
	"http_endpoint_request_size_bytes",
}

type uploadTestGolden struct {
	DirContents map[string][]string `yaml:"dir_contents"`
	Metrics     map[string]string   `yaml:"metrics,omitempty"`
}

type mockConfigManager struct {
	allowedList []string
}

func (m *mockConfigManager) AllowList() []string {
	return m.allowedList
}

func (m *mockConfigManager) allowSet() map[string]struct{} {
	allowSet := make(map[string]struct{}, len(m.allowedList))
	for _, name := range m.allowedList {
		allowSet[name] = struct{}{}
	}
	return allowSet
}

func (m *mockConfigManager) IsAllowed(name string) bool {
	_, ok := m.allowSet()[name]
	return ok
}

func runUploadTestCase(
	t *testing.T,
	handler http.Handler,
	req *http.Request,
	expectedCode int,
	reportsDir string,
	reg *prometheus.Registry,
) {
	t.Helper()

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, expectedCode, rr.Code, "Expected status code")

	contents, err := testutils.GetDirContents(t, reportsDir, 3)
	require.NoError(t, err, "Failed to get directory contents")

	got := uploadTestGolden{DirContents: make(map[string][]string)}
	for _, file := range contents {
		dir := filepath.Dir(file)
		got.DirContents[dir] = append(got.DirContents[dir], file)
	}

	if reg != nil {
		got.Metrics = make(map[string]string)
		for _, name := range deterministicHandlerMetricNames {
			b, err := testutil.CollectAndFormat(reg, expfmt.TypeTextPlain, name)
			require.NoError(t, err, "Failed to collect metrics for %s", name)
			got.Metrics[name] = string(b)
		}
	}

	want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
	assert.Equal(t, want, got, "Directory contents and metrics do not match golden file")
}

// newEndpointMiddlewareWrap creates a fresh registry, wraps handler with EndpointMiddleware,
// and returns both the wrapped handler and the registry for metric assertions.
func newEndpointMiddlewareWrap(handlerName string, handler http.Handler) (http.Handler, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	mw := metrics.NewEndpointMiddleware(reg)
	return mw.Wrap(handlerName, handler), reg
}

func createRequest(t *testing.T, target string, data []byte) *http.Request {
	t.Helper()

	body := bytes.NewReader(data)
	req := httptest.NewRequest(http.MethodPost, target, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func missingFileRequest(t *testing.T, target string) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	require.NoError(t, w.WriteField("not_a_file", "oops"), "Setup: failed to write field")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, target, &b)
	req.Header.Set("Content-Type", "application/json")
	return req
}
