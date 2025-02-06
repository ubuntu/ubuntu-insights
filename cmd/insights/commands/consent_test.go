package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
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
		"Get Global True":                      {args: []string{"consent"}},
		"Get Global False":                     {args: []string{"consent"}, consentDir: "false-global"},
		"Get Source True":                      {args: []string{"consent", "True"}},
		"Get Source False":                     {args: []string{"consent", "False"}},
		"Get Multiple Sources":                 {args: []string{"consent", "True", "False"}},
		"Get Multiple Sources Partial Missing": {args: []string{"consent", "True", "Unknown"}, wantErr: true},
		"Get Multiple Sources Partial Bad":     {args: []string{"consent", "True", "Bad-File", "False"}, wantErr: true},
		"Get Global Empty":                     {args: []string{"consent"}, consentDir: "empty-global"},

		"Get Global Missing":   {args: []string{"consent"}, removeFiles: []string{"consent.toml"}, wantErr: true},
		"Get Global Bad File":  {args: []string{"consent"}, consentDir: "bad-file-global", wantErr: true},
		"Get Global Bad Ext":   {args: []string{"consent"}, consentDir: "bad-ext-global", wantErr: true},
		"Get Global Bad Key":   {args: []string{"consent"}, consentDir: "bad-key-global"},
		"Get Global Bad Value": {args: []string{"consent"}, consentDir: "bad-value-global", wantErr: true},

		"Get Source Missing": {args: []string{"consent", "unknown"}, wantErr: true},

		"Set Global":                          {args: []string{"consent", "--consent-state=false"}},
		"Set Global Same":                     {args: []string{"consent", "--consent-state=true"}},
		"Set Source":                          {args: []string{"consent", "False", "--consent-state=true"}},
		"Set Source Same":                     {args: []string{"consent", "True", "--consent-state=true"}},
		"Set Source Multiple:":                {args: []string{"consent", "True", "False", "-c=false"}},
		"Set Source Missing":                  {args: []string{"consent", "Unknown", "--consent-state=true"}},
		"Set Source Multiple Partial Missing": {args: []string{"consent", "True", "Unknown", "-c=true"}},
		"Set Source Multiple Partial Bad":     {args: []string{"consent", "True", "Bad-File", "False", "-c=true"}},

		"Set Shorthand True": {args: []string{"consent", "-c=true"}},

		"Bad Command": {args: []string{"consent", "-unknown"}, wantUsageErr: true, wantErr: true},
		"Bad State":   {args: []string{"consent", "-c=bad"}, wantUsageErr: true, wantErr: true},
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
