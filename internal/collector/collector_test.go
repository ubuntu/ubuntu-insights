package collector_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector"
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

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		consentM collector.ConsentManager
		source   string
		period   uint
		dryRun   bool

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
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.consentM == nil {
				tc.consentM = cTrue
			}

			dir := t.TempDir()

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
