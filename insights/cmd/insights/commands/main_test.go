package commands_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	constantstestutils "github.com/ubuntu/ubuntu-insights/insights/internal/constants/testutils"
)

func TestMain(m *testing.M) {
	constantstestutils.Normalize()

	m.Run()
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

func newAppForTests(t *testing.T, args []string, initialPlatformConsent consentFixture, opts ...commands.Options) (app *commands.App, cachePath string) {
	t.Helper()

	cachePath = t.TempDir()
	require.NoError(t, testutils.CopyDir(t, filepath.Join("testdata", "reports"), cachePath), "Setup: could not copy cache dir")
	args = append(args, "--insights-dir", cachePath)

	consentPath := t.TempDir()
	require.NoError(t, testutils.CopyDir(t, filepath.Join("testdata", "consents"), consentPath), "Setup: could not copy consent dir")
	if initialPlatformConsent != "" {
		file := constants.PlatformConsentFile[:len(constants.PlatformConsentFile)-len(filepath.Ext(constants.PlatformConsentFile))]
		file += filepath.Ext(string(initialPlatformConsent))
		err := testutils.CopyFile(t, filepath.Join(consentPath, string(initialPlatformConsent)), filepath.Join(consentPath, file))
		require.NoError(t, err, "Setup: failed to setup platform consent file")
	}

	args = append(args, "--consent-dir", consentPath)

	app, err := commands.New(opts...)
	require.NoError(t, err, "Setup: could not create app")

	app.SetArgs(args)
	return app, cachePath
}
