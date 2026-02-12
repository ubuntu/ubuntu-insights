package insights_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
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
		consentFixture consentFixture
		readOnlyFile   []string

		ignoreGolden bool
		wantExitCode int
	}{
		// Get platform source
		"Get platform true state":                  {consentFixture: fixtureTrue},
		"Get platform false state":                 {consentFixture: fixtureFalse},
		"Get platform empty state":                 {consentFixture: fixtureEmpty},
		"Get platform bad value returns error":     {consentFixture: fixtureBadValue, wantExitCode: 1},
		"Get platform bad key":                     {consentFixture: fixtureBadKey},
		"Get platform bad file returns error":      {consentFixture: fixtureBadFile, wantExitCode: 1},
		"Get platform bad extension returns error": {consentFixture: fixtureBadExt, wantExitCode: 1},

		// Get specific source
		"Get specific true state":                    {config: "true.yaml"},
		"Get specific multiple states":               {config: "multiple.yaml"},
		"Get specific bad extension returns error":   {config: "bad-ext.yaml", wantExitCode: 1},
		"Get specific bad file returns error":        {config: "bad-file.yaml", wantExitCode: 1},
		"Get specific bad key":                       {config: "bad-key.yaml"},
		"Get specific bad value returns error":       {config: "bad-value.yaml", wantExitCode: 1},
		"Get specific empty file returns error":      {config: "empty.yaml", wantExitCode: 1},
		"Get specific missing file returns error":    {config: "missing.yaml", wantExitCode: 1},
		"Get specific multiple errors returns error": {config: "multiple-err.yaml", wantExitCode: 1},
		"Get specific multiple mixed returns error":  {config: "multiple-mixed.yaml", wantExitCode: 1},

		// Set platform source
		"Set platform true from true state":    {state: "true", consentFixture: fixtureTrue},
		"Set platform true from false state":   {state: "true", consentFixture: fixtureFalse},
		"Set platform true from empty state":   {state: "true", consentFixture: fixtureEmpty},
		"Set platform true from bad value":     {state: "true", consentFixture: fixtureBadValue},
		"Set platform true from bad key":       {state: "true", consentFixture: fixtureBadKey},
		"Set platform true from bad file":      {state: "true", consentFixture: fixtureBadFile},
		"Set platform true from bad extension": {state: "true", consentFixture: fixtureBadExt},

		"Set platform false from true state":    {state: "false", consentFixture: fixtureTrue},
		"Set platform false from false state":   {state: "false", consentFixture: fixtureFalse},
		"Set platform false from empty state":   {state: "false", consentFixture: fixtureEmpty},
		"Set platform false from bad value":     {state: "false", consentFixture: fixtureBadValue},
		"Set platform false from bad key":       {state: "false", consentFixture: fixtureBadKey},
		"Set platform false from bad file":      {state: "false", consentFixture: fixtureBadFile},
		"Set platform false from bad extension": {state: "false", consentFixture: fixtureBadExt},

		"Set platform invalid from true state returns usage error":  {state: "invalid", consentFixture: fixtureTrue, wantExitCode: 2},
		"Set platform invalid from false state returns usage error": {state: "invalid", consentFixture: fixtureFalse, wantExitCode: 2},
		"Set platform invalid from empty state returns usage error": {state: "invalid", consentFixture: fixtureEmpty, wantExitCode: 2},

		// Set specific source
		"Set specific true from true state":     {state: "true", config: "true.yaml"},
		"Set specific true from false state":    {state: "true", config: "false.yaml"},
		"Set specific true from multiple mixed": {state: "true", config: "multiple-mixed.yaml"},
		"Set specific true from missing file":   {state: "true", config: "missing.yaml"},

		"Set specific false from true state":     {state: "false", config: "true.yaml"},
		"Set specific false from false state":    {state: "false", config: "false.yaml"},
		"Set specific false from multiple mixed": {state: "false", config: "multiple-mixed.yaml"},

		"Set specific invalid from true state returns usage error":     {state: "invalid", config: "true.yaml", wantExitCode: 2},
		"Set specific invalid from false state returns usage error":    {state: "invalid", config: "false.yaml", wantExitCode: 2},
		"Set specific invalid from multiple mixed returns usage error": {state: "invalid", config: "multiple-mixed.yaml", wantExitCode: 2},

		// Set Read Only
		"Set platform against read only file returns error": {
			state: "false", consentFixture: fixtureTrue, readOnlyFile: []string{constants.PlatformConsentFile},
			ignoreGolden: runtime.GOOS != "windows",
			wantExitCode: readOnlyErrorCode()},
		"Set specific against read only file returns error": {
			state: "false", config: "true.yaml", consentFixture: fixtureTrue, readOnlyFile: []string{"True-consent.toml"},
			ignoreGolden: runtime.GOOS != "windows",
			wantExitCode: readOnlyErrorCode()},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.config == "" {
				tc.config = "default.yaml"
			}

			tc.config = filepath.Join("testdata", "configs", "consent", tc.config)
			paths := setupFixtures(t, tc.consentFixture)

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
