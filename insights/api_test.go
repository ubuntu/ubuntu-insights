// Package insights_test tests for Golang bindings.
package insights_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config insights.Config
	}{
		"Default config": {
			config: insights.Config{},
		},
		"Custom config": {
			config: insights.Config{
				ConsentDir:      "custom_consent_dir",
				InsightsDir:     "custom_insights_dir",
				SystemConfigDir: "custom_system_config_dir",
				Logger:          slog.Default(),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Resolve the config
			resolved := tc.config.Resolve()

			// Assert the resolved config
			assert.NotEmpty(t, resolved.ConsentDir)
			assert.NotEmpty(t, resolved.InsightsDir)
			assert.NotEmpty(t, resolved.SystemConfigDir)
			assert.NotNil(t, resolved.Logger)
		})
	}
}

// TestCollect tests the Collect insights.
func TestCollect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source       string
		collectFlags insights.CollectFlags

		wantErr bool
	}{
		"Source without metrics doesn't error": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				DryRun: true,
			},
		},
		"Source with metrics doesn't error": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				SourceMetricsPath: "custom.json",
				DryRun:            true,
			},
		},
		"Source with valid source metrics JSON doesn't error": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				SourceMetricsJSON: []byte(`{"key": "source metrics JSON"}`),
				DryRun:            true,
			},
		},
		"Returns report even if consent is false": {
			source: "valid_false",
			collectFlags: insights.CollectFlags{
				DryRun: true,
			},
		},
		"Missing consent file does not error in dry run": {
			source: "missing_consent_file",
			collectFlags: insights.CollectFlags{
				DryRun: true,
			},
		},

		// Error cases
		"Invalid source metrics JSON errors": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				SourceMetricsJSON: []byte(`{"key": "invalid source metrics JSON"`),
				DryRun:            true,
			},
			wantErr: true,
		},
		"Non-JSON object source metrics JSON errors": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				SourceMetricsJSON: []byte(`["array", "not", "object"]`),
				DryRun:            true,
			},
			wantErr: true,
		},
		"Setting both SourceMetricsPath and SourceMetricsJSON errors": {
			source: "valid_true",
			collectFlags: insights.CollectFlags{
				SourceMetricsPath: "custom.json",
				SourceMetricsJSON: []byte(`{"key": "source metrics JSON"}`),
				DryRun:            true,
			},
			wantErr: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				ConsentDir:  filepath.Join("testdata", "consent_files"),
				InsightsDir: dir,
			}

			if tc.collectFlags.SourceMetricsPath != "" {
				tc.collectFlags.SourceMetricsPath = filepath.Join("testdata", "metrics", tc.collectFlags.SourceMetricsPath)
			}

			// this is technically an integration test for dry-run.
			report, err := conf.Collect(tc.source, tc.collectFlags)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check the returned report
			mReport := collector.Insights{}
			err = json.Unmarshal(report, &mReport)
			require.NoError(t, err, "Failed to unmarshal report")
			assert.NotEmpty(t, mReport.InsightsVersion, "Insights version should not be empty")

			if tc.collectFlags.SourceMetricsJSON != nil || tc.collectFlags.SourceMetricsPath != "" {
				assert.NotEmpty(t, mReport.SourceMetrics, "Source metrics should not be empty")
			} else {
				assert.Empty(t, mReport.SourceMetrics, "Source metrics should be empty when not provided")
			}

			// test that dry run was applied.
			assert.NoDirExists(t, filepath.Join(dir, tc.source, "local"))
			assert.NoDirExists(t, filepath.Join(dir, tc.source, "uploaded"))
		})
	}
}

