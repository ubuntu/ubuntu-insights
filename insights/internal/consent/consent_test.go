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
)

func TestGetState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source      string
		defaultFile string

		wantErr bool
	}{
		"No Default File": {wantErr: true},

		// Default File Tests
		"Valid True Default File":    {defaultFile: "valid_true-consent.toml"},
		"Valid False Default File":   {defaultFile: "valid_false-consent.toml"},
		"Invalid Value Default File": {defaultFile: "invalid_value-consent.toml", wantErr: true},
		"Invalid File Default File":  {defaultFile: "invalid_file-consent.toml", wantErr: true},

		// Source Specific Tests
		"Valid True Default File, Valid True Source":    {defaultFile: "valid_true-consent.toml", source: "valid_true"},
		"Valid True Default File, Valid False Source":   {defaultFile: "valid_true-consent.toml", source: "valid_false"},
		"Valid True Default File, Invalid Value Source": {defaultFile: "valid_true-consent.toml", source: "invalid_value", wantErr: true},
		"Valid True Default File, Invalid File Source":  {defaultFile: "valid_true-consent.toml", source: "invalid_file", wantErr: true},
		"Valid True Default File, No File Source":       {defaultFile: "valid_true-consent.toml", source: "not_a_file", wantErr: true},

		// Invalid Default File, Source Specific Tests
		"Invalid Value Default File, Valid True Source":    {defaultFile: "invalid_value-consent.toml", source: "valid_true"},
		"Invalid Value Default File, Valid False Source":   {defaultFile: "invalid_value-consent.toml", source: "valid_false"},
		"Invalid Value Default File, Invalid Value Source": {defaultFile: "invalid_value-consent.toml", source: "invalid_value", wantErr: true},
		"Invalid Value Default File, Invalid File Source":  {defaultFile: "invalid_value-consent.toml", source: "invalid_file", wantErr: true},
		"Invalid Value Default File, No File Source":       {defaultFile: "invalid_value-consent.toml", source: "not_a_file", wantErr: true},

		// No Default File, Source Specific Tests
		"No Default File, Valid True Source":    {source: "valid_true"},
		"No Default File, Valid False Source":   {source: "valid_false"},
		"No Default File, Invalid Value Source": {source: "invalid_value", wantErr: true},
		"No Default File, Invalid File Source":  {source: "invalid_file", wantErr: true},
		"No Default File, No File Source":       {source: "not_a_file", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.defaultFile)
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
		consentStates map[string]bool
		defaultFile   string

		writeSource string
		writeState  bool

		wantErr bool
	}{
		// New File Tests
		"New File, Write Default False": {},
		"New File, Write Default True":  {writeState: true},
		"New File, Write Source True":   {writeSource: "new_true", writeState: true},
		"New File, Write Source False":  {writeSource: "new_false"},

		// Overwrite File, Different State
		"Overwrite File, Write Diff Default False": {defaultFile: "valid_true-consent.toml", writeState: false},
		"Overwrite File, Write Diff Default True":  {defaultFile: "valid_false-consent.toml", writeState: true},
		"Overwrite File, Write Diff Source True":   {defaultFile: "valid_true-consent.toml", writeSource: "valid_false", writeState: true},
		"Overwrite File, Write Diff Source False":  {defaultFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: false},

		// Overwrite File, Same State
		"Overwrite File, Write Default True":  {defaultFile: "valid_true-consent.toml", writeState: true},
		"Overwrite File, Write Default False": {defaultFile: "valid_false-consent.toml", writeState: false},
		"Overwrite File, Write Source True":   {defaultFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: true},
		"Overwrite File, Write Source False":  {defaultFile: "valid_false-consent.toml", writeSource: "valid_false", writeState: false},
	}

	type goldenFile struct {
		States    map[string]bool
		FileCount int
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.defaultFile)
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

func TestHasConsent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		source string

		defaultFile string

		want    bool
		wantErr bool
	}{
		"True Default-True Source":          {source: "valid_true", defaultFile: "valid_true-consent.toml", want: true},
		"True Default-False Source":         {source: "valid_false", defaultFile: "valid_true-consent.toml", want: false},
		"True Default-Invalid Value Source": {source: "invalid_value", defaultFile: "valid_true-consent.toml", want: true},
		"True Default-Invalid File Source":  {source: "invalid_file", defaultFile: "valid_true-consent.toml", want: true},
		"True Default-Not A File Source":    {source: "not_a_file", defaultFile: "valid_true-consent.toml", want: true},

		"False Default-True Source":          {source: "valid_true", defaultFile: "valid_false-consent.toml", want: true},
		"False Default-False Source":         {source: "valid_false", defaultFile: "valid_false-consent.toml", want: false},
		"False Default-Invalid Value Source": {source: "invalid_value", defaultFile: "valid_false-consent.toml", want: false},
		"False Default-Invalid File Source":  {source: "invalid_file", defaultFile: "valid_false-consent.toml", want: false},
		"False Default-Not A File Source":    {source: "not_a_file", defaultFile: "valid_false-consent.toml", want: false},

		"No Default-True Source":          {source: "valid_true", want: true},
		"No Default-False Source":         {source: "valid_false", want: false},
		"No Default-Invalid Value Source": {source: "invalid_value", wantErr: true},
		"No Default-Invalid File Source":  {source: "invalid_file", wantErr: true},
		"No Default-Not A File Source":    {source: "not_a_file", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpConsentFiles(t, tc.defaultFile)
			require.NoError(t, err, "Setup: failed to setup temporary consent files")
			cm := consent.New(slog.Default(), dir)

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

func setupTmpConsentFiles(t *testing.T, defaultFile string) (string, error) {
	t.Helper()

	// Setup temporary directory
	var err error
	dir := t.TempDir()

	if err = testutils.CopyDir(t, filepath.Join("testdata", "consent_files"), dir); err != nil {
		return dir, fmt.Errorf("failed to copy testdata directory to temporary directory: %v", err)
	}

	// Setup defaultFile if provided
	if defaultFile != "" {
		if err = testutils.CopyFile(t, filepath.Join(dir, defaultFile), filepath.Join(dir, "consent.toml")); err != nil {
			return dir, fmt.Errorf("failed to copy requested default consent file: %v", err)
		}
	}

	return dir, nil
}
