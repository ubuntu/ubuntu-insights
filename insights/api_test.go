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
				ConsentDir:  "custom_consent_dir",
				InsightsDir: "custom_insights_dir",
				Logger:      slog.Default(),
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

		// Error cases
		"Missing consent file errors": {
			source: "missing_consent_file",
			collectFlags: insights.CollectFlags{
				DryRun: true,
			},

			wantErr: true,
		},
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
			f, err := os.Open(filepath.Join(dir, tc.source, "local"))
			require.NoError(t, err, "Setup: failed to open temp directory")
			defer f.Close()

			_, err = f.Readdir(1)
			assert.ErrorIs(t, err, io.EOF)
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
