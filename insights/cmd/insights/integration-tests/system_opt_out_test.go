package insights_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

func TestSystemOptOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state         string
		initialConfig string

		wantExitCode int
	}{
		"Get defaults to false when no config exists": {},
		"Get true state":                  {initialConfig: "system_opt_out = true\n"},
		"Get false state":                 {initialConfig: "system_opt_out = false\n"},
		"Get bad value returns error":     {initialConfig: "system_opt_out = bad\n", wantExitCode: 1},
		"Set true with no config":         {state: "true"},
		"Set false with no config":        {state: "false"},
		"Set true from false state":       {state: "true", initialConfig: "system_opt_out = false\n"},
		"Set false from true state":       {state: "false", initialConfig: "system_opt_out = true\n"},
		"Set invalid returns usage error": {state: "invalid", wantExitCode: 2},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			paths := setupFixtures(t, fixtureTrue)
			setupSystemConfig(t, paths.systemConfig, tc.initialConfig)

			consentContents, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "Setup: failed to get consent directory contents")
			reportsContents, err := testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "Setup: failed to get reports directory contents")
			smContents, err := testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "Setup: failed to get source metrics directory contents")

			// #nosec:G204 - we control the command arguments in tests
			cmd := exec.Command(cliPath, "system-opt-out", "--system-config-dir", paths.systemConfig, "-vv")
			if tc.state != "" {
				cmd.Args = append(cmd.Args, "-s", tc.state)
			}
			out, err := cmd.CombinedOutput()
			if tc.wantExitCode == 0 {
				require.NoError(t, err, "unexpected CLI error: %v\n%s", err, out)
			}
			assert.Equal(t, tc.wantExitCode, cmd.ProcessState.ExitCode(), "unexpected exit code: %v\n%s", err, out)

			got, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "failed to get consent directory contents")
			assert.Equal(t, consentContents, got)

			got, err = testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "failed to get reports directory contents")
			assert.Equal(t, reportsContents, got)

			got, err = testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "failed to get source metrics directory contents")
			assert.Equal(t, smContents, got)

			got, err = testutils.GetDirContents(t, paths.systemConfig, 3)
			require.NoError(t, err, "failed to get system config directory contents")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got)
		})
	}
}

func setupSystemConfig(t *testing.T, systemConfigDir, initialConfig string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(systemConfigDir, 0700), "Setup: failed to create system config directory")

	if initialConfig == "" {
		return
	}

	require.NoError(t, os.WriteFile(filepath.Join(systemConfigDir, constants.SystemConfigFileName), []byte(initialConfig), 0600), "Setup: failed to write system config file")
}
