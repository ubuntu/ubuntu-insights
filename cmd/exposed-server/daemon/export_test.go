package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
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

	p := GenerateTestConfig(t, conf, daeConf)
	argsWithConf := []string{"--config", p}
	argsWithConf = append(argsWithConf, args...)

	a, err := New()
	require.NoError(t, err, "Setup: failed to create app")
	a.cmd.SetArgs(argsWithConf)
	return a
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

// GenerateTestConfig generates a temporary config file for testing.
func GenerateTestConfig(t *testing.T, origConf *AppConfig, daeConf *config.Conf) string {
	t.Helper()

	var conf appConfig

	if origConf != nil {
		conf = *origConf
	}

	if conf.Verbosity == 0 {
		conf.Verbosity = 2
	}

	daeConfPath := GenerateTestDaeConfig(t, daeConf)
	conf.Daemon.ConfigPath = daeConfPath

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
