package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/constants"
	"gopkg.in/yaml.v3"
)

type (
	AppConfig = appConfig
)

// Config returns the configuration of the app.
func (a *App) Config() AppConfig {
	return a.config
}

// NewForTests creates a new App instance for testing purposes.
func NewForTests(t *testing.T, conf *AppConfig, allowlistPath string, args ...string) *App {
	t.Helper()

	p := GenerateTestConfig(t, conf)
	argsWithConf := []string{allowlistPath, "--config", p}
	argsWithConf = append(argsWithConf, args...)

	a, err := New()
	require.NoError(t, err, "Setup: failed to create app")
	a.cmd.SetArgs(argsWithConf)
	return a
}

// GenerateTestAllowlist generates a temporary allowlist configuration file for testing.
func GenerateTestAllowlist(t *testing.T, allowlist *config.Conf) string {
	t.Helper()

	d, err := json.Marshal(allowlist)
	require.NoError(t, err, "Setup: failed to marshal dynamic server config for tests")
	allowlistPath := filepath.Join(t.TempDir(), "allowlist-test.yaml")
	require.NoError(t, os.WriteFile(allowlistPath, d, 0600), "Setup: failed to write dynamic config for tests")

	return allowlistPath
}

// GenerateTestConfig generates a temporary config file for testing.
func GenerateTestConfig(t *testing.T, origConf *AppConfig) string {
	t.Helper()

	var conf appConfig

	if origConf != nil {
		conf = *origConf
	}

	if conf.Verbosity == 0 {
		conf.Verbosity = 2
	}

	if conf.Daemon.ReportsDir == "" {
		conf.Daemon.ReportsDir = filepath.Join(t.TempDir(), constants.DefaultServiceReportsFolder)
	}

	d, err := yaml.Marshal(conf)
	require.NoError(t, err, "Setup: failed to marshal config for tests")

	confPath := filepath.Join(t.TempDir(), "testconfig.yaml")
	require.NoError(t, os.WriteFile(confPath, d, 0600), "Setup: failed to write config for tests")

	return confPath
}

// SetArgs set some arguments on root command for tests.
func (a *App) SetArgs(args ...string) {
	a.cmd.SetArgs(args)
}

// SetSilenceUsage set the SilenceUsage flag on root command for tests.
func (a *App) SetSilenceUsage(silence bool) {
	a.cmd.SilenceUsage = silence
}
