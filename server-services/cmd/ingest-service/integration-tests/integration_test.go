package ingest_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ubuntu/ubuntu-insights/server-services/internal/shared/constants"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

var cliPath string

func TestMain(m *testing.M) {
	execPath, cleanup, err := buildCLI("-tags=integrationtests")
	if err != nil {
		log.Printf("Setup: failed to build CLI: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	cliPath = execPath

	m.Run()
}

// buildCLI builds the CLI executable app and returns the path to the binary.
func buildCLI(extraArgs ...string) (execPath string, cleanup func(), err error) {
	projectRoot := testutils.ProjectRoot()

	tempDir, err := os.MkdirTemp("", "ubuntu-insights-ingest-tests-cli")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	cleanup = func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintf(os.Stderr, "Teardown: warning: could not clean up temporary directory: %v", err)
		}
	}

	execPath = filepath.Join(tempDir, constants.IngestServiceCmdName)
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}
	cmd := exec.Command("go", "build")
	cmd.Dir = projectRoot
	if testutils.CoverDirForTests() != "" {
		// -cover is a "positional flag", so it needs to come right after the "build" command.
		cmd.Args = append(cmd.Args, "-cover")
	}
	if testutils.IsRace() {
		cmd.Args = append(cmd.Args, "-race")
	}
	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "-o", execPath, "./server-services/cmd/ingest-service")

	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to build cli app(%v): %s", err, out)
	}

	return execPath, cleanup, err
}
