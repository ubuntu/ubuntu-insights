package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCollectFlags(t *testing.T) {
	testCases := []struct {
		name     string
		short    string
		required bool
		dirname  bool
	}{{
		name:     "source",
		short:    "s",
		required: true,
	}, {
		name:    "dir",
		dirname: true,
	}, {
		name:  "period",
		short: "p",
	}, {
		name:  "force",
		short: "f",
	}, {
		name:  "dry-run",
		short: "d",
	},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := collectCmd.Flags().Lookup(tc.name)
			assert.NotNil(t, flag)
			assert.Equal(t, tc.short, flag.Shorthand)

			if tc.required {
				assert.Equal(t, "true", flag.Annotations[cobra.BashCompOneRequiredFlag][0])
			} else {
				assert.Nil(t, flag.Annotations[cobra.BashCompOneRequiredFlag])
			}

			if tc.dirname {
				assert.Equal(t, []string{}, flag.Annotations[cobra.BashCompSubdirsInDir])
			} else {
				assert.Nil(t, flag.Annotations[cobra.BashCompSubdirsInDir])
			}
		})
	}
}
