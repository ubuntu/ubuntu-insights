package insights_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestConsent(t *testing.T) {
	t.Parallel()

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
		// Get Default
		"Get Default True":      {config: "default.yaml", consentFixture: "true"},
		"Get Default False":     {config: "default.yaml", consentFixture: "false"},
		"Get Default Empty":     {config: "default.yaml", consentFixture: "empty"},
		"Get Default Bad-Value": {config: "default.yaml", consentFixture: "bad-value", wantExitCode: 1},
		"Get Default Bad-Key":   {config: "default.yaml", consentFixture: "bad-key"},
		"Get Default Bad-File":  {config: "default.yaml", consentFixture: "bad-file", wantExitCode: 1},
		"Get Default Bad-Ext":   {config: "default.yaml", consentFixture: "bad-ext", wantExitCode: 1},

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

		// Set Default
		"Set Default True T":      {state: "true", config: "default.yaml", consentFixture: "true"},
		"Set Default False T":     {state: "true", config: "default.yaml", consentFixture: "false"},
		"Set Default Empty T":     {state: "true", config: "default.yaml", consentFixture: "empty"},
		"Set Default Bad-Value T": {state: "true", config: "default.yaml", consentFixture: "bad-value"},
		"Set Default Bad-Key T":   {state: "true", config: "default.yaml", consentFixture: "bad-key"},
		"Set Default Bad-File T":  {state: "true", config: "default.yaml", consentFixture: "bad-file"},
		"Set Default Bad-Ext T":   {state: "true", config: "default.yaml", consentFixture: "bad-ext"},

		"Set Default True F":      {state: "false", config: "default.yaml", consentFixture: "true"},
		"Set Default False F":     {state: "false", config: "default.yaml", consentFixture: "false"},
		"Set Default Empty F":     {state: "false", config: "default.yaml", consentFixture: "empty"},
		"Set Default Bad-Value F": {state: "false", config: "default.yaml", consentFixture: "bad-value"},
		"Set Default Bad-Key F":   {state: "false", config: "default.yaml", consentFixture: "bad-key"},
		"Set Default Bad-File F":  {state: "false", config: "default.yaml", consentFixture: "bad-file"},
		"Set Default Bad-Ext F":   {state: "false", config: "default.yaml", consentFixture: "bad-ext"},

		"Set Default True Invalid":  {state: "invalid", config: "default.yaml", consentFixture: "true", wantExitCode: 2},
		"Set Default False Invalid": {state: "invalid", config: "default.yaml", consentFixture: "false", wantExitCode: 2},
		"Set Default Empty Invalid": {state: "invalid", config: "default.yaml", consentFixture: "empty", wantExitCode: 2},

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
		"Set Default Read Only": {
			state: "false", config: "default.yaml", consentFixture: "true", readOnlyFile: []string{"consent.toml"},
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
				cmd.Args = append(cmd.Args, "-s", tc.state)
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
