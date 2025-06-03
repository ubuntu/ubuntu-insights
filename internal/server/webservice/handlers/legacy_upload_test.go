package handlers_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/internal/server/webservice/handlers"
)

func TestNewLegacyReport(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		apps []string
	}{
		"Empty": {
			apps: []string{},
		},
		"Single": {
			apps: []string{"ubuntu-report/distribution/desktop/version"},
		},
		"Multiple": {
			apps: []string{"ubuntu-report/distribution/desktop/version", "anotherapp"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rd := t.TempDir()
			mockConfig := &mockConfigManager{
				allowedList: tc.apps,
			}

			handler := handlers.NewLegacyReport(mockConfig, rd, 1<<10)
			assert.NotNil(t, handler)
			assert.Equal(t, rd, handler.ReportsDir())
			assert.Equal(t, tc.apps, mockConfig.AllowList())
		})
	}
}

func TestLegacyReportUpload(t *testing.T) {
	t.Parallel()
	const (
		defaultDistribution = "distribution"
		defaultVersion      = "version"
	)

	tests := map[string]struct {
		request       *http.Request
		method        string
		maxUploadSize int64

		expectedCode int
	}{
		"Valid Upload": {
			request: reportRequest(t, defaultDistribution, defaultVersion, []byte(`{"foo": "bar"}`)),
		},
		"Disallowed Distribution": {
			request:      reportRequest(t, "unknown-distribution", defaultVersion, []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Empty Distribution Name": {
			request:      reportRequest(t, "", defaultVersion, []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Disallowed Version": {
			request:      reportRequest(t, defaultDistribution, "unknown-version", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Empty Version Name": {
			request:      reportRequest(t, defaultDistribution, "", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
		},
		"Body Missing File": {
			request:      missingFileReportRequest(t, defaultDistribution, defaultVersion),
			expectedCode: http.StatusBadRequest,
		},
		"Invalid Method - GET": {
			request:      reportRequest(t, defaultDistribution, defaultVersion, []byte(`{"foo": "bar"}`)),
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
		},
		"Invalid Method - PUT": {
			request:      reportRequest(t, defaultDistribution, defaultVersion, []byte(`{"foo": "bar"}`)),
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
		},
		"File too large": {
			request:       reportRequest(t, defaultDistribution, defaultVersion, bytes.Repeat([]byte("a"), 1<<20)), // 1 MB
			maxUploadSize: 1 << 10,                                                                                 // 1 KB
			expectedCode:  http.StatusBadRequest,
		},
		"Invalid JSON": {
			request:      reportRequest(t, defaultDistribution, defaultVersion, []byte(`{"foo": "bar",}`)),
			expectedCode: http.StatusBadRequest,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockConfig := &mockConfigManager{
				allowedList: []string{"ubuntu-report/" + defaultDistribution + "/desktop/" + defaultVersion},
			}

			if tc.method == "" {
				tc.method = http.MethodPost
			}
			if tc.maxUploadSize == 0 {
				tc.maxUploadSize = 1 << 10 // 1 KB
			}
			if tc.expectedCode == 0 {
				tc.expectedCode = http.StatusOK
			}

			handler := handlers.NewLegacyReport(mockConfig, t.TempDir(), tc.maxUploadSize)
			tc.request.Method = tc.method

			runUploadTestCase(t, handler, tc.request, tc.expectedCode, handler.ReportsDir())
		})
	}
}

func reportRequest(t *testing.T, distribution, version string, data []byte) *http.Request {
	t.Helper()

	req := createRequest(t, "/"+distribution+"/desktop/"+version, data)
	req.SetPathValue("distribution", distribution)
	req.SetPathValue("version", version)
	return req
}

func missingFileReportRequest(t *testing.T, distribution, version string) *http.Request {
	t.Helper()

	req := missingFileRequest(t, "/"+distribution+"/desktop/"+version)
	req.SetPathValue("distribution", distribution)
	req.SetPathValue("version", version)
	return req
}
