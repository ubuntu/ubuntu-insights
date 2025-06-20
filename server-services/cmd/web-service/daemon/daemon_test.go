package daemon_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server-services/cmd/web-service/daemon"
	"github.com/ubuntu/ubuntu-insights/server-services/internal/shared/config"
	"github.com/ubuntu/ubuntu-insights/server-services/internal/shared/constants"
	"github.com/ubuntu/ubuntu-insights/server-services/internal/webservice"
)

func TestConfigArg(t *testing.T) {
	filename := "conf.yaml"
	configPath := filepath.Join(t.TempDir(), filename)
	require.NoError(t, os.WriteFile(configPath, []byte("Verbosity: 1"), 0600), "Setup: couldn't write config file")

	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	a.SetArgs("version", "--config", configPath)

	err = a.Run()
	require.NoError(t, err, "Run should not return an error")
	require.Equal(t, 1, a.Config().Verbosity)
}

func TestConfigEnv(t *testing.T) {
	// Set the environment variable to point to the config file
	t.Setenv("UBUNTU_INSIGHTS_WEB_SERVICE_DAEMON_READTIMEOUT", "1s")

	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	a.SetArgs("version")

	err = a.Run()
	require.NoError(t, err, "Run should not return an error")
	require.Equal(t, time.Second, a.Config().Daemon.ReadTimeout)
}

func TestConfigBadArg(t *testing.T) {
	filename := "conf.yaml"
	configPath := filepath.Join(t.TempDir(), filename)

	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	a.SetArgs("version", "--config", configPath)

	err = a.Run()
	require.Error(t, err, "Run should return an error")
}

func TestDaeConfigBadPathErrors(t *testing.T) {
	t.Parallel()

	conf := &daemon.AppConfig{
		Daemon: webservice.StaticConfig{
			ConfigPath: "/does/not/exist.yaml",
		},
	}
	a := daemon.NewForTests(t, conf, nil)

	chErr := make(chan error, 1)
	go func() {
		chErr <- a.Run()
	}()
	a.WaitReady()
	time.Sleep(50 * time.Millisecond)

	err := <-chErr
	require.Error(t, err, "Run should return with an error")
}

func TestNoUsageError(t *testing.T) {
	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	a.SetArgs("completion", "bash")

	err = a.Run()
	require.NoError(t, err, "Run should not return an error")

	isUsageError := a.UsageError()
	require.False(t, isUsageError, "No usage error is reported as such")
}

func TestUsageError(t *testing.T) {
	t.Parallel()

	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	a.SetArgs("doesnotexist")

	err = a.Run()
	require.Error(t, err, "Run should return an error")
	isUsageError := a.UsageError()
	require.True(t, isUsageError, "Usage error is reported as such")

	// Test when SilenceUsage is true
	a.SetSilenceUsage(true)
	assert.False(t, a.UsageError())

	// Test when SilenceUsage is false
	a.SetSilenceUsage(false)
	assert.True(t, a.UsageError())
}

func TestAppCanSigHupAfterExecute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Hup test on Windows")
	}
	r, w, err := os.Pipe()
	require.NoError(t, err, "Setup: pipe shouldn't fail")

	a, wait := startDaemon(t, nil, nil)
	a.Quit()
	wait()

	orig := os.Stdout
	os.Stdout = w

	a.Hup()

	os.Stdout = orig
	w.Close()

	var out bytes.Buffer
	_, err = io.Copy(&out, r)
	require.NoError(t, err, "Couldn't copy stdout to buffer")
	require.NotEmpty(t, out.String(), "Stacktrace is printed")
}

func TestBadConfigReturnsError(t *testing.T) {
	a, err := daemon.New()
	require.NoError(t, err, "Setup: New should not return an error")
	// Use version to still run preExec to load no config but without running server
	a.SetArgs("version", "--config", "/does/not/exist.yaml")

	err = a.Run()
	require.Error(t, err, "Run should return an error on config file")
}

func TestRootCmd(t *testing.T) {
	app, err := daemon.New()
	require.NoError(t, err)

	cmd := app.RootCmd()

	assert.NotNil(t, cmd, "Returned root cmd should not be nil")
	assert.Equal(t, constants.WebServiceCmdName, cmd.Name())
}

// startDaemon prepares and starts the daemon in the background. The done function should be called
// to wait for the daemon to stop.
//
// The done function should be called in the main goroutine for the test.
func startDaemon(t *testing.T, conf *daemon.AppConfig, daeConf *config.Conf) (app *daemon.App, done func()) {
	t.Helper()

	a := daemon.NewForTests(t, conf, daeConf)

	chErr := make(chan error, 1)
	go func() {
		chErr <- a.Run()
	}()
	a.WaitReady()
	time.Sleep(50 * time.Millisecond)

	return a, func() {
		err := <-chErr
		require.NoError(t, err, "Run should return without an error")
	}
}
