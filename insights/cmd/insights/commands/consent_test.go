package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

func TestConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args []string

		consentDir  string
		removeFiles []string

		wantErr      bool
		wantUsageErr bool
	}{
		// Get
		"Get Global True":      {args: []string{"consent"}},
		"Get Global False":     {args: []string{"consent"}, consentDir: "false-global"},
		"Get Source True":      {args: []string{"consent", "True"}},
		"Get Source False":     {args: []string{"consent", "False"}},
		"Get Multiple Sources": {args: []string{"consent", "True", "False"}},
		"Get Global Empty":     {args: []string{"consent"}, consentDir: "empty-global"},
		"Get Global Bad Key":   {args: []string{"consent"}, consentDir: "bad-key-global"},

		// Get Errors
		"Get Multiple Sources errors when source is missing ": {args: []string{"consent", "True", "Unknown"}, wantErr: true},
		"Get Multiple Sources errors when source file bad":    {args: []string{"consent", "True", "Bad-File", "False"}, wantErr: true},

		"Get errors when Global missing":   {args: []string{"consent"}, removeFiles: []string{"consent.toml"}, wantErr: true},
		"Get errors when Global bad file":  {args: []string{"consent"}, consentDir: "bad-file-global", wantErr: true},
		"Get errors when Global bad ext":   {args: []string{"consent"}, consentDir: "bad-ext-global", wantErr: true},
		"Get errors when Global bad value": {args: []string{"consent"}, consentDir: "bad-value-global", wantErr: true},

		"Get errors when source missing": {args: []string{"consent", "unknown"}, wantErr: true},

		// Set
		"Set global to new value":     {args: []string{"consent", "--state=false"}},
		"Set global to same value":    {args: []string{"consent", "--state=true"}},
		"Set source to new value":     {args: []string{"consent", "False", "--state=true"}},
		"Set source to same value":    {args: []string{"consent", "True", "--state=true"}},
		"Set multiple sources:":       {args: []string{"consent", "True", "False", "-s=false"}},
		"Set new source":              {args: []string{"consent", "Unknown", "--state=true"}},
		"Set existing and new source": {args: []string{"consent", "True", "Unknown", "-s=true"}},
		"Set existing and bad source": {args: []string{"consent", "True", "Bad-File", "False", "-s=true"}},

		"Set shorthand True": {args: []string{"consent", "-s=true"}},

		// Usage Errors
		"Usage errors when passing bad flag":           {args: []string{"consent", "-unknown"}, wantUsageErr: true, wantErr: true},
		"Usage errors when unparsable state is passed": {args: []string{"consent", "-s=bad"}, wantUsageErr: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentDir == "" {
				tc.consentDir = "true-global"
			}
			app, configDir := newAppForTests(t, tc.args, tc.consentDir, tc.removeFiles)

			err := app.Run()
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

			got, err := testutils.GetDirContents(t, configDir, 2)
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected consent files state")
		},
		)
	}
}

func newAppForTests(t *testing.T, args []string, consentDir string, removeFiles []string) (a *commands.App, cDir string) {
	t.Helper()

	cDir = t.TempDir()
	consentDir = filepath.Join("testdata", "consents", consentDir)
	require.NoError(t, testutils.CopyDir(t, consentDir, cDir), "Setup: could not copy consent dir")

	for _, file := range removeFiles {
		require.NoError(t, os.RemoveAll(filepath.Join(cDir, file)), "Setup: could not remove file")
	}

	a, err := commands.New()
	require.NoError(t, err, "Setup: could not create app")

	args = append(args, "--consent-dir", cDir)
	a.SetArgs(args)
	return a, cDir
}
