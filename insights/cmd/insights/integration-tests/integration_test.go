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
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	constantstestutils "github.com/ubuntu/ubuntu-insights/insights/internal/constants/testutils"
)

var cliPath string

type fixturePaths struct {
	consent       string
	reports       string
	sourceMetrics string
	base          string
}

func TestMain(m *testing.M) {
	constantstestutils.Normalize()

	execPath, cleanup, err := buildCLI("-tags=integrationtests")
	if err != nil {
		log.Printf("Setup: failed to build CLI: %v", err)
		os.Exit(1)
	}
	defer cleanup()
	cliPath = execPath

	m.Run()
}

// moduleRoot returns the path to the module's root directory.
func moduleRoot() string {
	// p is the path to the caller file, in this case {MODULE_ROOT}/cmd/insights/integration-tests/integration_test.go
	_, p, _, _ := runtime.Caller(0)
	// Ignores the last 4 elements -> /cmd/insights/integration-tests/integration_test.go
	for range 4 {
		p = filepath.Dir(p)
	}
	return p
}

// buildCLI builds the CLI executable app and returns the path to the binary.
func buildCLI(extraArgs ...string) (execPath string, cleanup func(), err error) {
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
	cmd.Dir = moduleRoot()
	if testutils.CoverDirForTests() != "" {
		// -cover is a "positional flag", so it needs to come right after the "build" command.
		cmd.Args = append(cmd.Args, "-cover")
	}
	if testutils.IsRace() {
		cmd.Args = append(cmd.Args, "-race")
	}
	cmd.Args = append(cmd.Args, extraArgs...)
	cmd.Args = append(cmd.Args, "-o", execPath, "./cmd/insights")

	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to build cli app(%v): %s", err, out)
	}

	return execPath, cleanup, err
}

type consentFixture string

const (
	fixtureNone           consentFixture = ""
	fixtureBadExt         consentFixture = "Bad-Ext-consent.txt"
	fixtureBadFile        consentFixture = "Bad-File-consent.toml"
	fixtureBadKey         consentFixture = "Bad-Key-consent.toml"
	fixtureBadValue       consentFixture = "Bad-Value-consent.toml"
	fixtureEmpty          consentFixture = "Empty-consent.toml"
	fixtureExtraEntries   consentFixture = "Extra-Entries-consent.toml"
	fixtureFalse          consentFixture = "False-consent.toml"
	fixtureImproperName   consentFixture = "Improper-name.toml"
	fixtureLongSourceTrue consentFixture = "Long-Source-True-consent.toml"
	fixtureTrue           consentFixture = "True-consent.toml"
)

// setupFixtures copies the fixture files to the temporary directory and sets up the platform consent file to match the requested fixture. It returns the paths to the copied fixtures.
func setupFixtures(t *testing.T, initialPlatformConsent consentFixture) fixturePaths {
	t.Helper()
	baseFixturesPath := filepath.Join("testdata", "fixtures")

	dir := t.TempDir()
	paths := fixturePaths{
		consent:       filepath.Join(dir, "consents"),
		reports:       filepath.Join(dir, "reports"),
		sourceMetrics: filepath.Join(dir, "source-metrics"),
		base:          dir,
	}

	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "consents"), paths.consent), "Setup: failed to copy consents fixture")
	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "reports"), paths.reports), "Setup: failed to copy reports fixture")
	require.NoError(t, testutils.CopyDir(t, filepath.Join(baseFixturesPath, "source-metrics"), paths.sourceMetrics), "Setup: failed to copy source-metrics fixture")

	if initialPlatformConsent != "" {
		file := constants.PlatformConsentFile[:len(constants.PlatformConsentFile)-len(filepath.Ext(constants.PlatformConsentFile))]
		file += filepath.Ext(string(initialPlatformConsent))
		err := testutils.CopyFile(t, filepath.Join(paths.consent, string(initialPlatformConsent)), filepath.Join(paths.consent, file))
		require.NoError(t, err, "Setup: failed to setup platform consent file")
	}

	return paths
}
