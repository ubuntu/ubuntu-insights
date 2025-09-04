package collector_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

var (
	cTrue    = testConsentChecker{consent: true}
	cFalse   = testConsentChecker{consent: false}
	cErr     = testConsentChecker{err: fmt.Errorf("consent error")}
	cErrTrue = testConsentChecker{consent: true, err: fmt.Errorf("consent error")}
)

type testConsentChecker struct {
	consent bool
	err     error
}

func (m testConsentChecker) HasConsent(source string) (bool, error) {
	return m.consent, m.err
}

type testSysInfo struct {
	info sysinfo.Info
	err  error
}

func (m testSysInfo) Collect() (sysinfo.Info, error) {
	return m.info, m.err
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config collector.Config

		logs    map[slog.Level]uint
		wantErr bool
	}{
		"Blank collector config": {
			config: collector.Config{},
			logs: map[slog.Level]uint{
				slog.LevelInfo: 2,
			},
		},

		"Custom source with sourceMetricsPath": {
			config: collector.Config{
				Source:            "customSource",
				SourceMetricsPath: "fakeSourceMetricsPath",
				CachePath:         "fakeCachePath",
			},
		},
		"Custom source with sourceMetricsJSON": {
			config: collector.Config{
				Source:            "customSource",
				SourceMetricsJSON: []byte(`{"test": "sourceMetricsJson"}`),
				CachePath:         "fakeCachePath",
			},
		},

		// Error cases
		"Both sourceMetricsPath and sourceMetricsJSON provided with customSource errors": {
			config: collector.Config{
				Source:            "customSource",
				SourceMetricsPath: "fakeSourceMetricsPath",
				SourceMetricsJSON: []byte(`{"test": "sourceMetricsJson"}`),
				CachePath:         "fakeCachePath",
			},
			wantErr: true,
		},
		"Invalid sourceMetricsJSON provided with customSource errors": {
			config: collector.Config{
				Source:            "customSource",
				SourceMetricsJSON: []byte(`{"test": "invalidSourceMetricsJson"`),
				CachePath:         "fakeCachePath",
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)
			err := tc.config.Sanitize(slog.New(&l))
			if tc.wantErr {
				require.Error(t, err, "SanitizeConfig should have returned an error")
			} else {
				require.NoError(t, err, "SanitizeConfig returned an unexpected error")
			}

			if !l.AssertLevels(t, tc.logs) {
				l.OutputLogs(t)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		consentM collector.Consent
		config   collector.Config

		nilConsent bool

		wantErr bool
	}{
		"Blank collector": {
			config: collector.Config{
				CachePath: t.TempDir(),
			},
		},
		"Empty Cache Path": {
			config: collector.Config{},
		},

		// Error cases
		"Nil Consent": {
			config: collector.Config{
				CachePath: t.TempDir(),
			},
			nilConsent: true,

			wantErr: true,
		},
		"Bad cache path errors": {
			config: collector.Config{
				CachePath: filepath.Join(t.TempDir(), "\x00invalid"),
			},

			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentM == nil && !tc.nilConsent {
				tc.consentM = cTrue
			}

			l := slog.New(slog.NewTextHandler(os.Stderr, nil))
			result, err := collector.New(l, tc.consentM, tc.config)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
		})
	}
}

