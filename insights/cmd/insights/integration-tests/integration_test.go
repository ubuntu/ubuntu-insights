package insights_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

var cliPath string

type fixturePaths struct {
	consent       string
	reports       string
	sourceMetrics string
	base          string
}

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

	tempDir, err := os.MkdirTemp("", "ubuntu-insights-tests-cli")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	cleanup = func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintf(os.Stderr, "Teardown: warning: could not clean up temporary directory: %v", err)
		}
	}

	execPath = filepath.Join(tempDir, constants.CmdName)
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
	cmd.Args = append(cmd.Args, "-o", execPath, "./insights/cmd/insights")

	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to build cli app(%v): %s", err, out)
	}

	return execPath, cleanup, err
}

// copyFixtures copies the fixture files to the temporary directory.
func copyFixtures(t *testing.T, consentFixture string) fixturePaths {
	t.Helper()
	baseFixturesPath := filepath.Join("testdata", "fixtures")

	dir := t.TempDir()
	paths := fixturePaths{
		consent:       filepath.Join(dir, "consents"),
		reports:       filepath.Join(dir, "reports"),
		sourceMetrics: filepath.Join(dir, "source-metrics"),
		base:          dir,
	}

	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "consents", consentFixture), paths.consent), "Setup: failed to copy consents fixture")
	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "reports"), paths.reports), "Setup: failed to copy reports fixture")
	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "source-metrics"), paths.sourceMetrics), "Setup: failed to copy source-metrics fixture")

	return paths
}
