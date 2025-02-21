package commands_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestCollect(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		args []string

		consentDir     string
		removeFiles    []string
		platformSource bool

		wantErr      bool
		wantUsageErr bool
	}{
		"Collect Basic": {
			args:           []string{"collect"},
			platformSource: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentDir == "" {
				tc.consentDir = "true-global"
			}

			var (
				gotCachePath string
				gotSource    string
				gotPeriod    uint
				gotDryRun    bool
			)
			newCollector := func(cm collector.ConsentManager, cachePath, source string, period uint, dryRun bool, args ...collector.Options) (collector.Collector, error) {
				gotCachePath = cachePath
				gotSource = source
				gotPeriod = period
				gotDryRun = dryRun

				return collector.New(cm, cachePath, source, period, true, args...)
			}

			a, _, cachePath := commands.NewAppForTests(t, tc.args, tc.consentDir, commands.WithNewCollector(newCollector))
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

			if tc.platformSource {
				assert.Equal(t, runtime.GOOS, gotSource)
				gotSource = "platform"
			}
			assert.Equal(t, cachePath, gotCachePath, "Cache path passed to app is not as expected")

			got := struct {
				Source string
				Period uint
				DryRun bool
			}{
				Source: gotSource,
				Period: gotPeriod,
				DryRun: gotDryRun,
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected collect command state")
		})
	}
}
