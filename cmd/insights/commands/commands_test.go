package commands_test

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args []string

		consentFiles map[string]bool
		otherFiles   map[string]string
		missingDir   bool
		dirNoPerms   bool

		wantErr       bool
		wantUsageErr  bool
		wantGetDirErr bool
	}{
		"Get Global True":      {args: []string{"consent"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": false}},
		"Get Global False":     {args: []string{"consent"}, consentFiles: map[string]bool{"consent.toml": false, "abc-consent.toml": false}},
		"Get Source True":      {args: []string{"consent", "abc"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": false}},
		"Get Source False":     {args: []string{"consent", "abc"}, consentFiles: map[string]bool{"consent.toml": false, "abc-consent.toml": false}},
		"Get Multiple Sources": {args: []string{"consent", "abc", "def"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": false, "def-consent.toml": false}},

		"Get Global Missing":     {args: []string{"consent"}, consentFiles: map[string]bool{"abc-consent.toml": false}, wantErr: true},
		"Get Global Missing Dir": {args: []string{"consent"}, consentFiles: map[string]bool{"consent.toml": false}, missingDir: true, wantGetDirErr: true, wantErr: true},
		"Get Global Bad File":    {args: []string{"consent"}, otherFiles: map[string]string{"consent.toml": "bad content"}, wantErr: true},
		"Get Global Bad Ext":     {args: []string{"consent"}, consentFiles: map[string]bool{"consent.txt": false}, wantErr: true},

		"Get Source Missing":     {args: []string{"consent", "abc"}, consentFiles: map[string]bool{"consent.toml": false}, wantErr: true},
		"Get Source Missing Dir": {args: []string{"consent", "abc"}, consentFiles: map[string]bool{"abc-consent.toml": false}, missingDir: true, wantGetDirErr: true, wantErr: true},

		"Set Global":             {args: []string{"consent", "--consent-state=true"}, consentFiles: map[string]bool{"consent.toml": false, "abc-consent.toml": false}},
		"Set Global Same":        {args: []string{"consent", "--consent-state=true"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": false}},
		"Set Source":             {args: []string{"consent", "abc", "--consent-state=true"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": false}},
		"Set Source Same":        {args: []string{"consent", "abc", "--consent-state=true"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": true}},
		"Set Source Multiple:":   {args: []string{"consent", "abc", "def", "-c=false"}, consentFiles: map[string]bool{"consent.toml": true, "abc-consent.toml": true, "def-consent.toml": false}},
		"Set Source Missing":     {args: []string{"consent", "abc", "--consent-state=true"}, consentFiles: map[string]bool{"consent.toml": false}},
		"Set Source Missing Dir": {args: []string{"consent", "abc", "--consent-state=true"}, consentFiles: map[string]bool{"abc-consent.toml": false}, missingDir: true},

		"Set Shorthand True": {args: []string{"consent", "-c=true"}, consentFiles: map[string]bool{"consent.toml": false, "abc-consent.toml": false}},

		"Bad Command":      {args: []string{"consent", "-unknown"}, wantUsageErr: true, wantErr: true},
		"Bad State":        {args: []string{"consent", "-c=bad"}, wantUsageErr: true, wantErr: true},
		"Set Dir No Perms": {args: []string{"consent", "-c=true"}, consentFiles: map[string]bool{"consent.toml": false, "abc-consent.toml": false}, dirNoPerms: true, wantGetDirErr: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			files := make(map[string][]byte, len(tc.otherFiles)+len(tc.consentFiles))
			for file, content := range tc.otherFiles {
				files[file] = []byte(content)
			}
			for file, state := range tc.consentFiles {
				data, err := toml.Marshal(consent.CFile{ConsentState: state})
				require.NoError(t, err, "Setup: could not marshal consent file")
				files[file] = data
			}

			app, configDir, _ := commands.NewForTests(t, commands.SetupConfig{MissingConfigDir: tc.missingDir, ConfigFiles: files}, tc.args...)

			if tc.dirNoPerms {
				// Remove write perms from the directory
				require.NoError(t, os.Chmod(configDir, 0400), "Setup: could not remove write perms from config dir")
				// #nosec G302 // configDir is a directory and should be allowed to be set to 0700
				t.Cleanup(func() { assert.NoError(t, os.Chmod(configDir, 0700), "Cleanup: could not restore config dir perms") })
			}

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
			if tc.wantGetDirErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected consent files state")
		},
		)
	}
}
