// Package testutils provides helper functions for testing
package testutils

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// CmdTestCase is a test case for testing cobra CMD flags.
type CmdTestCase struct {
	Name           string
	Short          string
	Required       bool
	Dirname        bool
	PersistentFlag bool
	BaseCmd        *cobra.Command
}

// FlagTestHelper is a helper function to test cobra CMD flags.
func FlagTestHelper(t *testing.T, testCase CmdTestCase) {
	t.Helper()
	var flag *pflag.Flag

	if testCase.PersistentFlag {
		flag = testCase.BaseCmd.PersistentFlags().Lookup(testCase.Name)
	} else {
		flag = testCase.BaseCmd.Flags().Lookup(testCase.Name)
	}
	assert.NotNil(t, flag)
	assert.Equal(t, testCase.Short, flag.Shorthand)

	if testCase.Required {
		assert.Equal(t, "true", flag.Annotations[cobra.BashCompOneRequiredFlag][0])
	} else {
		assert.Nil(t, flag.Annotations[cobra.BashCompOneRequiredFlag])
	}

	if testCase.Dirname {
		assert.Equal(t, []string{}, flag.Annotations[cobra.BashCompSubdirsInDir])
	} else {
		assert.Nil(t, flag.Annotations[cobra.BashCompSubdirsInDir])
	}
}

// SetupFakeCmdArgs sets up arguments to run a fake testing command.
func SetupFakeCmdArgs(fakeCmdFunc string, args ...string) []string {
	cmdArgs := []string{os.Args[0], fmt.Sprintf("-test.run=%s", fakeCmdFunc), "--", "GO_HELPER_PROCESS"}
	return append(cmdArgs, args...)
}

// GetFakeCmdArgs gets the arguments passed into a fake testing command, or errors without the proper environment.
func GetFakeCmdArgs() (args []string, err error) {
	args = os.Args
	for len(args) > 0 {
		if args[0] != "--" {
			args = args[1:]
			continue
		}
		args = args[1:]
		break
	}

	if len(args) == 0 || args[0] != "GO_HELPER_PROCESS" {
		return nil, fmt.Errorf("fake cmd called in non-testing environment")
	}

	return args[1:], nil
}
