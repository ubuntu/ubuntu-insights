package insights_test

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

var cliPath string

func TestMain(m *testing.M) {
	// TODO: Which pattern is better? TestMain with just one cli exec built, or each test building its own cli exec?
	// TODO: Should integration tests be independent tests per command, or should common usecases be tested in sequence? (eg., set consent, collect, upload all in one go?)
	// TODO: To pass a local server address to the CLI, should this be done through a generated viper config file or env variable? Additionally, should it be restricted somehow to only work with an integration test build? Or, should the server instead be setup using build tags, with the behavior being controlled by additional build tags/args?
	execPath, cleanup, err := testutils.BuildCLI("-tags=integrationtests")
	if err != nil {
		log.Printf("Setup: failed to build CLI: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	cliPath = execPath

	m.Run()
}

// buildCLI builds the CLI executable app and returns the path to the binary.
func buildCLI(t *testing.T, extraArgs ...string) (execPath string) {
	projectRoot := testutils.ProjectRoot()

	execPath = filepath.Join(t.TempDir(), constants.CmdName)
	cmd := exec.Command("go", "build")
	cmd.Dir = projectRoot
	if testutils.CoverDirForTests() != "" {
		// -cover is a "positional flag", so it needs to come right after the "build" command.
		cmd.Args = append(cmd.Args, "-cover")
	}
	if testutils.IsRace() {
		cmd.Args = append(cmd.Args, "-race")
	}
	cmd.Args = append(cmd.Args, "-gcflags=all=-N -l")
	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "-o", execPath, "./cmd/authd")

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "Setup: failed to build cli app: %s", out)

	return execPath
}
