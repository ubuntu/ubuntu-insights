package commands

import (
	"testing"

	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestConsentFlags(t *testing.T) {
	testCases := []testutils.CmdTestCase{
		{
			Name:    "source",
			Short:   "s",
			BaseCmd: consentCmd,
		}, {
			Name:    "consent-state",
			Short:   "c",
			BaseCmd: consentCmd,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutils.FlagTestHelper(t, tc)
		})
	}
}
