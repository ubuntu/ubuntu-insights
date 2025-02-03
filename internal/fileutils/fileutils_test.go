package fileutils_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
)

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		data            []byte
		fileExists      bool
		fileExistsPerms os.FileMode
		invalidDir      bool

		wantErrWin bool
		wantError  bool
	}{
		"Empty file":          {data: []byte{}},
		"Non-empty file":      {data: []byte("data")},
		"Override file":       {data: []byte("data"), fileExistsPerms: 0600, fileExists: true},
		"Override empty file": {data: []byte{}, fileExistsPerms: 0600, fileExists: true},

		"Existing empty file":     {data: []byte{}, fileExistsPerms: 0600, fileExists: true},
		"Existing non-empty file": {data: []byte("data"), fileExistsPerms: 0600, fileExists: true},

		"Override read-only file": {data: []byte("data"), fileExistsPerms: 0400, fileExists: true, wantError: runtime.GOOS == "windows"},
		"Override No Perms file":  {data: []byte("data"), fileExistsPerms: 0000, fileExists: true, wantError: runtime.GOOS == "windows"},
		"Invalid Dir":             {data: []byte("data"), invalidDir: true, wantError: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			oldFile := []byte("Old File!")
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "file")
			if tc.invalidDir {
				path = filepath.Join(path, "fake_dir")
			}

			if tc.fileExists {
				err := os.WriteFile(path, oldFile, tc.fileExistsPerms)
				require.NoError(t, err, "Setup: WriteFile should not return an error")
				t.Cleanup(func() { _ = os.Chmod(path, 0600) })
			}

			err := fileutils.AtomicWrite(path, tc.data)
			if tc.wantError || (tc.wantErrWin && runtime.GOOS == "windows") {
				require.Error(t, err, "AtomicWrite should return an error")

				// Check that the file was not overwritten
				if !tc.fileExists {
					return
				}

				if tc.invalidDir {
					path = filepath.Dir(path)
				}

				data, err := os.ReadFile(path)
				require.NoError(t, err, "ReadFile should not return an error")
				require.Equal(t, oldFile, data, "AtomicWrite should not overwrite the file")

				return
			}
			require.NoError(t, err, "AtomicWrite should not return an error")

			// Check that the file was written
			data, err := os.ReadFile(path)
			require.NoError(t, err, "ReadFile should not return an error")
			require.Equal(t, tc.data, data, "AtomicWrite should write the data to the file")
		})
	}
}
