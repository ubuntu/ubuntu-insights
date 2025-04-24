package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ubuntu/ubuntu-insights/cmd/exposed-server/server/handlers"
)

type mockConfigManager struct {
	BaseDir     string
	AllowedList []string
}

func (m *mockConfigManager) GetBaseDir() string {
	return m.BaseDir
}

func (m *mockConfigManager) GetAllowList() []string {
	return m.AllowedList
}

func setup(t *testing.T) (*handlers.UploadHandler, *mockConfigManager, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	mockConfig := &mockConfigManager{
		BaseDir:     tmpDir,
		AllowedList: []string{"testapp"},
	}

	uploadHandler := &handlers.UploadHandler{
		Config: mockConfig,
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return uploadHandler, mockConfig, cleanup
}

func createMultipartRequest(t *testing.T, app, filename string, data []byte) (*http.Request, error) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = fw.Write(data)
	if err != nil {
		return nil, err
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/"+app, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetPathValue("app", app)
	return req, nil
}

func TestUploadSuccess(t *testing.T) {
	t.Parallel()
	handler, mockConfig, cleanup := setup(t)
	defer cleanup()

	data := []byte(`{"foo": "bar"}`)
	req, err := createMultipartRequest(t, "testapp", "sample.json", data)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d", rr.Code)
	}

	files, err := os.ReadDir(filepath.Join(mockConfig.GetBaseDir(), "testapp"))
	if err != nil {
		t.Fatal("Expected file to be written but directory read failed:", err)
	}
	if len(files) != 1 {
		t.Fatalf("Expected one file to be written, got %d", len(files))
	}
}

func TestUploadDisallowedApp(t *testing.T) {
	t.Parallel()
	mockConfig := &mockConfigManager{
		AllowedList: []string{"allowedapp"},
	}

	handler := &handlers.UploadHandler{Config: mockConfig}

	req, err := createMultipartRequest(t, "notallowed", "sample.json", []byte(`{"foo": "bar"}`))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestUploadMissingApp(t *testing.T) {
	t.Parallel()
	handler, _, cleanup := setup(t)
	defer cleanup()

	req, err := createMultipartRequest(t, "", "sample.json", []byte(`{"foo": "bar"}`))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestUploadMissingFile(t *testing.T) {
	t.Parallel()
	handler, _, cleanup := setup(t)
	defer cleanup()

	// Make a POST request but without a "file" part
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload/testapp", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetPathValue("app", "testapp")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for missing file, got %d", rr.Code)
	}
}

func TestUploadInvalidMethod(t *testing.T) {
	t.Parallel()
	handler, _, cleanup := setup(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPut, "/upload/testapp", nil)
	req.SetPathValue("app", "testapp")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Expected 405 Method Not Allowed, got %d", rr.Code)
	}
}

func TestUploadFileTooLarge(t *testing.T) {
	t.Parallel()
	handler, mockConfig, cleanup := setup(t)
	defer cleanup()
	// Create a file that's too big
	oversizedData := bytes.Repeat([]byte("a"), handlers.MaxUploadSize+1)

	req, err := createMultipartRequest(t, "testapp", "huge.json", oversizedData)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("Expected status 413 Request Entity Too Large, got %d", rr.Code)
	}

	files, err := os.ReadDir(filepath.Join(mockConfig.GetBaseDir(), "testapp"))
	if err == nil && len(files) > 0 {
		t.Fatalf("Expected no file to be written for oversized input, but found %d files", len(files))
	}
}

func TestUploadInvalidJSONContent(t *testing.T) {
	t.Parallel()
	handler, _, cleanup := setup(t)
	defer cleanup()

	// Intentionally invalid JSON data
	invalidJSON := []byte(`{"invalid": true,,,`)

	req, err := createMultipartRequest(t, "testapp", "invalid.json", invalidJSON)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 Bad Request for invalid JSON, got %d", rr.Code)
	}
}