func TestCompile(t *testing.T) {
	t.Parallel()

	const (
		mockTime   = 10
		maxReports = 5
	)

	// Note that config.Source is always set in the tests.
	tests := map[string]struct {
		consentM collector.Consent
		config   collector.Config

		time    int64
		sysInfo collector.SysInfo
		noDir   bool
		wantErr bool
	}{
		"Basic": {
			consentM: cTrue,
		},
		"With SourceMetrics": {
			config: collector.Config{
				SourceMetricsPath: "testdata/source_metrics/normal.json",
			},
			consentM: cTrue,
		},
		"With SourceMetrics JSON": {
			config: collector.Config{
				SourceMetricsJSON: []byte(`{"test": "sourceMetricsJson"}`),
			},
			consentM: cTrue,
		},
		"Consent False": {
			consentM: cFalse,
		},

		// Error cases
		"Non-existent source metrics file": {
			config: collector.Config{
				SourceMetricsPath: "testdata/source_metrics/nonexistent.json",
			},
			consentM: cTrue,
			wantErr:  true,
		},
		"Invalid source metrics file": {
			config: collector.Config{
				SourceMetricsPath: "testdata/source_metrics/invalid.json",
			},
			consentM: cTrue,
			wantErr:  true,
		},
		"Bad ext source metrics file": {
			config: collector.Config{
				SourceMetricsPath: "testdata/source_metrics/bad_ext.json",
			},
			consentM: cTrue,
			wantErr:  true,
		},
		"Non-json object sourceMetricsJSON errors": {
			config: collector.Config{
				SourceMetricsJSON: []byte(`123`),
			},
			consentM: cTrue,
			wantErr:  true,
		},
		"Empty source metrics file": {
			config: collector.Config{
				SourceMetricsPath: "testdata/source_metrics/empty.json",
			},
			consentM: cTrue,
			wantErr:  true,
		},
		"SysInfo Collect Error": {
			consentM: cTrue,
			sysInfo:  testSysInfo{info: sysinfo.Info{}, err: fmt.Errorf("sysinfo error")},
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.config.Source = "source"

			dir := t.TempDir()
			sDir := filepath.Join(dir, tc.config.Source)
			require.NoError(t, testutils.CopyDir(t, filepath.Join("testdata", "reports_cache"), sDir), "Setup: failed to copy reports cache")
			tc.config.CachePath = dir

			if tc.sysInfo == nil {
				tc.sysInfo = testSysInfo{info: sysinfo.Info{}, err: nil}
			}

			if tc.time == 0 {
				tc.time = mockTime
			}

			opts := []collector.Options{
				collector.WithTime(tc.time),
				collector.WithSysInfo(func(l *slog.Logger, opts ...sysinfo.Options) collector.SysInfo {
					return tc.sysInfo
				}),
				collector.WithMaxReports(maxReports),
			}

			l := slog.New(slog.NewTextHandler(os.Stderr, nil))
			c, err := collector.New(l, tc.consentM, tc.config, opts...)
			require.NoError(t, err, "Setup: failed to create collector")

			results, err := c.Compile()
			if tc.wantErr {
				require.Error(t, err)
				require.Empty(t, results)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, results)

			assert.Equal(t, constants.Version, results.InsightsVersion, "Compiled insights should have the expected version")
			results.InsightsVersion = "Tests"

			got, err := json.MarshalIndent(results, "", "  ")
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGolden(t, string(got))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Collect should return expected sys information")
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()

	const (
		mockTime = 10
		source   = "source"
	)

	invalidInsights := collector.Insights{SourceMetrics: make(map[string]any)}
	invalidInsights.SourceMetrics["Invalid"] = func() {}

	tests := map[string]struct {
		consentM collector.Consent

		config     collector.Config
		period     uint32
		force      bool
		dryRun     bool
		maxReports uint32
		insights   collector.Insights

		time  int64
		noDir bool

		wantErr bool
	}{
		"Writes report to disk": {
			period:     1,
			maxReports: 5,
		},
		"Does not write or cleanup if dryRun": {
			period:     1,
			dryRun:     true,
			maxReports: 1,
		},
		"Cleans up old reports if max reports exceeded": {
			period:     1,
			maxReports: 2,
		},
		"Writes report to disk and creates dir if they do not exist": {
			period:     1,
			maxReports: 5,
			noDir:      true,
		},
		"Force dryRun does not create dir if it does not exist": {
			period:     1,
			dryRun:     true,
			force:      true,
			maxReports: 5,
			noDir:      true,
		},
		"No consent writes opt-out": {
			period:     1,
			consentM:   cFalse,
			maxReports: 5,
		},
		"Writes reports to disk if duplicate reports and force": {
			period:     1,
			consentM:   cTrue,
			force:      true,
			maxReports: 5,
			time:       5,
		},
		"Force deletes duplicate reports": {
			period:     10,
			force:      true,
			maxReports: 5,
			time:       25,
		},

		// Error cases
		"Errors if consent errors": {
			period:     1,
			consentM:   cErr,
			maxReports: 5,
			wantErr:    true,
		},
		"Errors if consent true but errors": {
			period:     1,
			consentM:   cErrTrue,
			maxReports: 5,
			wantErr:    true,
		},
		"Errors if Insights cannot be marshaled": {
			config: collector.Config{
				CachePath: "",
			},
			period:     1,
			maxReports: 5,
			insights:   invalidInsights,
			wantErr:    true,
		},
		"Errors if there are duplicate reports": {
			period:     1,
			consentM:   cTrue,
			maxReports: 5,
			time:       5,
			wantErr:    true,
		},
		"Errors if period is zero and there are duplicates": {
			period:     0,
			consentM:   cTrue,
			maxReports: 5,
			time:       5,
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.config.Source == "" {
				tc.config.Source = source
			}

			dir := t.TempDir()
			if tc.consentM == nil {
				tc.consentM = cTrue
			}
			if tc.insights.SourceMetrics == nil {
				tc.insights.SourceMetrics = make(map[string]any)
			}
			tc.insights.SourceMetrics["Test Name"] = name

			sDir := filepath.Join(dir, source)
			require.NoError(t, testutils.CopyDir(t, filepath.Join("testdata", "reports_cache"), sDir), "Setup: failed to copy reports cache")
			if tc.noDir {
				require.NoError(t, os.RemoveAll(sDir), "Setup: failed to remove reports cache")
			}
			tc.config.CachePath = dir

			if tc.time == 0 {
				tc.time = mockTime
			}

			opts := []collector.Options{
				collector.WithTime(tc.time),
				collector.WithMaxReports(tc.maxReports),
			}

			l := slog.New(slog.NewTextHandler(os.Stderr, nil))
			c, err := collector.New(l, tc.consentM, tc.config, opts...)
			require.NoError(t, err, "Setup: failed to create collector")

			err = c.Write(tc.insights, tc.period, tc.force, tc.dryRun)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got, err := testutils.GetDirContents(t, sDir, 5)
			if tc.noDir && tc.dryRun {
				require.ErrorIs(t, err, os.ErrNotExist, "GetDirContents should have failed with dir not existing")
				return
			}
			require.NoError(t, err)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got)
		})
	}
}
