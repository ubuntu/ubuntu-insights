package config_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0600), "failed to write temp config file")
	return tmpFile
}

func TestLoadValidConfig(t *testing.T) {
	t.Parallel()
	content := `{
		"base_dir": "/tmp/data",
		"allowList": ["foo", "bar"]
	}`
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	if err := cm.Load(); err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if got := cm.BaseDir(); got != "/tmp/data" {
		t.Errorf("expected base_dir /tmp/data, got %s", got)
	}

	expected := []string{"foo", "bar"}
	if got := cm.AllowList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected allowList %v, got %v", expected, got)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	t.Parallel()
	content := `{
		"base_dir": "/tmp/data",
		"allowList": ["foo", "bar"]` // Missing closing brace
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	require.Error(t, cm.Load(), "expected error loading malformed JSON")
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()
	cm := config.New("nonexistent.json")
	require.Error(t, cm.Load(), "expected error loading missing config file")
}

func TestWatchMissingFile(t *testing.T) {
	t.Parallel()
	cm := config.New("somewhere/nonexistent.json")
	watchErr := testWatch(t, cm)

	select {
	case err := <-watchErr:
		require.Error(t, err, "expected error watching missing config file")
	case <-time.After(200 * time.Millisecond):
		require.Fail(t, "expected error watching missing config file")
	}
}

func TestWatchConfigReloadsOnChange(t *testing.T) {
	t.Parallel()
	initial := `{"base_dir": "/tmp/initial", "allowList": ["alpha"]}`
	updated := `{"base_dir": "/tmp/updated", "allowList": ["beta"]}`
	tmpFile := createTempConfigFile(t, initial)

	cm := config.New(tmpFile)
	require.NoError(t, cm.Load(), "Setup: initial load failed")

	watchErr := testWatch(t, cm)

	require.NoError(t, os.WriteFile(tmpFile, []byte(updated), 0600), "Setup: failed to write updated config")

	time.Sleep(time.Second) // let watcher reload

	assert.Equal(t, "/tmp/updated", cm.BaseDir(), "expected base_dir to be updated")
	if got := cm.AllowList(); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Errorf("expected allowList [beta], got %v", got)
	}

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchConfigRemoved(t *testing.T) {
	t.Parallel()
	logs := map[slog.Level]uint{
		slog.LevelInfo: 2,
	}

	initial := `{"base_dir": "/tmp/initial", "allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)

	l := testutils.NewMockHandler(slog.LevelDebug)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	require.NoError(t, cm.Load(), "Setup: initial load failed")
	watchErr := testWatch(t, cm)

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	require.NoError(t, os.Remove(tmpFile), "Setup: failed to remove config file")
	time.Sleep(200 * time.Millisecond) // let watcher reload

	require.Error(t, cm.Load(), "Expected error loading removed config file")

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchIgnoresIrrelevantFiles(t *testing.T) {
	t.Parallel()
	logs := map[slog.Level]uint{
		slog.LevelInfo: 2,
	}

	initial := `{"base_dir": "/tmp/initial", "allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)
	irrelevantFile := filepath.Join(filepath.Dir(tmpFile), "irrelevant.txt")

	l := testutils.NewMockHandler(slog.LevelDebug)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	require.NoError(t, cm.Load(), "Setup: initial load failed")
	watchErr := testWatch(t, cm)

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	require.NoError(t, os.WriteFile(irrelevantFile, []byte("irrelevant content"), 0600), "Setup: failed to write irrelevant file")
	time.Sleep(200 * time.Millisecond) // let watcher reload

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchWarnsIfLoadFails(t *testing.T) {
	t.Parallel()

	initial := `{"base_dir": "/tmp/initial", "allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)

	l := testutils.NewMockHandler(slog.LevelInfo)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	require.NoError(t, cm.Load(), "Setup: initial load failed")
	watchErr := testWatch(t, cm)

	require.NoError(t, os.WriteFile(tmpFile, []byte("invalid json"), 0600), "Setup: failed to write invalid config")
	time.Sleep(time.Second) // let watcher reload

	// There are sometimes two warning entries due to how different OSes handle events related to os.WriteFile.
	levels := l.GetLevels()
	assert.GreaterOrEqual(t, levels[slog.LevelWarn], uint(1), "expected at least one warning log")
	assert.LessOrEqual(t, levels[slog.LevelWarn], uint(2), "expected at most two warning logs")

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestConfigManagerReadWhileWrite(t *testing.T) {
	content := `{}`
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	err := os.WriteFile(tmpFile, []byte(`{"base_dir":"/tmp/test","allowList":["foo"]}`), 0600)
	require.NoError(t, err, "Setup: Failed to write initial config")
	require.NoError(t, cm.Load(), "Setup: Failed to load initial config")

	var wg sync.WaitGroup
	writeCount := 100
	readCount := 100

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range writeCount {
			_ = os.WriteFile(tmpFile, fmt.Appendf(nil, `{"base_dir":"/tmp/test%d","allowList":["foo"]}`, i), 0600)
			_ = cm.Load()
		}
	}()

	// Reader goroutines
	for range readCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cm.BaseDir()
			_ = cm.AllowList()
		}()
	}

	wg.Wait()
	require.Equal(t, "/tmp/test99", cm.BaseDir(), "Expected base_dir to be /tmp/test99")
	require.Equal(t, []string{"foo"}, cm.AllowList(), "Expected allowList to be [foo]")
}

func testWatch(t *testing.T, cm *config.Manager) chan error {
	t.Helper()
	watchErr := make(chan error, 1)
	go func() {
		defer close(watchErr)
		watchErr <- cm.Watch(t.Context())
	}()
	time.Sleep(100 * time.Millisecond) // let watcher initialize
	return watchErr
}
