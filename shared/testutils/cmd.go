// Package testutils provides helper functions for testing
package testutils

import (
	"fmt"
	"os"
)

const helperStr = "GO_HELPER_PROCESS"

// SetupFakeCmdArgs sets up arguments to run a fake testing command.
func SetupFakeCmdArgs(fakeCmdFunc string, args ...string) []string {
	cmdArgs := []string{os.Args[0], fmt.Sprintf("-test.run=%s", fakeCmdFunc), "--", helperStr}
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

	if len(args) == 0 || args[0] != helperStr {
		return nil, fmt.Errorf("fake cmd called in non-testing environment")
	}

	return args[1:], nil
}

// SetupHelperCoverdir creates a directory and sets GOCOVERDIR to it but only if in a helper and GOCOVERDIR is set.
// It is the callers job to remove the directory.
// This function will exit with 1 if it cannot create the directory.
func SetupHelperCoverdir() (string, bool) {
	base, ok := os.LookupEnv("GOCOVERDIR")
	if !ok {
		return base, ok
	}

	helper := false
	for _, a := range os.Args {
		if a == helperStr {
			helper = true
			break
		}
	}

	if !helper {
		return "", false
	}

	d, err := os.MkdirTemp(base, "*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temporary coverage directory: %v", err)
		os.Exit(1)
	}
	os.Setenv("GOCOVERDIR", d)
	return d, true
}
