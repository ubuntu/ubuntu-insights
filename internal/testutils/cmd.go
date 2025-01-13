package testutils

import (
	"os/exec"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

type CmdTestCase struct {
	Name           string
	Short          string
	Required       bool
	Dirname        bool
	PersistentFlag bool
	BaseCmd        *cobra.Command
}

func FlagTestHelper(t *testing.T, testCase CmdTestCase) {
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

func MakeTestCmd(want string) *exec.Cmd {
	return exec.Command("echo", "-n", "-e", want)
}
