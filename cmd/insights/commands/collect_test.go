package commands_test

import (
	"fmt"
	"math"
	"math/big"
	"path/filepath"
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

	overflowInt := big.NewInt(math.MaxInt)
	overflowInt.Add(overflowInt, big.NewInt(1))

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
		}, "Collect Source no Metrics": {
			args:         []string{"collect", "source"},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect source normal": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json")},
		}, "Collect source normal, period": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10"},
		}, "Collect source normal, dry-run": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--dry-run"},
		}, "Collect source normal, period, dry-run": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10", "--dry-run"},
		}, "Collect source normal, period, dry-run, force": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10", "--dry-run", "--force"},
		}, "Collect source dir": {
			args:         []string{"collect", "source", filepath.Join("testdata", "source_metrics")},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect source invalid path": {
			args:         []string{"collect", "source", "invalid-path"},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect source invalid JSON": {
			args:    []string{"collect", "source", filepath.Join("testdata", "source_metrics", "invalid.json")},
			wantErr: true,
		}, "Collect bad flag": {
			args:         []string{"collect", "--bad-flag"},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect period not int": {
			args:         []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=not-int"},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect period negative": {
			args:         []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=-1"},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect period overflow": {
			args:           []string{"collect", fmt.Sprintf("--period=%s", overflowInt.String())},
			platformSource: true,
			wantErr:        true,
		}, "Collect dry run, verbose 1": {
			args:           []string{"collect", "--dry-run", "-v"},
			platformSource: true,
		}, "Collect dry run, verbose 2": {
			args:           []string{"collect", "--dry-run", "-vv"},
			platformSource: true,
		}, "Collect False-consent source": {
			args: []string{"collect", "False", filepath.Join("testdata", "source_metrics", "normal.json")},
		}, "Collect Bad-File-consent source": {
			args: []string{"collect", "Bad-File", filepath.Join("testdata", "source_metrics", "normal.json")},
		}, "Collect Bad-File consent global source": {
			args:       []string{"collect", "Bad-File", filepath.Join("testdata", "source_metrics", "normal.json")},
			consentDir: "bad-file-global",
			wantErr:    true,
		}, "Collect nArgs 3": {
			args:         []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "extra-arg"},
			wantErr:      true,
			wantUsageErr: true,
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
			newCollector := func(cm collector.Consent, cachePath, source string, period uint, dryRun bool, args ...collector.Options) (collector.Collector, error) {
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
				return
			}
			require.False(t, a.UsageError())

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
