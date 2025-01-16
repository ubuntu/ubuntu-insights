package consent_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

// consentDir is a struct that holds a test's temporary directory and its locks.
// It should be cleaned up after the test is done.
type consentDir struct {
	dir string
}

func TestGetConsentStates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources    []string
		globalFile string

		wantErr bool
	}{
		"No Global File": {},

		// Global File Tests
		"Valid True Global File":    {globalFile: "valid_true-consent.toml"},
		"Valid False Global File":   {globalFile: "valid_false-consent.toml"},
		"Invalid Value Global File": {globalFile: "invalid_value-consent.toml"},
		"Invalid File Global File":  {globalFile: "invalid_file-consent.toml"},

		// Source Specific Tests
		"Valid True Global File, Valid True Source":                       {globalFile: "valid_true-consent.toml", sources: []string{"valid_true"}},
		"Valid True Global File, Valid False Source":                      {globalFile: "valid_true-consent.toml", sources: []string{"valid_false"}},
		"Valid True Global File, Invalid Value Source":                    {globalFile: "valid_true-consent.toml", sources: []string{"invalid_value"}},
		"Valid True Global File, Invalid File Source":                     {globalFile: "valid_true-consent.toml", sources: []string{"invalid_file"}},
		"Valid True Global File, No File Source":                          {globalFile: "valid_true-consent.toml", sources: []string{"not_a_file"}},
		"Valid True Global File, 2 Multiple Sources (VTrue, VFalse)":      {globalFile: "valid_true-consent.toml", sources: []string{"valid_true", "valid_false"}},
		"Valid True Global File, 3 Multiple Sources (VTrue, VFalse, NAF)": {globalFile: "valid_true-consent.toml", sources: []string{"valid_true", "valid_false", "not_a_file"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cDir, err := setupTmpConsentFiles(t, tc.globalFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			defer cDir.cleanup(t)
			cm := consent.New(cDir.dir)

			got, err := cm.GetConsentStates(tc.sources)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentStates should return expected consent states")
		})
	}
}

func TestSetConsentStates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources       []string
		consentStates map[string]bool
		globalFile    string

		writeSource string
		writeState  bool

		wantErr bool
	}{
		// New File Tests
		"New File, Write Global False": {},
		"New File, Write Global True":  {writeState: true},
		"New File, Write Source True":  {writeSource: "new_true", writeState: true},
		"New File, Write Source False": {writeSource: "new_false"},

		// Overwrite File, Different State
		"Overwrite File, Write Diff Global False": {globalFile: "valid_true-consent.toml", writeState: false},
		"Overwrite File, Write Diff Global True":  {globalFile: "valid_false-consent.toml", writeState: true},
		"Overwrite File, Write Diff Source True":  {globalFile: "valid_true-consent.toml", writeSource: "valid_false", writeState: true},
		"Overwrite File, Write Diff Source False": {globalFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: false},

		// Overwrite File, Same State
		"Overwrite File, Write Global True":  {globalFile: "valid_true-consent.toml", writeState: true},
		"Overwrite File, Write Global False": {globalFile: "valid_false-consent.toml", writeState: false},
		"Overwrite File, Write Source True":  {globalFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: true},
		"Overwrite File, Write Source False": {globalFile: "valid_false-consent.toml", writeSource: "valid_false", writeState: false},
	}

	type goldenFile struct {
		States    consent.States
		FileCount uint
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cDir, err := setupTmpConsentFiles(t, tc.globalFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			defer cDir.cleanup(t)
			cm := consent.New(cDir.dir)

			err = cm.SetConsentState(tc.writeSource, tc.writeState)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			states, err := cm.GetConsentStates(tc.sources)
			require.NoError(t, err, "got an unexpected error while getting consent states")

			d, err := os.ReadDir(cDir.dir)
			require.NoError(t, err, "failed to read temporary directory")
			got := goldenFile{States: states, FileCount: uint(len(d))}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentStates should return expected consent states")
		})
	}
}

// cleanup unlocks all the locks and removes the temporary directory including its contents.
func (cDir consentDir) cleanup(t *testing.T) {
	t.Helper()
	assert.NoError(t, os.RemoveAll(cDir.dir), "Cleanup: failed to remove temporary directory")
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func copyDir(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func setupTmpConsentFiles(t *testing.T, globalFile string) (*consentDir, error) {
	t.Helper()
	cDir := consentDir{}

	// Setup temporary directory
	var err error
	cDir.dir, err = os.MkdirTemp("", "consent-files")
	if err != nil {
		return &cDir, fmt.Errorf("failed to create temporary directory: %v", err)
	}

	if err = copyDir(filepath.Join("testdata", "consent_files"), cDir.dir); err != nil {
		return &cDir, fmt.Errorf("failed to copy testdata directory to temporary directory: %v", err)
	}

	// Setup globalFile if provided
	if globalFile != "" {
		if err = copyFile(filepath.Join(cDir.dir, globalFile), filepath.Join(cDir.dir, "consent.toml")); err != nil {
			return &cDir, fmt.Errorf("failed to copy requested global consent file: %v", err)
		}
	}

	return &cDir, nil
}
