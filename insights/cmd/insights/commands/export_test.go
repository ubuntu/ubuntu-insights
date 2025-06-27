package commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

type (
	NewUploader  = newUploader
	NewCollector = newCollector
)

// SetArgs sets the arguments for the command.
func (a *App) SetArgs(args []string) {
	a.cmd.SetArgs(args)
}

// WithNewUploader sets the new uploader function for the app.
func WithNewUploader(nu NewUploader) Options {
	return func(o *options) {
		o.newUploader = nu
	}
}

// WithNewCollector sets the new collector function for the app.
func WithNewCollector(nc NewCollector) Options {
	return func(o *options) {
		o.newCollector = nc
	}
}

// NewAppForTests creates a new app for testing.
func NewAppForTests(t *testing.T, args []string, consentDir string, opts ...Options) (app *App, consentPath, cachePath string) {
	t.Helper()

	cachePath = filepath.Join(t.TempDir())
	cacheDir := filepath.Join("testdata", "reports")
	require.NoError(t, testutils.CopyDir(t, cacheDir, cachePath), "Setup: could not copy cache dir")
	args = append(args, "--insights-dir", cachePath)

	consentPath = t.TempDir()
	consentDir = filepath.Join("testdata", "consents", consentDir)
	require.NoError(t, testutils.CopyDir(t, consentDir, consentPath), "Setup: could not copy consent dir")

	args = append(args, "--consent-dir", consentPath)

	app, err := New(opts...)
	require.NoError(t, err, "Setup: could not create app")

	app.SetArgs(args)
	return app, consentPath, cachePath
}
