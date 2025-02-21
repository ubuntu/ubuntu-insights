package collector_test

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
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

type MockTimeProvider struct {
	CurrentTime int64
}

func (m MockTimeProvider) Now() time.Time {
	return time.Unix(m.CurrentTime, 0)
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		consentM       collector.ConsentManager
		source         string
		period         uint
		dryRun         bool
		nilConsent     bool
		emptyCachePath bool

		wantErr bool
	}{
		"Blank collector": {
			source: "source",
		},
		"Dry run": {
			source: "source",
			dryRun: true,
		},

		"Overflow Period": {
			source:  "source",
			period:  math.MaxInt + 1,
			wantErr: true,
		},
		"Empty source": {
			source:  "",
			wantErr: true,
		},
		"Nil Consent": {
			source:     "source",
			nilConsent: true,

			wantErr: true,
		},
		"Empty Cache Path": {
			source:         "source",
			emptyCachePath: true,
			wantErr:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentM == nil && !tc.nilConsent {
				tc.consentM = cTrue
			}

			dir := t.TempDir()
			if tc.emptyCachePath {
				dir = ""
			}

			result, err := collector.New(tc.consentM, dir, tc.source, tc.period, tc.dryRun)
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

	tests := map[string]struct {
		consentM          collector.ConsentManager
		source            string
		sourceMetricsFile string
		period            uint
		dryRun            bool
		force             bool

		sysInfo collector.SysInfo
		noDir   bool

		wantErr bool
	}{
		"Basic": {
			period:   1,
			consentM: cTrue,
			source:   "source",
		},
		"Dry Run": {
			period:   1,
			consentM: cTrue,
			source:   "source",
			dryRun:   true,
		},
		"With SourceMetrics": {
			period:            1,
			consentM:          cTrue,
			source:            "source",
			sourceMetricsFile: "normal.json",
		},

		"Consent False": {
			period:   1,
			consentM: cFalse,
			source:   "source",
		},
		"Consent Error": {
			period:   1,
			consentM: cErr,
			source:   "source",
			wantErr:  true,
		},
		"Consent Error True": {
			period:   1,
			consentM: cErrTrue,
			source:   "source",
			wantErr:  true,
		},
		"Period 0": {
			period:   0,
			consentM: cTrue,
			source:   "source",
			wantErr:  true,
		},
		"Non-existent source metrics file": {
			period:            1,
			consentM:          cTrue,
			source:            "source",
			sourceMetricsFile: "nonexistent.json",
			wantErr:           true,
		},
		"Invalid source metrics file": {
			period:            1,
			consentM:          cTrue,
			source:            "source",
			sourceMetricsFile: "invalid.json",
			wantErr:           true,
		},
		"Bad ext source metrics file": {
			period:            1,
			consentM:          cTrue,
			source:            "source",
			sourceMetricsFile: "bad_ext.json",
			wantErr:           true,
		},
		"Empty source metrics file": {
			period:            1,
			consentM:          cTrue,
			source:            "source",
			sourceMetricsFile: "empty.json",
			wantErr:           true,
		},
		"Duplicate report": {
			period:   20,
			consentM: cTrue,
			source:   "source",
			wantErr:  true,
		},
		"SysInfo Collect Error": {
			period:   1,
			consentM: cTrue,
			source:   "source",
			sysInfo:  testSysInfo{info: sysinfo.Info{}, err: fmt.Errorf("sysinfo error")},
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			sdir := filepath.Join(dir, tc.source)
			require.NoError(t, testutils.CopyDir(t, filepath.Join("testdata", "reports_cache"), sdir), "Setup: failed to copy reports cache")

			if tc.sysInfo == nil {
				tc.sysInfo = testSysInfo{info: sysinfo.Info{}, err: nil}
			}

			opts := []collector.Options{
				collector.WithTimeProvider(MockTimeProvider{CurrentTime: mockTime}),
				collector.WithSysInfo(tc.sysInfo),
				collector.WithMaxReports(maxReports),
			}

			if tc.sourceMetricsFile != "" {
				opts = append(opts, collector.WithSourceMetricsPath(filepath.Join("testdata", "source_metrics", tc.sourceMetricsFile)))
			}

			c, err := collector.New(tc.consentM, dir, tc.source, tc.period, tc.dryRun, opts...)
			require.NoError(t, err, "Setup: failed to create collector")

			got, err := c.Compile(tc.force)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, got)

			if insights, ok := got.(collector.Insights); ok {
				assert.NotEmpty(t, insights)
				if insights.SourceMetrics == nil {
					insights.SourceMetrics = make(map[string]interface{})
				}

				g := struct {
					InsightsVersion string
					SourceMetrics   map[string]interface{}
				}{InsightsVersion: insights.InsightsVersion, SourceMetrics: insights.SourceMetrics}
				want := testutils.LoadWithUpdateFromGoldenYAML(t, g)
				require.EqualValues(t, want, g)
				return
			}

			require.Equal(t, constants.OptOutJSON, got)
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
}
