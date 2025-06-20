package fileutils_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/shared/fileutils"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		data            []byte
		fileExists      bool
		fileExistsPerms os.FileMode
		invalidDir      bool

		wantError bool
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
			if tc.wantError {
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

func TestReadFileLogError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		file string
		want string

		log bool
	}{
		"No file": {file: "", want: "", log: true},

		"Empty file":  {file: "testdata/empty", want: ""},
		"Normal file": {file: "testdata/random", want: "Leftover vegetables!"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewMockHandler(slog.LevelDebug)

			got := fileutils.ReadFileLogError(tc.file, slog.New(&l))

			assert.Equal(t, tc.want, got, "ReadFileLogError should return the expected result")

			if tc.log {
				assert.NotEmpty(t, l.HandleCalls, "ReadFileLogError should log the expected errors")
			} else {
				assert.Empty(t, l.HandleCalls, "ReadFileLogError should not log unless expected")
			}
		})
	}
}

func TestConvertUnitToBytes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		unit  string
		value int

		want      int
		wantError bool
	}{
		"No unit": {unit: "", value: 256, want: 256},

		"Lowercase unit": {unit: "m", value: 2, want: 2048 * 1024},
		"Uppercase unit": {unit: "GB", value: 4, want: 4096 * 1024 * 1024},

		"Mixed unit": {unit: "TiB", value: 1, want: 1024 * 1024 * 1024 * 1024},

		"Odd unit": {unit: "gigahertz", value: 1024, want: 1024, wantError: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := fileutils.ConvertUnitToBytes(tc.unit, tc.value)

			assert.Equal(t, tc.want, got, "ConvertUnitToBytes should return the expected result")

			if tc.wantError {
				assert.Error(t, err, "ConvertUnitToBytes should error the expected errors")
			} else {
				assert.NoError(t, err, "ConvertUnitToBytes should not error unless expected")
			}
		})
	}
}
