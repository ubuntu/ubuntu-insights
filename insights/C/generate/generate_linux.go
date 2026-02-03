//go:build tools

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

const libname = "libinsights.so.0"

var buildTargets = []string{"libinsights.go", "log_handler.go", "internal.go"}

func main() {
	if err := buildSharedLibs(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build shared libraries: %v\n", err)
		os.Exit(1)
	}

	if err := copyTypesHeader(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy types header: %v\n", err)
		os.Exit(1)
	}
}

func buildSharedLibs() error {
	if version := os.Getenv("DEB_VERSION_UPSTREAM"); version != "" {
		constants.Version = version
	}
	ldflags := fmt.Sprintf("-X=github.com/ubuntu/ubuntu-insights/insights/internal/constants.Version=%s -extldflags -Wl,-soname,%s", constants.Version, libname)

	args := []string{"build", //nolint:gosec // This is controlled by the build process and also filtered here
		"-buildmode=c-shared",
		"-trimpath",
		"-ldflags", ldflags,
		"-o", fmt.Sprintf("../generated/%s", libname),
	}
	args = append(args, buildTargets...)
	cmd := exec.Command("go", args...)

	// Apply pedantic flags to CGO_CFLAGS when not doing production build.
	cgoCFlags := os.Getenv("CGO_CFLAGS")
	if cgoCFlags == "" {
		cmd.Env = append(os.Environ(), "CGO_CFLAGS=-Wextra -Werror")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build command failed with output: %s and error: %v", output, err)
	}

	// Rename the header file to insights.h
	lastDot := strings.LastIndex(libname, ".")
	expectedHeader := libname[:lastDot] + ".h"
	if err := os.Rename(fmt.Sprintf("../generated/%s", expectedHeader), "../generated/insights.h"); err != nil {
		return err
	}
	return nil
}

func copyTypesHeader() error {
	if output, err := exec.Command("cp", "./types.h", "../generated/types.h").CombinedOutput(); err != nil {
		return fmt.Errorf("copy command failed with output: %s and error: %v", output, err)
	}
	return nil
}
