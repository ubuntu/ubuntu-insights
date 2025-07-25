package webservice

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
)

type DConfigManager = dConfigManager

// HTTPServer returns the HTTP server for testing purposes.
func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

// GenerateTestDaemonConfig generates a temporary daemon config file for testing.
func GenerateTestDaemonConfig(t *testing.T, daeConf *config.Conf) string {
	t.Helper()

	d, err := json.Marshal(daeConf)
	require.NoError(t, err, "Setup: failed to marshal dynamic server config for tests")
	daeConfPath := filepath.Join(t.TempDir(), "daemon-testconfig.yaml")
	require.NoError(t, os.WriteFile(daeConfPath, d, 0600), "Setup: failed to write dynamic config for tests")

	return daeConfPath
}

// PrimaryAddr returns the true address of the primary server.
func (s *Server) PrimaryAddr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.primaryAddr
}

// MetricsAddr returns the true address of the metrics server.
func (s *Server) MetricsAddr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricsAddr
}
