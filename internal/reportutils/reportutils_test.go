package reportutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/reportutils"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestGetReportPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       []string
		subDir      string
		subDirFiles []string
		time        uint64
		period      uint

		wantErr bool
	}{
		"Empty Directory":        {time: 1, period: 500},
		"Files in subDir":        {subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}, time: 1, period: 500},
		"Empty subDir":           {subDir: "subdir", time: 1, period: 500},
		"Invalid File Extension": {files: []string{"1.txt", "2.txt"}, time: 1, period: 500},
		"Invalid File Names":     {files: []string{"-1.json", "-2.json", "-3.json", "test.json", "one.json"}, time: 1, period: 500},

		"Specific Time: Single Valid Report": {files: []string{"1.json", "2.json"}, time: 2, period: 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir, err := setupTmpDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			defer os.RemoveAll(dir)

			got, err := reportutils.GetReportPath(dir, tc.time, tc.period)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGolden(t, got)
			require.Equal(t, want, got, "GetReportPath should return the most recent report within the period window")
		})
	}
}

func setupTmpDir(t *testing.T, files []string, subDir string, subDirFiles []string) (string, error) {
	t.Helper()

	dir, err := os.MkdirTemp("", "reportutils-test")
	if err != nil {
		return "", err
	}

	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := os.WriteFile(path, []byte{}, 0600); err != nil {
			return "", err
		}
	}

	if subDir != "" {
		subDirPath := filepath.Join(dir, subDir)
		if err := os.Mkdir(subDirPath, 0700); err != nil {
			return "", err
		}

		for _, file := range subDirFiles {
			path := filepath.Join(subDirPath, file)
			if err := os.WriteFile(path, []byte{}, 0600); err != nil {
				return "", err
			}
		}
	}

	return dir, nil
}