package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ubuntu/ubuntu-insights/cmd/ingest-server/server/config"
)


func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}
	return tmpFile
}

func TestLoad_ValidConfig(t *testing.T) {
	content := `{
		"base_dir": "/tmp/data",
		"allowList": ["foo", "bar"]
	}`
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	if err := cm.Load(); err != nil {
		t.Fatalf("expected no error loading config, got %v", err)
	}

	if got := cm.GetBaseDir(); got != "/tmp/data" {
		t.Errorf("expected base_dir /tmp/data, got %s", got)
	}

	expected := []string{"foo", "bar"}
	if got := cm.GetAllowList(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected allowList %v, got %v", expected, got)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	content := `{
		"base_dir": "/tmp/data",
		"allowList": ["foo", "bar"]` // Missing closing brace
	tmpFile := createTempConfigFile(t, content)

	cm := config.New(tmpFile)
	if err := cm.Load(); err == nil {
		t.Fatal("expected error loading malformed JSON, got nil")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cm := config.New("nonexistent.json")
	if err := cm.Load(); err == nil {
		t.Fatal("expected error loading missing config file, got nil")
	}
}

func TestWatch_ConfigReloadsOnChange(t *testing.T) {
	initial := `{"base_dir": "/tmp/initial", "allowList": ["alpha"]}`
	updated := `{"base_dir": "/tmp/updated", "allowList": ["beta"]}`
	tmpFile := createTempConfigFile(t, initial)

	cm := config.New(tmpFile)
	if err := cm.Load(); err != nil {
		t.Fatalf("initial load failed: %v", err)
	}

	go cm.Watch()
	time.Sleep(100 * time.Millisecond) // let watcher initialize

	if err := os.WriteFile(tmpFile, []byte(updated), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	time.Sleep(200 * time.Millisecond) // let watcher reload

	if cm.GetBaseDir() != "/tmp/updated" {
		t.Errorf("expected base_dir to be updated, got %s", cm.GetBaseDir())
	}
	if got := cm.GetAllowList(); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Errorf("expected allowList [beta], got %v", got)
	}
}
