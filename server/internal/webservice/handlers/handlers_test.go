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
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

type mockConfigManager struct {
	allowedList []string
}

func (m *mockConfigManager) AllowList() []string {
	return m.allowedList
}

func (m *mockConfigManager) AllowSet() map[string]struct{} {
	allowSet := make(map[string]struct{}, len(m.allowedList))
	for _, name := range m.allowedList {
		allowSet[name] = struct{}{}
	}
	return allowSet
}

func runUploadTestCase(
	t *testing.T,
	handler http.Handler,
	req *http.Request,
	expectedCode int,
	reportsDir string,
) {
	t.Helper()

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, expectedCode, rr.Code, "Expected status code")

	contents, err := testutils.GetDirContents(t, reportsDir, 3)
	require.NoError(t, err, "Failed to get directory contents")

	got := make(map[string][]string)
	for _, file := range contents {
		dir := filepath.Dir(file)
		got[dir] = append(got[dir], file)
	}

	want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
	assert.Equal(t, want, got, "Directory contents do not match golden file")
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
