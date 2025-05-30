package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/webservice/handlers"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

type mockConfigManager struct {
	allowedList []string
}

func (m *mockConfigManager) AllowList() []string {
	return m.allowedList
}

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
		expectNoFile bool
	}{
		"Valid Upload": {
			request: createRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
		},
		"Disallowed App": {
			request:      createRequest(t, "unknown-app", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
			expectNoFile: true,
		},
		"Empty App Name": {
			request:      createRequest(t, "", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
			expectNoFile: true,
		},
		"Body Missing File": {
			request:      missingFileRequest(t, defaultApp),
			expectedCode: http.StatusBadRequest,
			expectNoFile: true,
		},
		"Invalid Method - GET": {
			request:      createRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
			expectNoFile: true,
		},
		"Invalid Method - PUT": {
			request:      createRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
			expectNoFile: true,
		},
		"File too large": {
			request:       createRequest(t, defaultApp, bytes.Repeat([]byte("a"), 1<<20)), // 1 MB
			maxUploadSize: 1 << 10,                                                        // 1 KB
			expectedCode:  http.StatusBadRequest,
			expectNoFile:  true,
		},
		"Invalid JSON": {
			request:      createRequest(t, defaultApp, []byte(`{"foo": "bar",}`)),
			expectedCode: http.StatusBadRequest,
			expectNoFile: true,
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

			rr := httptest.NewRecorder()
			tc.request.Method = tc.method
			handler.ServeHTTP(rr, tc.request)

			assert.Equal(t, tc.expectedCode, rr.Code, "Expected status code")
			contents, err := testutils.GetDirContents(t, handler.ReportsDir(), 3)
			require.NoError(t, err, "Failed to get directory contents")

			// Remove uuid from filename
			got := make(map[string][]string)
			for _, file := range contents {
				dir := filepath.Dir(file)
				got[dir] = append(got[dir], file)
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Directory contents do not match golden file")
		})
	}
}

func createRequest(t *testing.T, app string, data []byte) *http.Request {
	t.Helper()

	body := bytes.NewReader(data)
	req := httptest.NewRequest(http.MethodPost, "/upload/"+app, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("app", app)
	return req
}

func missingFileRequest(t *testing.T, app string) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	require.NoError(t, w.WriteField("not_a_file", "oops"), "Setup: failed to write field")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/"+app, &b)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("app", app)
	return req
}
