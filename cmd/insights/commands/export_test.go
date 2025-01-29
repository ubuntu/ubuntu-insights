package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type SetupConfig struct {
	MissingConfigDir   bool
	MissingCacheDir    bool
	MissingLocalDir    bool
	MissingUploadedDir bool

	ConfigFiles      map[string][]byte
	LocalDirFiles    map[string][]byte
	UploadedDirFiles map[string][]byte
}

func NewForTests(t *testing.T, config SetupConfig, args ...string) (app *App, configDir, cacheDir string) {
	t.Helper()

	configDir, cacheDir = SetupDirs(t, config)

	app, err := New()
	require.NoError(t, err, "Setup: could not create app")

	app.rootConfig.ConsentDir = configDir
	app.rootConfig.InsightsDir = cacheDir
	app.cmd.SetArgs(args)

	return app, configDir, cacheDir
}

func SetupDirs(t *testing.T, config SetupConfig) (configDir, cacheDir string) {
	t.Helper()

	configDir = filepath.Join(t.TempDir())
	cacheDir = filepath.Join(t.TempDir())

	// TODO: Implement uploadedDir and localDir after uploader component merge (PR #9)

	for file, content := range config.ConfigFiles {
		require.NoError(t, os.WriteFile(filepath.Join(configDir, file), content, 0600), "Setup: could not write config file")
	}

	if config.MissingConfigDir {
		require.NoError(t, os.RemoveAll(configDir), "Setup: could not remove config dir")
	}
	if config.MissingCacheDir {
		require.NoError(t, os.RemoveAll(cacheDir), "Setup: could not remove cache dir")
	}

	return configDir, cacheDir
}
