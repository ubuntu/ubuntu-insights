package commands_test

import (
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	overflowInt := big.NewInt(math.MaxInt)
	overflowInt.Mul(overflowInt, overflowInt)

	tests := map[string]struct {
		args []string

		consentDir       string
		noDefaultConsent bool

		wantErr      bool
		wantUsageErr bool
	}{
		"Collect Basic": {
			args: []string{"collect"},
		},
		"Collect source normal": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json")},
		}, "Collect source normal, period": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10"},
		}, "Collect source normal, dry-run": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--dry-run"},
		}, "Collect source normal, period, dry-run": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10", "--dry-run"},
		}, "Collect source normal, period, dry-run, force": {
			args: []string{"collect", "source", filepath.Join("testdata", "source_metrics", "normal.json"), "--period=10", "--dry-run", "--force"},
		}, "Collect dry run, verbose 1": {
			args: []string{"collect", "--dry-run", "-v"},
		}, "Collect dry run, verbose 2": {
			args: []string{"collect", "--dry-run", "-vv"},
		}, "Collect False-consent source": {
			args: []string{"collect", "False", filepath.Join("testdata", "source_metrics", "normal.json")},
		}, "Collect Bad-File-consent source": {
			args: []string{"collect", "Bad-File", filepath.Join("testdata", "source_metrics", "normal.json")},
		},

		"Exit 0 when no consent files": {
			args:             []string{"collect", "Unknown", filepath.Join("testdata", "source_metrics", "normal.json")},
			noDefaultConsent: true,
		},

		// Error cases
		"Collect Source no Metrics": {
			args:         []string{"collect", "source"},
			wantErr:      true,
			wantUsageErr: true,
		},
		"Collect source dir": {
			args:         []string{"collect", "source", filepath.Join("testdata", "source_metrics")},
			wantErr:      true,
			wantUsageErr: true,
		}, "Collect source invalid path": {
			args:         []string{"collect", "source", "invalid-path"},
			wantErr:      true,
			wantUsageErr: true,
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
			args:         []string{"collect", fmt.Sprintf("--period=%s", overflowInt.String())},
			wantErr:      true,
			wantUsageErr: true,
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
				tc.consentDir = "true"
			}

			var gotConfig collector.Config
			mc := &mockCollector{}
			newCollector := func(l *slog.Logger, cm collector.Consent, c collector.Config, args ...collector.Options) (collector.Collector, error) {
				gotConfig = c

				return mc, nil
			}

			a, consentPath, cachePath := commands.NewAppForTests(t, tc.args, tc.consentDir, commands.WithNewCollector(newCollector))
			if tc.noDefaultConsent {
				require.NoError(t, os.Remove(filepath.Join(consentPath, "consent.toml")), "Setup: could not remove default consent file")
			}

			err := a.Run()
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, tc.wantUsageErr, a.UsageError(), "Unexpected usage error state")
				return
			}

			require.NoError(t, err)
			require.False(t, a.UsageError())

			assert.Equal(t, cachePath, gotConfig.CachePath, "Cache path passed to app is not as expected")

			got := struct {
				Source string
				Period uint32
				Force  bool
				DryRun bool
			}{
				Source: gotConfig.Source,
				Period: mc.gotPeriod,
				Force:  mc.gotForce,
				DryRun: mc.gotDryRun,
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Unexpected collect command state")
		})
	}
}

func TestCollectCollectorErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		compileErr error
		writeErr   error

		wantErr bool
	}{
		"No Errors": {},
		"Consent file not error does not return error": {
			writeErr: consent.ErrConsentFileNotFound,
		},

		"Compile Error": {
			compileErr: fmt.Errorf("compile error"),
			wantErr:    true,
		},
		"Write Error": {
			writeErr: fmt.Errorf("write error"),
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mc := &mockCollector{
				compileErr: tc.compileErr,
				writeErr:   tc.writeErr,
			}
			newCollector := func(l *slog.Logger, cm collector.Consent, c collector.Config, args ...collector.Options) (collector.Collector, error) {
				return mc, nil
			}

			a, _, _ := commands.NewAppForTests(t, []string{"collect"}, "true", commands.WithNewCollector(newCollector))
			err := a.Run()

			assert.False(t, a.UsageError(), "Expected no usage error")
			if tc.wantErr {
				require.Error(t, err, "Expected error but got none")
				return
			}

			require.NoError(t, err, "Unexpected error running collect command")
		})
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		newCollectorErr error
		wantErr         bool
		wantUsageErr    bool
	}{
		"No error": {},
		"Generic collector error": {
			newCollectorErr: fmt.Errorf("requested collector error"),
			wantErr:         true,
		},
		"collector.ErrSanitizeError": {
			newCollectorErr: collector.ErrSanitizeError,
			wantErr:         true,
			wantUsageErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mc := &mockCollector{
				compileErr: tc.newCollectorErr,
			}
			newCollector := func(l *slog.Logger, cm collector.Consent, c collector.Config, args ...collector.Options) (collector.Collector, error) {
				return mc, mc.compileErr
			}

			a, _, _ := commands.NewAppForTests(t, []string{"collect"}, "true", commands.WithNewCollector(newCollector))
			err := a.Run()

			assert.Equal(t, tc.wantUsageErr, a.UsageError(), "Unexpected usage error state")
			if tc.wantErr {
				require.Error(t, err, "Expected error but got none")
				return
			}
			require.NoError(t, err, "Unexpected error running collect command")
		})
	}
}

type mockCollector struct {
	compileErr error
	writeErr   error

	gotPeriod uint32
	gotForce  bool
	gotDryRun bool
}

func (m *mockCollector) Compile() (collector.Insights, error) {
	return collector.Insights{}, m.compileErr
}

func (m *mockCollector) Write(insights collector.Insights, period uint32, force, dryRun bool) error {
	m.gotPeriod = period
	m.gotForce = force
	m.gotDryRun = dryRun
	return m.writeErr
}
