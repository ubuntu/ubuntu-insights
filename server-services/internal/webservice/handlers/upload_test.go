package handlers_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/server-services/internal/webservice/handlers"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		apps []string
	}{
		"Empty": {
			apps: []string{},
		},
		"Single": {
			apps: []string{"testapp"},
		},
		"Multiple": {
			apps: []string{"testapp", "anotherapp"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rd := t.TempDir()
			mockConfig := &mockConfigManager{
				allowedList: tc.apps,
			}

			handler := handlers.NewUpload(mockConfig, rd, 1<<10)
			assert.NotNil(t, handler)
			assert.Equal(t, rd, handler.ReportsDir())
			assert.Equal(t, tc.apps, mockConfig.AllowList())
		})
	}
}

func TestUpload(t *testing.T) {
	t.Parallel()
	const defaultApp = "testapp"

	tests := map[string]struct {
		request       *http.Request
		method        string
		maxUploadSize int64

		expectedCode int
	}{
		"Valid Upload": {
			request: insightsRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
		},
		"Disallowed App": {
			request:      insightsRequest(t, "unknown-app", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Empty App Name": {
			request:      insightsRequest(t, "", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Body Missing File": {
			request:      missingFileInsightsRequest(t, defaultApp),
			expectedCode: http.StatusBadRequest,
		},
		"Invalid Method - GET": {
			request:      insightsRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
		},
		"Invalid Method - PUT": {
			request:      insightsRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
		},
		"File too large": {
			request:       insightsRequest(t, defaultApp, bytes.Repeat([]byte("a"), 1<<20)), // 1 MB
			maxUploadSize: 1 << 10,                                                          // 1 KB
			expectedCode:  http.StatusBadRequest,
		},
		"Invalid JSON": {
			request:      insightsRequest(t, defaultApp, []byte(`{"foo": "bar",}`)),
			expectedCode: http.StatusBadRequest,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockConfig := &mockConfigManager{
				allowedList: []string{defaultApp},
			}

			if tc.method == "" {
				tc.method = http.MethodPost
			}
			if tc.maxUploadSize == 0 {
				tc.maxUploadSize = 1 << 10 // 1 KB
			}
			if tc.expectedCode == 0 {
				tc.expectedCode = http.StatusAccepted
			}

			handler := handlers.NewUpload(mockConfig, t.TempDir(), tc.maxUploadSize)
			tc.request.Method = tc.method

			runUploadTestCase(t, handler, tc.request, tc.expectedCode, handler.ReportsDir())
		})
	}
}

func insightsRequest(t *testing.T, app string, data []byte) *http.Request {
	t.Helper()

	req := createRequest(t, "/upload/"+app, data)
	req.SetPathValue("app", app)
	return req
}

func missingFileInsightsRequest(t *testing.T, app string) *http.Request {
	t.Helper()

	req := missingFileRequest(t, "/upload/"+app)
	req.SetPathValue("app", app)
	return req
}
