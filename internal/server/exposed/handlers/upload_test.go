package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/exposed/handlers"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

type mockConfigManager struct {
	baseDir     string
	allowedList []string
}

func (m *mockConfigManager) BaseDir() string {
	return m.baseDir
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

			bd := t.TempDir()
			mockConfig := &mockConfigManager{
				baseDir:     bd,
				allowedList: tc.apps,
			}

			handler := handlers.NewUpload(mockConfig, 1<<10)
			assert.NotNil(t, handler)
			assert.Equal(t, bd, mockConfig.BaseDir())
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
			request: createMultipartRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
		},
		"Disallowed App": {
			request:      createMultipartRequest(t, "unknown-app", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
			expectNoFile: true,
		},
		"Empty App Name": {
			request:      createMultipartRequest(t, "", []byte(`{"foo": "bar"}`)),
			expectedCode: http.StatusForbidden,
			expectNoFile: true,
		},
		"Body Missing File": {
			request:      missingFileRequest(t, defaultApp),
			expectedCode: http.StatusBadRequest,
			expectNoFile: true,
		},
		"Invalid Method - GET": {
			request:      createMultipartRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
			expectNoFile: true,
		},
		"Invalid Method - PUT": {
			request:      createMultipartRequest(t, defaultApp, []byte(`{"foo": "bar"}`)),
			method:       http.MethodPut,
			expectedCode: http.StatusMethodNotAllowed,
			expectNoFile: true,
		},
		"File too large": {
			request:       createMultipartRequest(t, defaultApp, bytes.Repeat([]byte("a"), 1<<20)), // 1 MB
			maxUploadSize: 1 << 10,                                                                 // 1 KB
			expectedCode:  http.StatusRequestEntityTooLarge,
			expectNoFile:  true,
		},
		"Invalid JSON": {
			request:      createMultipartRequest(t, defaultApp, []byte(`{"foo": "bar",}`)),
			expectedCode: http.StatusBadRequest,
			expectNoFile: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockConfig := &mockConfigManager{
				baseDir:     t.TempDir(),
				allowedList: []string{defaultApp},
			}

			if tc.method == "" {
				tc.method = http.MethodPost
			}
			if tc.maxUploadSize == 0 {
				tc.maxUploadSize = 1 << 10 // 1 KB
			}
			if tc.expectedCode == 0 {
				tc.expectedCode = http.StatusCreated
			}

			handler := handlers.NewUpload(mockConfig, tc.maxUploadSize)

			rr := httptest.NewRecorder()
			tc.request.Method = tc.method
			handler.ServeHTTP(rr, tc.request)

			assert.Equal(t, tc.expectedCode, rr.Code, "Expected status code")
			contents, err := testutils.GetDirContents(t, mockConfig.BaseDir(), 3)
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

func createMultipartRequest(t *testing.T, app string, data []byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", uuid.New().String()+".json")
	require.NoError(t, err, "Setup: failed to create form file")
	_, err = fw.Write(data)
	require.NoError(t, err, "Setup: failed to write data to form file")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/"+url.PathEscape(app), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
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
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetPathValue("app", app)
	return req
}
