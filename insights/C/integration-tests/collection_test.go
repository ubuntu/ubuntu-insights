package libinsights_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestCollect(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState      *bool // nil = unknown/default
		usePlatformSource bool
		extraArgs         []string
		sourceMetrics     map[string]any
		sourceMetricsRaw  string
		sourceMetricsPath string
		wantErr           bool
	}{
		"DryRun with Consent": {
			consentState: boolPtr(true),
			extraArgs:    []string{"--dry-run"},
		},
		"Collect with Consent": {
			consentState: boolPtr(true),
		},
		"Collect OptOut": {
			consentState: boolPtr(false),
		},
		"System Collect with Consent": {
			consentState:      boolPtr(true),
			usePlatformSource: true,
		},
		"Source Metrics Valid JSON": {
			consentState: boolPtr(true),
			sourceMetrics: map[string]any{
				"key": "value",
			},
		},
		"Source Metrics Invalid JSON": {
			consentState:     boolPtr(true),
			sourceMetricsRaw: "{invalid-json",
			wantErr:          true,
		},
		"Source Metrics Non-existent File": {
			consentState:      boolPtr(true),
			sourceMetricsPath: "/non-existent/path/metrics.json",
			wantErr:           true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)
			fixture.printReport = true

			targetSource := fixture.source
			if tc.usePlatformSource {
				targetSource = ""
			}

			if tc.consentState != nil {
				_, err := runDriver(t, fixture, "set-consent", targetSource, fmt.Sprintf("%v", *tc.consentState))
				require.NoError(t, err)
			}

			args := []string{"collect", targetSource}
			args = append(args, tc.extraArgs...)

			// Prepare source metrics file if needed
			if tc.sourceMetrics != nil {
				f := filepath.Join(t.TempDir(), "metrics.json")
				data, err := json.Marshal(tc.sourceMetrics)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f, data, 0600))
				args = append(args, "--source-metrics", f)
			} else if tc.sourceMetricsRaw != "" {
				f := filepath.Join(t.TempDir(), "metrics.json")
				require.NoError(t, os.WriteFile(f, []byte(tc.sourceMetricsRaw), 0600))
				args = append(args, "--source-metrics", f)
			} else if tc.sourceMetricsPath != "" {
				args = append(args, "--source-metrics", tc.sourceMetricsPath)
			}

			out, err := runDriver(t, fixture, args...)

			if tc.wantErr {
				require.Error(t, err, "Expected collect to fail but it succeeded")
				return
			}
			require.NoError(t, err, "Failed to collect: %s", out)

			got := validateReports(t, fixture.insightsDir)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collected reports do not match expected")
		})
	}
}

func TestCompile(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState      *bool
		sourceMetrics     map[string]any
		sourceMetricsRaw  string
		sourceMetricsPath string
		wantErr           bool
	}{
		"Compile with Consent": {
			consentState: boolPtr(true),
		},
		"Compile OptOut": {
			consentState: boolPtr(false),
		},
		"Source Metrics Valid JSON": {
			consentState: boolPtr(true),
			sourceMetrics: map[string]any{
				"key": "value",
			},
		},
		"Source Metrics Invalid JSON": {
			consentState:     boolPtr(true),
			sourceMetricsRaw: "{invalid-json",
			wantErr:          true,
		},
		"Source Metrics Non-existent File": {
			consentState:      boolPtr(true),
			sourceMetricsPath: "/non-existent/path/metrics.json",
			wantErr:           true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)
			fixture.printReport = true

			if tc.consentState != nil {
				_, err := runDriver(t, fixture, "set-consent", fixture.source, fmt.Sprintf("%v", *tc.consentState))
				require.NoError(t, err)
			}

			args := []string{"compile"}

			if tc.sourceMetrics != nil {
				f := filepath.Join(t.TempDir(), "metrics.json")
				data, err := json.Marshal(tc.sourceMetrics)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(f, data, 0600))
				args = append(args, "--source-metrics", f)
			} else if tc.sourceMetricsRaw != "" {
				f := filepath.Join(t.TempDir(), "metrics.json")
				require.NoError(t, os.WriteFile(f, []byte(tc.sourceMetricsRaw), 0600))
				args = append(args, "--source-metrics", f)
			} else if tc.sourceMetricsPath != "" {
				args = append(args, "--source-metrics", tc.sourceMetricsPath)
			}

			out, err := runDriver(t, fixture, args...)

			if tc.wantErr {
				require.Error(t, err, "Expected compile to fail but it succeeded")
				return
			}
			require.NoError(t, err, "Failed to compile: %s", out)

			// Expect some metric content or at least valid JSON
			assert.NoError(t, json.Unmarshal([]byte(out), &map[string]any{}))

			// Compile ignores consent state since it doesn't write to disk.
			assert.NotContains(t, out, `"OptOut":true`)
		})
	}
}

func TestCompileAndWrite(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState bool
	}{
		"With Consent":    {consentState: true},
		"Without Consent": {consentState: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)
			fixture.printReport = true

			_, err := runDriver(t, fixture, "set-consent", fixture.source, fmt.Sprintf("%v", tc.consentState))
			require.NoError(t, err)

			out, err := runDriver(t, fixture, "compile")
			require.NoError(t, err)
			reportContent := out
			t.Logf("Compile output: %s", reportContent)

			tmpReport := filepath.Join(t.TempDir(), "report.json")
			err = os.WriteFile(tmpReport, []byte(reportContent), 0600)
			require.NoError(t, err)

			// Write command
			out, err = runDriver(t, fixture, "write", fixture.source, tmpReport)
			require.NoError(t, err, "Write failed details: %s", out)

			got := validateReports(t, fixture.insightsDir)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Compiled reports do not match expected")
		})
	}
}
