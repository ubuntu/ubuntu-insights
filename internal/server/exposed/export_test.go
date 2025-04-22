package exposed

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
)

type DConfigManager = dConfigManager

// Addr returns the address of the HTTP server for testing purposes.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// HTTPServer returns the HTTP server for testing purposes.
func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

// GenerateTestDaeConfig generates a temporary daemon config file for testing.
func GenerateTestDaeConfig(t *testing.T, daeConf *config.Conf) string {
	t.Helper()

	d, err := json.Marshal(daeConf)
	require.NoError(t, err, "Setup: failed to marshal dynamic server config for tests")
	daeConfPath := filepath.Join(t.TempDir(), "daemon-testconfig.yaml")
	require.NoError(t, os.WriteFile(daeConfPath, d, 0600), "Setup: failed to write dynamic config for tests")

	return daeConfPath
}
