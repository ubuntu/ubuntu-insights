package consent_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestGetState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source     string
		globalFile string

		wantErr bool
	}{
		"No Global File": {wantErr: true},

		// Global File Tests
		"Valid True Global File":    {globalFile: "valid_true-consent.toml"},
		"Valid False Global File":   {globalFile: "valid_false-consent.toml"},
		"Invalid Value Global File": {globalFile: "invalid_value-consent.toml", wantErr: true},
		"Invalid File Global File":  {globalFile: "invalid_file-consent.toml", wantErr: true},

		// Source Specific Tests
		"Valid True Global File, Valid True Source":    {globalFile: "valid_true-consent.toml", source: "valid_true"},
		"Valid True Global File, Valid False Source":   {globalFile: "valid_true-consent.toml", source: "valid_false"},
		"Valid True Global File, Invalid Value Source": {globalFile: "valid_true-consent.toml", source: "invalid_value", wantErr: true},
		"Valid True Global File, Invalid File Source":  {globalFile: "valid_true-consent.toml", source: "invalid_file", wantErr: true},
		"Valid True Global File, No File Source":       {globalFile: "valid_true-consent.toml", source: "not_a_file", wantErr: true},

		// Invalid Global File, Source Specific Tests
		"Invalid Value Global File, Valid True Source":    {globalFile: "invalid_value-consent.toml", source: "valid_true"},
		"Invalid Value Global File, Valid False Source":   {globalFile: "invalid_value-consent.toml", source: "valid_false"},
		"Invalid Value Global File, Invalid Value Source": {globalFile: "invalid_value-consent.toml", source: "invalid_value", wantErr: true},
		"Invalid Value Global File, Invalid File Source":  {globalFile: "invalid_value-consent.toml", source: "invalid_file", wantErr: true},
		"Invalid Value Global File, No File Source":       {globalFile: "invalid_value-consent.toml", source: "not_a_file", wantErr: true},

		// No Global File, Source Specific Tests
		"No Global File, Valid True Source":    {source: "valid_true"},
		"No Global File, Valid False Source":   {source: "valid_false"},
		"No Global File, Invalid Value Source": {source: "invalid_value", wantErr: true},
		"No Global File, Invalid File Source":  {source: "invalid_file", wantErr: true},
		"No Global File, No File Source":       {source: "not_a_file", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.globalFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(dir)

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
		States    map[string]bool
		FileCount int
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.globalFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(dir)

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

func TestHasConsent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string

		globalFile string

		want    bool
		wantErr bool
	}{
		"True Global-True Source":          {source: "valid_true", globalFile: "valid_true-consent.toml", want: true},
		"True Global-False Source":         {source: "valid_false", globalFile: "valid_true-consent.toml", want: false},
		"True Global-Invalid Value Source": {source: "invalid_value", globalFile: "valid_true-consent.toml", want: true},
		"True Global-Invalid File Source":  {source: "invalid_file", globalFile: "valid_true-consent.toml", want: true},
		"True Global-Not A File Source":    {source: "not_a_file", globalFile: "valid_true-consent.toml", want: true},

		"False Global-True Source":          {source: "valid_true", globalFile: "valid_false-consent.toml", want: true},
		"False Global-False Source":         {source: "valid_false", globalFile: "valid_false-consent.toml", want: false},
		"False Global-Invalid Value Source": {source: "invalid_value", globalFile: "valid_false-consent.toml", want: false},
		"False Global-Invalid File Source":  {source: "invalid_file", globalFile: "valid_false-consent.toml", want: false},
		"False Global-Not A File Source":    {source: "not_a_file", globalFile: "valid_false-consent.toml", want: false},

		"No Global-True Source":          {source: "valid_true", want: true},
		"No Global-False Source":         {source: "valid_false", want: false},
		"No Global-Invalid Value Source": {source: "invalid_value", wantErr: true},
		"No Global-Invalid File Source":  {source: "invalid_file", wantErr: true},
		"No Global-Not A File Source":    {source: "not_a_file", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.globalFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(dir)

			got, err := cm.HasConsent(tc.source)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			require.Equal(t, tc.want, got, "HasConsent should return expected consent state")
		})
	}
}

func setupTmpConsentFiles(t *testing.T, globalFile string) (string, error) {
	t.Helper()

	// Setup temporary directory
	var err error
	dir := t.TempDir()

	if err = testutils.CopyDir(t, filepath.Join("testdata", "consent_files"), dir); err != nil {
		return dir, fmt.Errorf("failed to copy testdata directory to temporary directory: %v", err)
	}

	// Setup globalFile if provided
	if globalFile != "" {
		if err = testutils.CopyFile(t, filepath.Join(dir, globalFile), filepath.Join(dir, "consent.toml")); err != nil {
			return dir, fmt.Errorf("failed to copy requested global consent file: %v", err)
		}
	}

	return dir, nil
}
