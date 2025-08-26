package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
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
func NewForTests(t *testing.T, conf *AppConfig, daeConf *config.Conf, args ...string) *App {
	t.Helper()

	if conf == nil {
		conf = &AppConfig{}
	}

	if conf.ReportsDir == "" {
		conf.ReportsDir = filepath.Join(t.TempDir(), "reports")
	}

	allowlistConf := conf.ConfigPath
	if allowlistConf == "" {
		allowlistConf = GenerateTestDaemonConfig(t, daeConf)
	}

	p := GenerateTestConfig(t, conf)
	argsWithConf := []string{allowlistConf, "--config", p}
	argsWithConf = append(argsWithConf, args...)

	a, err := New()
	require.NoError(t, err, "Setup: failed to create app")
	a.cmd.SetArgs(argsWithConf)
	return a
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
