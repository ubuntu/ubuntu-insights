package commands

import (
	"testing"

	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestCollectFlags(t *testing.T) {
	testCases := []testutils.CmdTestCase{
		{
			Name:     "source",
			Short:    "s",
			Required: true,
			BaseCmd:  collectCmd,
		},
		{
			Name:    "dir",
			Dirname: true,
			BaseCmd: collectCmd,
		},
		{
			Name:    "period",
			Short:   "p",
			BaseCmd: collectCmd,
		},
		{
			Name:    "force",
			Short:   "f",
			BaseCmd: collectCmd,
		},
		{
			Name:    "dry-run",
			Short:   "d",
			BaseCmd: collectCmd,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			testutils.FlagTestHelper(t, tc)
		})
	}
}
