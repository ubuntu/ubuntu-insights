package commands_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

func TestUpload(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args []string

		consentDir  string
		removeFiles []string

		wantErr      bool
		wantUsageErr bool
	}{
		"Upload All Sources": {
			args: []string{"upload"},
		},
		"Upload Source True": {
			args: []string{"upload", "True"},
		},
		"Upload Source False": {
			args: []string{"upload", "False"},
		},
		"Upload Source Unknown Consent": {
			args: []string{"upload", "Unknown"},
		},
		"Upload Source True, Bad-Key": {
			args: []string{"upload", "True", "Bad-Key"},
		},
		"Upload All High Min Age": {
			args: []string{"upload", "--min-age=1000000"},
		},
		"Upload All Sources, Force": {
			args: []string{"upload", "--force"},
		},
		"Upload All Sources, Bad Flag": {
			args:         []string{"upload", "--unknown"},
			wantUsageErr: true,
			wantErr:      true,
		},
		"Upload All Sources, Bad Min Age": {
			args:         []string{"upload", "--min-age=bad"},
			wantUsageErr: true,
			wantErr:      true,
		},
		"Min-Age Overflow": {
			args:    []string{"upload", "--min-age=18446744073709551615"},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentDir == "" {
				tc.consentDir = "true-global"
			}

			gotSources := make([]string, 0)
			var gotMinAge uint

			newUploader := func(cm uploader.ConsentManager, cachePath, source string, minAge uint, dryRun bool, args ...uploader.Options) (uploader.Uploader, error) {
				gotSources = append(gotSources, source)
				gotMinAge = minAge

				return uploader.New(cm, cachePath, source, minAge, true, args...)
			}
			a, _, _ := newAppUploaderForTests(t, tc.args, newUploader, tc.consentDir)
			err := a.Run()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantUsageErr {
				require.True(t, a.UsageError())
			} else {
				require.False(t, a.UsageError())
			}

			type results struct {
				Sources []string
				MinAge  uint
			}

			got := results{
				Sources: gotSources,
				MinAge:  gotMinAge,
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected upload command state")
		})
	}
}

func newAppUploaderForTests(t *testing.T, args []string, newUploader commands.NewUploader, consentDir string) (app *commands.App, consentPath, cachePath string) {
	t.Helper()

	cachePath = filepath.Join(t.TempDir())
	cacheDir := filepath.Join("testdata", "reports")
	require.NoError(t, testutils.CopyDir(t, cacheDir, cachePath), "Setup: could not copy cache dir")
	args = append(args, "--insights-dir", cachePath)

	consentPath = t.TempDir()
	consentDir = filepath.Join("testdata", "consents", consentDir)
	require.NoError(t, testutils.CopyDir(t, consentDir, consentPath), "Setup: could not copy consent dir")

	args = append(args, "--consent-dir", consentPath)

	app, err := commands.New(commands.WithNewUploader(newUploader))
	require.NoError(t, err, "Setup: could not create app")

	app.SetArgs(args)

	return app, consentPath, cachePath
}
