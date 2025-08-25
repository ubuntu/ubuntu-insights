package insights_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	const defaultTime = 50000

	tests := map[string]struct {
		source         string
		sourceMetrics  string
		config         string
		consentFixture string
		readOnlyFile   []string
		maxReports     uint
		time           int
		sysinfoErr     bool
		platformOnly   string

		useSysinfo           bool
		removeDefaultConsent bool

		skipReportCheck bool
		wantExitCode    int
	}{
		// Platform
		"Platform Linux": {
			platformOnly: "linux",
		},
		"Platform Windows": {
			platformOnly: "windows",
		},
		"Platform Darwin": {
			platformOnly: "darwin",
		},

		// True Consent
		"True Normal": {
			source:        "True",
			sourceMetrics: "normal.json",
		},
		"True Bad-Ext": {
			source:        "True",
			sourceMetrics: "bad_ext.txt",
		},
		"True Empty": {
			source:        "True",
			sourceMetrics: "empty.json",
			wantExitCode:  1,
		},
		"True Invalid": {
			source:        "True",
			sourceMetrics: "invalid.json",
			wantExitCode:  1,
		},
		"True Normal Default False": {
			source:         "True",
			sourceMetrics:  "normal.json",
			consentFixture: "false",
		},
		"True Normal Default Bad-file": {
			source:         "True",
			sourceMetrics:  "normal.json",
			consentFixture: "bad-file",
		},
		"True Normal Force": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "force.yaml",
		},
		"True Normal DryRun": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "dry.yaml",
		},
		"True Normal Period": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "period.yaml",
			wantExitCode:  1,
		},
		"True Normal DryRun Force": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "dry-force.yaml",
		},
		"True Normal DryRun Force Period": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "dry-force-period.yaml",
		},
		"True Normal Force Period": {
			source:        "True",
			sourceMetrics: "normal.json",
			config:        "force-period.yaml",
		},
		"True Normal MaxReports": {
			source:        "True",
			sourceMetrics: "normal.json",
			maxReports:    2,
		},
		"True Sysinfo Err": {
			source:        "True",
			sourceMetrics: "normal.json",
			sysinfoErr:    true,
			wantExitCode:  1,
		},
		"True No Metrics": {
			source:       "True",
			wantExitCode: 2,
		},

		// False Consent
		"False Normal": {
			source:        "False",
			sourceMetrics: "normal.json",
		},
		"False Invalid": {
			source:        "False",
			sourceMetrics: "invalid.json",
			wantExitCode:  1,
		},
		"False Normal MaxReports": {
			source:        "False",
			sourceMetrics: "normal.json",
			maxReports:    2,
		},
		"False No Metrics": {
			source:       "False",
			wantExitCode: 2,
		},

		// Unknown Consent
		"Unknown-A Normal": {
			source:        "Unknown-A",
			sourceMetrics: "normal.json",
		},
		"Unknown-A Invalid": {
			source:        "Unknown-A",
			sourceMetrics: "invalid.json",
			wantExitCode:  1,
		},
		"Unknown-A Normal Default False": {
			source:         "Unknown-A",
			sourceMetrics:  "normal.json",
			consentFixture: "false",
		},
		"Unknown-A Invalid Default False": {
			source:         "Unknown-A",
			sourceMetrics:  "invalid.json",
			consentFixture: "false",
			wantExitCode:   1,
		},
		"Unknown-A Normal Default Bad-file": {
			source:         "Unknown-A",
			sourceMetrics:  "normal.json",
			consentFixture: "bad-file",
			wantExitCode:   1,
		},

		// SysInfo Tests
		"Platform SysInfo": {
			useSysinfo:      true,
			skipReportCheck: true,
		},
		"True SysInfo": {
			source:          "True",
			sourceMetrics:   "normal.json",
			skipReportCheck: true,
			useSysinfo:      true,
		},
		"True SysInfo No Metrics": {
			source:       "True",
			useSysinfo:   true,
			wantExitCode: 2,
		},
		"False SysInfo": {
			source:        "False",
			sourceMetrics: "normal.json",
			useSysinfo:    true,
		},

		"Exit 0 no metrics if unable to read source consent and no default consent file": {
			source:               "Unknown-B",
			sourceMetrics:        "normal.json",
			useSysinfo:           true,
			removeDefaultConsent: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.platformOnly != "" && tc.platformOnly != runtime.GOOS {
				t.Skipf("skipping test on %s", runtime.GOOS)
			}

			if tc.time == 0 {
				tc.time = defaultTime
			}

			if tc.consentFixture == "" {
				tc.consentFixture = defaultConsentFixture
			}
			paths := copyFixtures(t, tc.consentFixture)

			if tc.removeDefaultConsent {
				require.NoError(t, os.Remove(filepath.Join(paths.consent, "consent.toml")))
			}
			for _, f := range tc.readOnlyFile {
				testutils.MakeReadOnly(t, filepath.Join(paths.consent, f))
			}

			consentContents, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err)

			smContents, err := testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err)

			if tc.sourceMetrics != "" {
				tc.sourceMetrics = filepath.Join(paths.sourceMetrics, tc.sourceMetrics)
			}

			// #nosec:G204 - we control the command arguments in tests
			cmd := exec.Command(cliPath, "collect")
			if tc.source != "" {
				cmd.Args = append(cmd.Args, tc.source, tc.sourceMetrics)
			}
			if tc.config != "" {
				tc.config = filepath.Join("testdata", "configs", "collect", tc.config)
				cmd.Args = append(cmd.Args, "--config", tc.config)
			}
			cmd.Args = append(cmd.Args, "-vv")
			cmd.Args = append(cmd.Args, "--consent-dir", paths.consent)
			cmd.Args = append(cmd.Args, "--insights-dir", paths.reports)
			cmd.Env = append(cmd.Env, os.Environ()...)
			if tc.maxReports != 0 {
				cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_REPORTS="+fmt.Sprint(tc.maxReports))
			}
			if tc.time != 0 {
				cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_TIME="+fmt.Sprint(tc.time))
			}

			if !tc.useSysinfo {
				cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_SYSINFO=true")
				if tc.sysinfoErr {
					cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_SYSINFO_ERR=error")
				}
			}

			out, err := cmd.CombinedOutput()
			if tc.wantExitCode == 0 {
				require.NoError(t, err, "unexpected CLI error: %v\n%s", err, out)
			}
			assert.Equal(t, tc.wantExitCode, cmd.ProcessState.ExitCode(), "unexpected exit code: %v\n%s", err, out)

			// Check that the consent directory was not modified
			gotContents, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "failed to get consent directory contents")
			assert.Equal(t, consentContents, gotContents)

			// Check that the source metrics directory was not modified
			gotContents, err = testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "failed to get source metrics directory contents")
			assert.Equal(t, smContents, gotContents)

			if tc.skipReportCheck {
				return
			}

			got, err := testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "failed to get directory contents")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got)
		})
	}
}