func TestCompile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		compileFlags insights.CompileFlags

		wantErr bool
	}{
		"Without source metrics": {},
		"With valid JSON source metrics": {
			compileFlags: insights.CompileFlags{
				SourceMetricsJSON: []byte(`{"key": "source metrics JSON"}`),
			},
		},
		"With valid source metrics path": {
			compileFlags: insights.CompileFlags{
				SourceMetricsPath: "custom.json",
			},
		},

		// Error cases
		"Errors with both SourceMetricsJSON and SourceMetricsPath": {
			compileFlags: insights.CompileFlags{
				SourceMetricsJSON: []byte(`{"key": "source metrics JSON"}`),
				SourceMetricsPath: "custom.json",
			},
			wantErr: true,
		},
		"Errors with invalid JSON source metrics": {
			compileFlags: insights.CompileFlags{
				SourceMetricsJSON: []byte(`{"key": "invalid metrics JSON"`),
			},
			wantErr: true,
		},
		"Errors with non-object source metrics JSON": {
			compileFlags: insights.CompileFlags{
				SourceMetricsJSON: []byte(`["array", "not", "object"]`),
			},
			wantErr: true,
		},
		"Errors with invalid source metrics path": {
			compileFlags: insights.CompileFlags{
				SourceMetricsPath: "invalid.json",
			},
			wantErr: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.compileFlags.SourceMetricsPath != "" {
				tc.compileFlags.SourceMetricsPath = filepath.Join("testdata", "metrics", tc.compileFlags.SourceMetricsPath)
			}

			// Compile doesn't use anything but the logger.
			report, err := insights.Config{}.Compile(tc.compileFlags)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check the returned report
			mReport := collector.Insights{}
			err = json.Unmarshal(report, &mReport)
			require.NoError(t, err, "Failed to unmarshal report")
			assert.NotEmpty(t, mReport.InsightsVersion, "Insights version should not be empty")

			if tc.compileFlags.SourceMetricsJSON != nil || tc.compileFlags.SourceMetricsPath != "" {
				assert.NotEmpty(t, mReport.SourceMetrics, "Source metrics should not be empty")
			} else {
				assert.Empty(t, mReport.SourceMetrics, "Source metrics should be empty when not provided")
			}
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string
		report []byte

		wantErr bool
	}{
		"Valid source and empty insights doesn't error": {
			source: "valid_true",
			report: []byte(`{}`),
		},
		"Valid source and insights doesn't error": {
			source: "valid_true",
			report: []byte(`{"insightsVersion": "1.0.0", "sourceMetrics": {"inner": "value"}}`),
		},

		// Error cases
		"Invalid source errors": {
			source:  "invalid",
			report:  []byte(`{}`),
			wantErr: true,
		},
		"Nil insights errors": {
			source:  "valid_true",
			report:  nil,
			wantErr: true,
		},
		"Unexpect insights fields errors": {
			source:  "valid_true",
			report:  []byte(`{"insightsVersion": "1.0.0", "sourceMetrics": {"inner": "value"}, "unexpectedField": "value"}`),
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				ConsentDir:  filepath.Join("testdata", "consent_files"),
				InsightsDir: dir,
			}

			err := conf.Write(tc.source, tc.report, insights.WriteFlags{DryRun: true})
			if tc.wantErr {
				require.Error(t, err, "Expected error from write but got none")
				return
			}
			require.NoError(t, err, "Got unexpected error from write")

			// test that dry run was applied.
			assert.NoDirExists(t, filepath.Join(dir, tc.source, "local"))
			assert.NoDirExists(t, filepath.Join(dir, tc.source, "uploaded"))
		})
	}
}

// TestUpload tests the Upload insights.
func TestUpload(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources []string

		wantErr bool
	}{
		"Valid source doesn't error": {
			sources: []string{"valid_true"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				ConsentDir:  filepath.Join("testdata", "consent_files"),
				InsightsDir: dir,
			}

			flags := insights.UploadFlags{
				MinAge: 0,
				Force:  false,
				DryRun: true,
			}

			if tc.sources == nil {
				tc.sources = []string{}
			}

			// this is technically an integration test for dry-run.
			err := conf.Upload(tc.sources, flags)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// test that dry run was applied.
			for _, source := range tc.sources {
				f, err := os.Open(filepath.Join(dir, source, "uploaded"))
				require.NoError(t, err, "Setup: failed to open temp directory")
				defer f.Close()

				_, err = f.Readdir(1)
				assert.ErrorIs(t, err, io.EOF)
			}
		})
	}
}

