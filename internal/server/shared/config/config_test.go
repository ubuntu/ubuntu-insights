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
		"allowList": ["foo", "bar"]
	}`
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	if err := cm.Load(); err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	expected := []string{"foo", "bar"}
	if got := cm.AllowList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected allowList %v, got %v", expected, got)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	t.Parallel()
	content := `{
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
	watchEvent, watchErr, err := cm.Watch(t.Context())
	require.Error(t, err, "Expected error starting watch on missing config file")

	select {
	case <-watchErr:
		require.Fail(t, "expected no error in watchErr channel")
	case <-watchEvent:
		require.Fail(t, "expected no event for missing config file")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchConfigReloadsOnChange(t *testing.T) {
	t.Parallel()
	initial := `{"allowList": ["alpha"]}`
	updated := `{"allowList": ["beta"]}`
	tmpFile := createTempConfigFile(t, initial)

	cm := config.New(tmpFile)
	require.NoError(t, cm.Load(), "Setup: initial load failed")

	watchEvent, watchErr, err := cm.Watch(t.Context())
	require.NoError(t, err, "Setup: failed to start watch")

	require.NoError(t, os.WriteFile(tmpFile, []byte(updated), 0600), "Setup: failed to write updated config")

	time.Sleep(time.Second) // let watcher reload

	if got := cm.AllowList(); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Errorf("expected allowList [beta], got %v", got)
	}

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-watchEvent:
	case <-time.After(200 * time.Millisecond):
		require.Fail(t, "expected change event")
	}
}

func TestWatchConfigRemoved(t *testing.T) {
	t.Parallel()
	logs := map[slog.Level]uint{
		slog.LevelInfo: 2,
	}

	initial := `{"allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)

	l := testutils.NewMockHandler(slog.LevelDebug)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	watchEvent, watchErr, err := cm.Watch(t.Context())
	require.NoError(t, err, "Setup: failed to start watch")

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	require.NoError(t, os.Remove(tmpFile), "Setup: failed to remove config file")
	time.Sleep(200 * time.Millisecond) // let watcher reload

	if !l.AssertLevels(t, logs) {
		l.OutputLogs(t)
	}

	// Ensure that no channels are written to, as there isn't a successful reload
	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-watchEvent:
		require.Fail(t, "expected no successful change event")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchIgnoresIrrelevantFiles(t *testing.T) {
	t.Parallel()
	logs := map[slog.Level]uint{
		slog.LevelInfo: 2,
	}

	initial := `{"allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)
	irrelevantFile := filepath.Join(filepath.Dir(tmpFile), "irrelevant.txt")

	l := testutils.NewMockHandler(slog.LevelDebug)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	watchEvent, watchErr, err := cm.Watch(t.Context())
	require.NoError(t, err, "Setup: failed to start watch")

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
	case <-watchEvent:
		require.Fail(t, "expected no change event")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchWarnsIfLoadFails(t *testing.T) {
	t.Parallel()

	initial := `{"allowList": ["alpha"]}`
	tmpFile := createTempConfigFile(t, initial)

	l := testutils.NewMockHandler(slog.LevelInfo)
	cm := config.New(tmpFile, config.WithLogger(slog.New(&l)))
	watchEvent, watchErr, err := cm.Watch(t.Context())
	require.NoError(t, err, "Setup: failed to start watch")

	require.NoError(t, os.WriteFile(tmpFile, []byte("invalid json"), 0600), "Setup: failed to write invalid config")
	time.Sleep(time.Second) // let watcher reload

	// There are sometimes two warning entries due to how different OSes handle events related to os.WriteFile.
	levels := l.GetLevels()
	assert.GreaterOrEqual(t, levels[slog.LevelWarn], uint(1), "expected at least one warning log")
	assert.LessOrEqual(t, levels[slog.LevelWarn], uint(2), "expected at most two warning logs")

	select {
	case err := <-watchErr:
		require.NoError(t, err, "expected no error watching config file")
	case <-watchEvent:
		require.Fail(t, "expected no change event")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestConfigManagerReadWhileWrite(t *testing.T) {
	content := `{}`
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	err := os.WriteFile(tmpFile, []byte(`{"allowList":["foo"]}`), 0600)
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
			_ = os.WriteFile(tmpFile, fmt.Appendf(nil, `{"allowList":["foo", "foo%d"]}`, i), 0600)
			_ = cm.Load()
		}
	}()

	// Reader goroutines
	for range readCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cm.AllowList()
		}()
	}

	wg.Wait()
	require.Equal(t, []string{"foo", "foo99"}, cm.AllowList(), "Expected allowList to be [foo, foo99]")
}
