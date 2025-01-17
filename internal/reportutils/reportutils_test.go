package reportutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/reportutils"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestGetPeriodStart(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		period int

		wantErr error
	}{
		"Valid Period": {period: 500},

		"Invalid Negative Period": {period: -500, wantErr: reportutils.ErrInvalidPeriod},
		"Invalid Zero Period":     {period: 0, wantErr: reportutils.ErrInvalidPeriod},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := reportutils.GetPeriodStart(tc.period)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err, "got an unexpected error")

			require.IsType(t, int64(0), got)
		})
	}
}

func TestGetReportTime(t *testing.T) {
	tests := map[string]struct {
		path string

		wantErr bool
	}{
		"Valid Report Time":           {path: "1627847285.json", wantErr: false},
		"Valid Report Time with Path": {path: "/some/dir/1627847285.json", wantErr: false},
		"Alt Extension":               {path: "1627847285.txt", wantErr: false},

		"Empty File Name":               {path: ".json", wantErr: true},
		"Invalid Report Time":           {path: "invalid.json", wantErr: true},
		"Invalid Report Mixed":          {path: "i-1.json", wantErr: true},
		"Invalid Report Time with Path": {path: "/123/123/invalid.json", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := reportutils.GetReportTime(tc.path)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetReportTime should return the report time from the report path")
		})
	}
}

func TestGetReportPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       []string
		subDir      string
		subDirFiles []string
		time        int64
		period      int

		wantErr error
	}{
		"Empty Directory":        {time: 1, period: 500},
		"Files in subDir":        {subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}, time: 1, period: 500},
		"Empty subDir":           {subDir: "subdir", time: 1, period: 500},
		"Invalid File Extension": {files: []string{"1.txt", "2.txt"}, time: 1, period: 500},
		"Invalid File Names":     {files: []string{"i-1.json", "i-2.json", "i-3.json", "test.json", "one.json"}, time: -100, period: 500},

		"Specific Time Single Valid Report": {files: []string{"1.json", "2.json"}, time: 2, period: 1},
		"Negative Timestamp":                {files: []string{"-100.json", "-101.json"}, time: -150, period: 100},
		"Not Inclusive Period":              {files: []string{"1.json", "7.json"}, time: 2, period: 7},

		"Invalid Negative Period": {files: []string{"1.json", "7.json"}, time: 2, period: -7, wantErr: reportutils.ErrInvalidPeriod},
		"Invalid Zero Period":     {files: []string{"1.json", "7.json"}, time: 2, period: 0, wantErr: reportutils.ErrInvalidPeriod},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir, err := setupTmpDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			defer os.RemoveAll(dir)

			got, err := reportutils.GetReportPath(dir, tc.time, tc.period)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err, "got an unexpected error")

			if got != "" {
				got, err = filepath.Rel(dir, got)
				require.NoError(t, err, "failed to get relative path")
			}

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