// TestGetConsentState tests the GetConsentState insights.
func TestGetConsentState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string

		expected bool
		wantErr  bool
	}{
		"True consent returns CONSENT_TRUE": {
			source:   "valid_true",
			expected: true,
		},

		"False consent returns CONSENT_FALSE": {
			source:   "valid_false",
			expected: false,
		},

		"Missing consent returns CONSENT_UNKNOWN": {
			source:  "missing_consent_file",
			wantErr: true, // missing consent file should return an error
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			conf := insights.Config{
				ConsentDir: filepath.Join("testdata", "consent_files"),
			}

			got, err := conf.GetConsentState(tc.source)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestSetConsentState tests the SetConsentState insights.
func TestSetConsentState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string
		state  bool

		wantErr bool
	}{
		"True consent sets CONSENT_TRUE": {
			source: "true",
			state:  true,
		},

		"False consent sets CONSENT_FALSE": {
			source: "false",
			state:  false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				ConsentDir:  dir,
				InsightsDir: t.TempDir(),
			}

			// this is technically an integration test.
			err := conf.SetConsentState(tc.source, tc.state)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			state, err := conf.GetConsentState(tc.source)
			require.NoError(t, err, "Failed to get consent state after setting it")
			assert.Equal(t, tc.state, state)
		})
	}
}

// TestIsSystemOptOut tests the IsSystemOptOut API method.
func TestIsSystemOptOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		// When systemOptOutFile is set, the file is copied into the temp dir as system-config.toml.
		// When empty, the directory exists but contains no config file.
		systemOptOutFile string

		want    bool
		wantErr bool
	}{
		"No config file returns false": {},
		"Opted-out config returns true": {
			systemOptOutFile: "opted_out-system-config.toml",
			want:             true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			if tc.systemOptOutFile != "" {
				src := filepath.Join("testdata", "system_config", tc.systemOptOutFile)
				dst := filepath.Join(dir, "system-config.toml")
				data, err := os.ReadFile(src)
				require.NoError(t, err, "Setup: failed to read system config fixture")
				require.NoError(t, os.WriteFile(dst, data, 0600), "Setup: failed to write system config file") //nolint:gosec // test fixture, path is t.TempDir()
			}

			conf := insights.Config{SystemConfigDir: dir}
			got, err := conf.IsSystemOptOut()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestSetSystemOptOut tests the SetSystemOptOut API method.
func TestSetSystemOptOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setState bool
	}{
		"Set opt-out true":  {setState: true},
		"Set opt-out false": {setState: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			conf := insights.Config{SystemConfigDir: dir}

			err := conf.SetSystemOptOut(tc.setState)
			require.NoError(t, err)

			got, err := conf.IsSystemOptOut()
			require.NoError(t, err, "Failed to read system opt-out after setting it")
			assert.Equal(t, tc.setState, got)
		})
	}
}

// TestCollectSystemOptOut verifies that when the system opt-out is active,
// Collect writes an opt-out report regardless of per-source consent.
func TestCollectSystemOptOut(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source       string
		systemOptOut bool
	}{
		"System opted-out, user consent true writes opt-out report": {
			source:       "valid_true",
			systemOptOut: true,
		},
		"System not opted-out, user consent true writes full report": {
			source:       "valid_true",
			systemOptOut: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			insightsDir := t.TempDir()
			systemConfigDir := t.TempDir()

			if tc.systemOptOut {
				err := insights.Config{SystemConfigDir: systemConfigDir}.SetSystemOptOut(true)
				require.NoError(t, err, "Setup: failed to set system opt-out")
			}

			conf := insights.Config{
				ConsentDir:      filepath.Join("testdata", "consent_files"),
				InsightsDir:     insightsDir,
				SystemConfigDir: systemConfigDir,
			}

			_, err := conf.Collect(tc.source, insights.CollectFlags{})
			require.NoError(t, err)

			// Verify a report was written to the local folder.
			localDir := filepath.Join(insightsDir, tc.source, "local")
			entries, err := os.ReadDir(localDir)
			require.NoError(t, err, "local reports directory should exist")
			require.Len(t, entries, 1, "exactly one report should be written")

			data, err := os.ReadFile(filepath.Join(localDir, entries[0].Name()))
			require.NoError(t, err)

			if tc.systemOptOut {
				assert.Contains(t, string(data), `"OptOut":true`, "system opted-out report should contain opt-out payload")
			} else {
				assert.NotContains(t, string(data), `"OptOut":true`, "non-opted-out report should not contain opt-out payload")
			}
		})
	}
}
