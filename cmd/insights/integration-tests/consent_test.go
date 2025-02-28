package insights_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestConsent(t *testing.T) {
	t.Parallel()
	const defaultConsentFixture = "true-global"

	readOnlyErrorCode := func() int {
		if runtime.GOOS == "windows" {
			return 1
		}
		return 0
	}

	tests := map[string]struct {
		state          string
		config         string
		consentFixture string
		readOnlyFile   []string

		ignoreGolden bool
		wantExitCode int
	}{
		// Get Global
		"Get Global True":      {config: "global.yaml", consentFixture: "true-global"},
		"Get Global False":     {config: "global.yaml", consentFixture: "false-global"},
		"Get Global Empty":     {config: "global.yaml", consentFixture: "empty-global"},
		"Get Global Bad-Value": {config: "global.yaml", consentFixture: "bad-value-global", wantExitCode: 1},
		"Get Global Bad-Key":   {config: "global.yaml", consentFixture: "bad-key-global"},
		"Get Global Bad-File":  {config: "global.yaml", consentFixture: "bad-file-global", wantExitCode: 1},
		"Get Global Bad-Ext":   {config: "global.yaml", consentFixture: "bad-ext-global", wantExitCode: 1},

		// Get Source
		"Get Source True":           {config: "true.yaml"},
		"Get Source Multiple":       {config: "multiple.yaml"},
		"Get Source Bad-Ext":        {config: "bad-ext.yaml", wantExitCode: 1},
		"Get Source Bad-File":       {config: "bad-file.yaml", wantExitCode: 1},
		"Get Source Bad-Key":        {config: "bad-key.yaml"},
		"Get Source Bad-Value":      {config: "bad-value.yaml", wantExitCode: 1},
		"Get Source Empty":          {config: "empty.yaml", wantExitCode: 1},
		"Get Source Missing":        {config: "missing.yaml", wantExitCode: 1},
		"Get Source Multiple Err":   {config: "multiple-err.yaml", wantExitCode: 1},
		"Get Source Multiple Mixed": {config: "multiple-mixed.yaml", wantExitCode: 1},

		// Set Global
		"Set Global True T":      {state: "true", config: "global.yaml", consentFixture: "true-global"},
		"Set Global False T":     {state: "true", config: "global.yaml", consentFixture: "false-global"},
		"Set Global Empty T":     {state: "true", config: "global.yaml", consentFixture: "empty-global"},
		"Set Global Bad-Value T": {state: "true", config: "global.yaml", consentFixture: "bad-value-global"},
		"Set Global Bad-Key T":   {state: "true", config: "global.yaml", consentFixture: "bad-key-global"},
		"Set Global Bad-File T":  {state: "true", config: "global.yaml", consentFixture: "bad-file-global"},
		"Set Global Bad-Ext T":   {state: "true", config: "global.yaml", consentFixture: "bad-ext-global"},

		"Set Global True F":      {state: "false", config: "global.yaml", consentFixture: "true-global"},
		"Set Global False F":     {state: "false", config: "global.yaml", consentFixture: "false-global"},
		"Set Global Empty F":     {state: "false", config: "global.yaml", consentFixture: "empty-global"},
		"Set Global Bad-Value F": {state: "false", config: "global.yaml", consentFixture: "bad-value-global"},
		"Set Global Bad-Key F":   {state: "false", config: "global.yaml", consentFixture: "bad-key-global"},
		"Set Global Bad-File F":  {state: "false", config: "global.yaml", consentFixture: "bad-file-global"},
		"Set Global Bad-Ext F":   {state: "false", config: "global.yaml", consentFixture: "bad-ext-global"},

		"Set Global True Invalid":  {state: "invalid", config: "global.yaml", consentFixture: "true-global", wantExitCode: 2},
		"Set Global False Invalid": {state: "invalid", config: "global.yaml", consentFixture: "false-global", wantExitCode: 2},
		"Set Global Empty Invalid": {state: "invalid", config: "global.yaml", consentFixture: "empty-global", wantExitCode: 2},

		// Set Source
		"Set Source True T":           {state: "true", config: "true.yaml"},
		"Set Source False T":          {state: "true", config: "false.yaml"},
		"Set Source Multiple Mixed T": {state: "true", config: "multiple-mixed.yaml"},
		"Set Source Missing T":        {state: "true", config: "missing.yaml"},

		"Set Source True F":           {state: "false", config: "true.yaml"},
		"Set Source False F":          {state: "false", config: "false.yaml"},
		"Set Source Multiple Mixed F": {state: "false", config: "multiple-mixed.yaml"},

		"Set Source True Invalid":           {state: "invalid", config: "true.yaml", wantExitCode: 2},
		"Set Source False Invalid":          {state: "invalid", config: "false.yaml", wantExitCode: 2},
		"Set Source Multiple Mixed Invalid": {state: "invalid", config: "multiple-mixed.yaml", wantExitCode: 2},

		// Set Read Only
		"Set Global Read Only": {
			state: "false", config: "global.yaml", consentFixture: "true-global", readOnlyFile: []string{"consent.toml"},
			ignoreGolden: runtime.GOOS != "windows",
			wantExitCode: readOnlyErrorCode()},
		"Set Source Read Only": {
			state: "false", config: "true.yaml", readOnlyFile: []string{"True-consent.toml"},
			ignoreGolden: runtime.GOOS != "windows",
			wantExitCode: readOnlyErrorCode()},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentFixture == "" {
				tc.consentFixture = defaultConsentFixture
			}
			tc.config = filepath.Join("testdata", "configs", "consent", tc.config)
			paths := copyFixtures(t, tc.consentFixture)

			for _, f := range tc.readOnlyFile {
				testutils.MakeReadOnly(t, filepath.Join(paths.consent, f))
			}

			reportsContents, err := testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "Setup: failed to get directory contents")
			smContents, err := testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "Setup: failed to get directory contents")

			// #nosec:G204 - we control the command arguments in tests
			cmd := exec.Command(cliPath, "consent", "--config", tc.config, "-vv")
			cmd.Args = append(cmd.Args, "--consent-dir", paths.consent)
			if tc.state != "" {
				cmd.Args = append(cmd.Args, "-c", tc.state)
			}
			out, err := cmd.CombinedOutput()
			if tc.wantExitCode == 0 {
				require.NoError(t, err, "unexpected CLI error: %v\n%s", err, out)
			}
			assert.Equal(t, tc.wantExitCode, cmd.ProcessState.ExitCode(), "unexpected exit code: %v\n%s", err, out)

			// Check that the reports and source-metrics directories were not modified
			got, err := testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "failed to get directory contents")
			assert.Equal(t, reportsContents, got)

			got, err = testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "failed to get directory contents")
			assert.Equal(t, smContents, got)

			got, err = testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "failed to get directory contents")

			if tc.ignoreGolden {
				return
			}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got)
		})
	}
}
