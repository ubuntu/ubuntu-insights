package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

// BuildCLI builds the CLI executable app and returns the path to the binary.
func BuildCLI(extraArgs ...string) (execPath string, cleanup func(), err error) {
	projectRoot := ProjectRoot()

	tempDir, err := os.MkdirTemp("", "ubuntu-insights-tests-cli")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	cleanup = func() { os.RemoveAll(tempDir) }

	execPath = filepath.Join(tempDir, constants.CmdName)
	cmd := exec.Command("go", "build")
	cmd.Dir = projectRoot
	if CoverDirForTests() != "" {
		// -cover is a "positional flag", so it needs to come right after the "build" command.
		cmd.Args = append(cmd.Args, "-cover")
	}
	if IsRace() {
		cmd.Args = append(cmd.Args, "-race")
	}
	cmd.Args = append(cmd.Args, "-gcflags=all=-N -l")
	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "-o", execPath, "./cmd/authd")

	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to build cli app(%v): %s", err, out)
	}

	return execPath, cleanup, err
}
