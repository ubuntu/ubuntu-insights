package processor_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/models"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/processor"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/storage"
)

type mockUploader struct {
	Uploaded []*models.DBFileData
	Err      error
}

func (m *mockUploader) Upload(ctx context.Context, _ storage.DBExecutor, data *models.DBFileData) error {
	if m.Err != nil {
		return m.Err
	}
	m.Uploaded = append(m.Uploaded, data)
	return nil
}

func writeTestJSON(t *testing.T, dir, appID string, timestamp string) {
	t.Helper()

	data := map[string]any{
		"AppID":          appID,
		"Generated":      timestamp,
		"Schema Version": "1.0.0",
		"Common":         map[string]any{"Software": map[string]any{"os": "ubuntu-22.04"}},
		"AppData":        map[string]any{"Version": "1.15.1"},
	}

	appDir := filepath.Join(dir, appID)
	require.NoError(t, os.MkdirAll(appDir, 0750))

	path := filepath.Join(appDir, "test.json")
	content, err := json.Marshal(data)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, content, 0600))
}

func TestProcessFiles_Success(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	appID := "example-app"
	now := time.Now().Unix()

	writeTestJSON(t, tmpDir, appID, strconv.FormatInt(now, 10))

	cfg := &config.ServiceConfig{
		InputDir: tmpDir,
	}

	mock := &mockUploader{}

	ctx := context.Background()
	err := processor.ProcessFiles(ctx, cfg, mock)
	require.NoError(t, err)

	require.Len(t, mock.Uploaded, 1)
	assert.Equal(t, appID, mock.Uploaded[0].AppID)

	// Ensure file was removed after processing
	files, err := filepath.Glob(filepath.Join(tmpDir, appID, "*.json"))
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestProcessFiles_InvalidJSON(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	appID := "badapp"

	appDir := filepath.Join(tmpDir, appID)
	require.NoError(t, os.MkdirAll(appDir, 0750))

	path := filepath.Join(appDir, "test.json")
	require.NoError(t, os.WriteFile(path, []byte(`{ invalid json`), 0600))

	cfg := &config.ServiceConfig{InputDir: tmpDir}
	mock := &mockUploader{}
	err := processor.ProcessFiles(context.Background(), cfg, mock)

	require.NoError(t, err) // It's logged, not fatal
	assert.Empty(t, mock.Uploaded)

	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "File should be removed")
}

func TestProcessFiles_TimestampInFuture(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	appID := "futureapp"

	future := time.Now().Add(24 * time.Hour).Unix()
	writeTestJSON(t, tmpDir, appID, strconv.FormatInt(future, 10))

	cfg := &config.ServiceConfig{InputDir: tmpDir}
	mock := &mockUploader{}
	err := processor.ProcessFiles(context.Background(), cfg, mock)

	require.NoError(t, err)
	assert.Empty(t, mock.Uploaded)
}

func TestProcessFiles_UploadFails(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	appID := "failapp"

	now := time.Now().Unix()
	writeTestJSON(t, tmpDir, appID, strconv.FormatInt(now, 10))

	cfg := &config.ServiceConfig{InputDir: tmpDir}
	mock := &mockUploader{Err: errors.New("upload failed")}

	err := processor.ProcessFiles(context.Background(), cfg, mock)

	require.ErrorContains(t, err, "upload failed")
	assert.Empty(t, mock.Uploaded)

	files, _ := filepath.Glob(filepath.Join(tmpDir, appID, "*.json"))
	assert.NotEmpty(t, files, "File should not be removed on upload failure")
}

func TestProcessFiles_ContextCancelled(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	appID := "cancelapp"

	now := time.Now().Unix()
	writeTestJSON(t, tmpDir, appID, strconv.FormatInt(now, 10))

	cfg := &config.ServiceConfig{InputDir: tmpDir}
	mock := &mockUploader{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := processor.ProcessFiles(ctx, cfg, mock)

	require.NoError(t, err)
	assert.Empty(t, mock.Uploaded)
}
