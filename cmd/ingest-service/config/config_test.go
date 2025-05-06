package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/ingest-service/config"
)

func writeTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)

	return tmpFile
}

func TestLoad_ValidConfig(t *testing.T) {
	t.Parallel()
	json := `
{
  "input_dir": "/var/lib/ingest",
  "db": {
    "host": "localhost",
    "port": 5432,
    "user": "admin",
    "password": "secret",
    "db_name": "insights",
    "sslmode": "disable"
  }
}
`
	path := writeTempConfigFile(t, json)

	cfg, err := config.Load(path)
	require.NoError(t, err)

	require.Equal(t, "/var/lib/ingest", cfg.InputDir)
	require.Equal(t, "localhost", cfg.DB.Host)
	require.Equal(t, 5432, cfg.DB.Port)
	require.Equal(t, "admin", cfg.DB.User)
	require.Equal(t, "secret", cfg.DB.Password)
	require.Equal(t, "insights", cfg.DB.DBName)
	require.Equal(t, "disable", cfg.DB.SSLMode)
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := config.Load("/non/existent/file.json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening config file")
}

func TestLoad_InvalidJSON(t *testing.T) {
	t.Parallel()
	json := `{"input_dir": "/tmp", "db": {"host": "localhost",` // malformed JSON
	path := writeTempConfigFile(t, json)

	_, err := config.Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding config JSON")
}
