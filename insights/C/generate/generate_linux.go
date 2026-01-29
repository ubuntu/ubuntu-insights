//go:build tools

package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

const libname = "libinsights.so.0"

var buildTargets = []string{"libinsights.go", "log_handler.go", "internal.go"}

func main() {
	if err := buildSharedLibs(); err != nil {
		slog.Error("Failed to build shared libraries", "error", err)
		os.Exit(1)
	}

	if err := copyTypesHeader(); err != nil {
		slog.Error("Failed to copy types header", "error", err)
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

	if output, err := exec.Command("go", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("build command failed with output: %q and error: %v", output, err)
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
		return fmt.Errorf("copy command failed with output: %q and error: %v", output, err)
	}
	return nil
}
