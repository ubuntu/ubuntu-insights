package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

func TestGetSystemOptOut(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args          []string
		initialConfig string

		wantErr      bool
		wantUsageErr bool
	}{
		// Get
		"Get opted-out true":                        {args: []string{"system-opt-out"}, initialConfig: "system_opt_out = true\n"},
		"Get opted-out false":                       {args: []string{"system-opt-out"}, initialConfig: "system_opt_out = false\n"},
		"Get with no config file defaults to false": {args: []string{"system-opt-out"}},
		"Get with the quiet flag":                   {args: []string{"system-opt-out", "--quiet"}, initialConfig: "system_opt_out = true\n"},

		// Errors
		"Get errors when config file is malformed": {args: []string{"system-opt-out"}, initialConfig: "system_opt_out = bad\n", wantErr: true},

		// Usage Errors
		"Usage errors when passing bad flag":                    {args: []string{"system-opt-out", "-unknown"}, wantErr: true, wantUsageErr: true},
		"Usage errors when passing an argument":                 {args: []string{"system-opt-out", "extra"}, wantErr: true, wantUsageErr: true},
		"Usage errors when verbose and quiet are used together": {args: []string{"system-opt-out", "--verbose", "--quiet"}, wantErr: true, wantUsageErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app, systemConfigDir := newSystemOptOutApp(t, tc.args, tc.initialConfig)
			preRunDirContents, err := testutils.GetDirContents(t, systemConfigDir, 2)
			require.NoError(t, err, "Setup: failed to read system config dir")

			err = app.Run()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantUsageErr {
				assert.True(t, app.UsageError())
			} else {
				assert.False(t, app.UsageError())
			}

			postRunDirContents, err := testutils.GetDirContents(t, systemConfigDir, 2)
			require.NoError(t, err)
			require.Equal(t, preRunDirContents, postRunDirContents, "Get should not modify the system config dir")
		})
	}
}

func TestSetSystemOptOut(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args          []string
		initialConfig string

		wantUsageErr bool
	}{
		// Set
		"Set true with no existing file":  {args: []string{"system-opt-out", "--state=true"}},
		"Set false with no existing file": {args: []string{"system-opt-out", "--state=false"}},
		"Set true overrides false":        {args: []string{"system-opt-out", "--state=true"}, initialConfig: "system_opt_out = false\n"},
		"Set false overrides true":        {args: []string{"system-opt-out", "-s=false"}, initialConfig: "system_opt_out = true\n"},
		"Set shorthand true":              {args: []string{"system-opt-out", "-s=true"}},
		"Set does not error with quiet":   {args: []string{"system-opt-out", "--state=true", "--quiet"}},

		// Usage Errors
		"Usage errors when state is unparsable":            {args: []string{"system-opt-out", "-s=bad"}, wantUsageErr: true},
		"Usage errors when state is unparsable with quiet": {args: []string{"system-opt-out", "-s=bad", "--quiet"}, wantUsageErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			app, systemConfigDir := newSystemOptOutApp(t, tc.args, tc.initialConfig)

			err := app.Run()
			if tc.wantUsageErr {
				require.Error(t, err)
				assert.True(t, app.UsageError())
			} else {
				require.NoError(t, err)
				assert.False(t, app.UsageError())
			}

			got, err := testutils.GetDirContents(t, systemConfigDir, 2)
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected system config file state")
		})
	}
}

// newSystemOptOutApp builds an App configured to use a temporary system config directory.
// If initialConfig is not empty, it is written to the system config file before the app runs.
func newSystemOptOutApp(t *testing.T, args []string, initialConfig string) (app *commands.App, systemConfigDir string) {
	t.Helper()

	systemConfigDir = t.TempDir()
	if initialConfig != "" {
		require.NoError(t, os.WriteFile(
			filepath.Join(systemConfigDir, constants.SystemConfigFileName),
			[]byte(initialConfig), 0600),
			"Setup: failed to write initial system config file")
	}

	args = append(args, "--system-config-dir", systemConfigDir)

	app, err := commands.New()
	require.NoError(t, err, "Setup: could not create app")

	app.SetArgs(args)
	return app, systemConfigDir
}
