package commands_test

import (
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
		"Not specifying a source gets all sources": {
			args: []string{"upload"},
		},
		"Sets source to True": {
			args: []string{"upload", "True"},
		},
		"Sets source to False": {
			args: []string{"upload", "False"},
		},
		"Sets source to Consent": {
			args: []string{"upload", "Unknown"},
		},
		"Sets sources to True and Bad-Key": {
			args: []string{"upload", "True", "Bad-Key"},
		},
		"Passes min-age flag": {
			args: []string{"upload", "--min-age=1000000"},
		},
		"Passes force flag": {
			args: []string{"upload", "--force"},
		},
		"Passes dry-run flag": {
			args: []string{"upload", "--dry-run"},
		},
		"Passes backoff-retry flag": {
			args: []string{"upload", "--backoff-retry"},
		},

		// Error cases
		"Usage error when passing an unknown flag": {
			args:         []string{"upload", "--unknown"},
			wantUsageErr: true,
			wantErr:      true,
		},
		"Usage error when non-uint passed through the min-age flag": {
			args:         []string{"upload", "--min-age=bad"},
			wantUsageErr: true,
			wantErr:      true,
		},
		"Errors when min-age is set to a value that would overflow": {
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
			var (
				gotMinAge       uint
				dRun            bool
				gotBackoffRetry bool
			)
			newUploader := func(cm uploader.Consent, cachePath, source string, minAge uint, dryRun, backoffRetry bool, args ...uploader.Options) (uploader.Uploader, error) {
				gotSources = append(gotSources, source)
				gotMinAge = minAge
				dRun = dryRun
				gotBackoffRetry = backoffRetry

				return uploader.New(cm, cachePath, source, minAge, true, backoffRetry, args...)
			}
			a, _, _ := commands.NewAppForTests(t, tc.args, tc.consentDir, commands.WithNewUploader(newUploader))
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
				Sources  []string
				MinAge   uint
				DryRun   bool
				ExpRetry bool
			}

			got := results{
				Sources:  gotSources,
				MinAge:   gotMinAge,
				ExpRetry: gotBackoffRetry,
				DryRun:   dRun,
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected upload command state")
		})
	}
}
