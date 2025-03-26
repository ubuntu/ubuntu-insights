// Package insights_test tests for Golang bindings.
package insights_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/pkg/insights"
)

// TestCollect tests the Collect API.
func TestCollect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		metricsPath string
		source      string

		wantErr bool
	}{
		"Source with metrics doesn't error": {
			metricsPath: "custom.json",
			source:      "valid_true",
		},

		"Source with no metrics errors": {
			metricsPath: "",
			source:      "missing_consent_file",

			wantErr: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				Source:      tc.source,
				ConsentDir:  filepath.Join("testdata", "consent_files"),
				InsightsDir: dir,
				Verbose:     false,
			}

			flags := insights.CollectFlags{
				Period: 0,
				Force:  false,
				DryRun: true,
			}

			mPath := ""
			if tc.metricsPath != "" {
				mPath = filepath.Join("testdata", "metrics", tc.metricsPath)
			}

			// this is technically an integration test for dry-run.
			err := conf.Collect(mPath, flags)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// test that dry run was applied.
			f, err := os.Open(filepath.Join(dir, tc.source, "local"))
			require.NoError(t, err, "Setup: failed to open temp directory")
			defer f.Close()

			_, err = f.Readdir(1)
			assert.ErrorIs(t, err, io.EOF)
		})
	}
}

// TestUpload tests the Upload API.
func TestUpload(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string

		wantErr bool
	}{
		"Valid source doesn't error": {
			source: "valid_true",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			conf := insights.Config{
				Source:      tc.source,
				ConsentDir:  filepath.Join("testdata", "consent_files"),
				InsightsDir: dir,
				Verbose:     false,
			}

			flags := insights.UploadFlags{
				MinAge: 0,
				Force:  false,
				DryRun: true,
			}

			// this is technically an integration test for dry-run.
			err := conf.Upload(flags)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// test that dry run was applied.
			f, err := os.Open(filepath.Join(dir, tc.source, "uploaded"))
			require.NoError(t, err, "Setup: failed to open temp directory")
			defer f.Close()

			_, err = f.Readdir(1)
			assert.ErrorIs(t, err, io.EOF)
		})
	}
}

// TestGetConsentState tests the GetConsentState API.
func TestGetConsentState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string

		expected insights.ConsentState
	}{
		"True consent returns CONSENT_TRUE": {
			source:   "valid_true",
			expected: insights.ConsentTrue,
		},

		"False consent returns CONSENT_FALSE": {
			source:   "valid_false",
			expected: insights.ConsentFalse,
		},

		"Missing consent returns CONSENT_UNKNOWN": {
			source:   "missing_consent_file",
			expected: insights.ConsentUnknown,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			conf := insights.Config{
				Source:     tc.source,
				ConsentDir: filepath.Join("testdata", "consent_files"),
			}

			got := conf.GetConsentState()

			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestSetConsentState tests the SetConsentState API.
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
				Source:      tc.source,
				ConsentDir:  dir,
				InsightsDir: t.TempDir(),
				Verbose:     false,
			}

			// this is technically an integration test.
			err := conf.SetConsentState(tc.state)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var want insights.ConsentState = insights.ConsentFalse
			if tc.state {
				want = insights.ConsentTrue
			}

			assert.Equal(t, want, conf.GetConsentState())
		})
	}
}
