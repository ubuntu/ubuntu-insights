package consent_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	constantstestutils "github.com/ubuntu/ubuntu-insights/insights/internal/constants/testutils"
)

func TestMain(m *testing.M) {
	constantstestutils.Normalize()

	m.Run()
}

func TestGetState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source                     string
		initialPlatformConsentFile string

		wantErr bool
	}{
		// Default (Platform) Consent Tests
		"No platform consent file errors":         {wantErr: true},
		"Valid true platform file returns true":   {initialPlatformConsentFile: "valid_true-consent.toml"},
		"Valid false platform file returns false": {initialPlatformConsentFile: "valid_false-consent.toml"},
		"Invalid value platform file errors":      {initialPlatformConsentFile: "invalid_value-consent.toml", wantErr: true},
		"Invalid file platform file errors":       {initialPlatformConsentFile: "invalid_file-consent.toml", wantErr: true},

		// Source Specific Tests
		"Valid true source returns true":   {source: "valid_true"},
		"Valid false source returns false": {source: "valid_false"},
		"Invalid value source errors":      {source: "invalid_value", wantErr: true},
		"Invalid file source errors":       {source: "invalid_file", wantErr: true},
		"No file source errors":            {source: "not_a_file", wantErr: true},
		"Ignores irrelevant source files":  {initialPlatformConsentFile: "valid_false-consent.toml", source: "valid_true"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.initialPlatformConsentFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(slog.Default(), dir)

			got, err := cm.GetState(tc.source)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentState should return expected consent state")
		})
	}
}

func TestSetState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		consentStates              map[string]bool
		initialPlatformConsentFile string

		writeSource string
		writeState  bool

		wantErr bool
	}{
		// New Platform Consent File Tests
		"New File, Write Platform False": {writeState: false},
		"New File, Write Platform True":  {writeState: true},
		"New File, Write Source True":    {writeSource: "new_true", writeState: true},
		"New File, Write Source False":   {writeSource: "new_false"},

		// Overwrite Platform Consent File, Different State
		"Overwrite File, Write Diff Platform False": {initialPlatformConsentFile: "valid_true-consent.toml", writeState: false},
		"Overwrite File, Write Diff Platform True":  {initialPlatformConsentFile: "valid_false-consent.toml", writeState: true},
		"Overwrite File, Write Diff Source True":    {initialPlatformConsentFile: "valid_true-consent.toml", writeSource: "valid_false", writeState: true},
		"Overwrite File, Write Diff Source False":   {initialPlatformConsentFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: false},

		// Overwrite Platform Consent File, Same State
		"Overwrite File, Write Platform True":  {initialPlatformConsentFile: "valid_true-consent.toml", writeState: true},
		"Overwrite File, Write Platform False": {initialPlatformConsentFile: "valid_false-consent.toml", writeState: false},
		"Overwrite File, Write Source True":    {initialPlatformConsentFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: true},
		"Overwrite File, Write Source False":   {initialPlatformConsentFile: "valid_false-consent.toml", writeSource: "valid_false", writeState: false},
	}

	type goldenFile struct {
		States    map[string]bool
		FileCount int
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.initialPlatformConsentFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(slog.Default(), dir)

			err = cm.SetState(tc.writeSource, tc.writeState)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			states, err := cm.GetAllSourceConsentStates(true)
			require.NoError(t, err, "got an unexpected error while getting consent states")

			d, err := os.ReadDir(dir)
			require.NoError(t, err, "failed to read temporary directory")
			got := goldenFile{States: states, FileCount: len(d)}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentStates should return expected consent states")
		})
	}
}

// setupTmpConsentFiles sets up a temporary directory with consent files copied from testdata/consent_files. If a platformConsentFile is provided, it copies that file to be the platform consent file. It returns the path to the temporary directory.
func setupTmpConsentFiles(t *testing.T, initialPlatformConsent string) (string, error) {
	t.Helper()

	// Setup temporary directory
	var err error
	dir := t.TempDir()

	if err = testutils.CopyDir(t, filepath.Join("testdata", "consent_files"), dir); err != nil {
		return dir, fmt.Errorf("failed to copy testdata directory to temporary directory: %v", err)
	}

	// Setup platformConsentFile if provided
	if initialPlatformConsent != "" {
		if err = testutils.CopyFile(t, filepath.Join(dir, initialPlatformConsent), filepath.Join(dir, constants.PlatformConsentFile)); err != nil {
			return dir, fmt.Errorf("failed to copy requested platform consent file: %v", err)
		}
	}

	return dir, nil
}
