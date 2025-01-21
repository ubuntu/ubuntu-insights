package fileutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		data       []byte
		fileExists bool

		wantError bool
	}{
		"Empty file":          {data: []byte{}, fileExists: false},
		"Non-empty file":      {data: []byte("data"), fileExists: false},
		"Override file":       {data: []byte("data"), fileExists: true},
		"Override empty file": {data: []byte{}, fileExists: true},

		"Existing empty file":     {data: []byte{}, fileExists: true},
		"Existing non-empty file": {data: []byte("data"), fileExists: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			oldFile := []byte("Old File!")
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "file")
			if tc.fileExists {
				err := fileutils.AtomicWrite(path, oldFile)
				require.NoError(t, err, "Setup: AtomicWrite should not return an error")
			}

			err := fileutils.AtomicWrite(path, tc.data)
			if tc.wantError {
				require.Error(t, err, "AtomicWrite should return an error")

				// Check that the file was not overwritten
				data, err := os.ReadFile(path)
				require.NoError(t, err, "ReadFile should not return an error")
				require.Equal(t, oldFile, data, "AtomicWrite should not overwrite the file")
			} else {
				require.NoError(t, err, "AtomicWrite should not return an error")

				// Check that the file was written
				data, err := os.ReadFile(path)
				require.NoError(t, err, "ReadFile should not return an error")
				require.Equal(t, tc.data, data, "AtomicWrite should write the data to the file")
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fileExists      bool
		parentDirIsFile bool

		wantExists bool
		wantError  bool
	}{
		"Returns_true_when_file_exists":                      {fileExists: true, wantExists: true},
		"Returns_false_when_file_does_not_exist":             {fileExists: false, wantExists: false},
		"Returns_false_when_parent_directory_does_not_exist": {fileExists: false, wantExists: false},

		"Error_when_parent_directory_is_a_file": {parentDirIsFile: true, wantError: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "file")
			if tc.fileExists {
				err := fileutils.AtomicWrite(path, []byte{})
				require.NoError(t, err, "AtomicWrite should not return an error")
			}
			if tc.parentDirIsFile {
				path = filepath.Join(tempDir, "file", "file")
				err := fileutils.AtomicWrite(filepath.Join(tempDir, "file"), []byte{})
				require.NoError(t, err, "AtomicWrite should not return an error")
			}

			exists, err := fileutils.FileExists(path)
			if tc.wantError {
				require.Error(t, err, "FileExists should return an error")
			} else {
				require.NoError(t, err, "FileExists should not return an error")
			}
			require.Equal(t, tc.wantExists, exists, "FileExists should return the expected result")
		})
	}
}
