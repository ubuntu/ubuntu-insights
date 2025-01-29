package report_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	report "github.com/ubuntu/ubuntu-insights/internal/report"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestGetPeriodStart(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		period int

		wantErr error
	}{
		"Valid Period": {period: 500},

		"Invalid Negative Period": {period: -500, wantErr: report.ErrInvalidPeriod},
		"Invalid Zero Period":     {period: 0, wantErr: report.ErrInvalidPeriod},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := report.GetPeriodStart(tc.period)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err, "got an unexpected error")

			require.IsType(t, int64(0), got)
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path string

		wantErr bool
	}{
		"Valid Report":           {path: "1627847285.json"},
		"Valid Report with Path": {path: "/some/dir/1627847285.json"},

		"Empty File Name":               {path: ".json", wantErr: true},
		"Invalid Report Time":           {path: "invalid.json", wantErr: true},
		"Invalid Report Mixed":          {path: "i-1.json", wantErr: true},
		"Invalid Report Time with Path": {path: "/123/123/invalid.json", wantErr: true},
		"Alt Extension":                 {path: "1627847285.txt", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := report.New(tc.path)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "New should return a new report object")
		})
	}
}

func TestGetForPeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       []string
		subDir      string
		subDirFiles []string
		time        int64
		period      int
		invalidDir  bool

		wantSpecificErr error
		wantGenericErr  bool
	}{
		"Empty Directory":        {time: 1, period: 500},
		"Files in subDir":        {subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}, time: 1, period: 500},
		"Empty subDir":           {subDir: "subdir", time: 1, period: 500},
		"Invalid File Extension": {files: []string{"1.txt", "2.txt"}, time: 1, period: 500},
		"Invalid File Names":     {files: []string{"i-1.json", "i-2.json", "i-3.json", "test.json", "one.json"}, time: -100, period: 500},

		"Specific Time Single Valid Report": {files: []string{"1.json", "2.json"}, time: 2, period: 1},
		"Negative Timestamp":                {files: []string{"-100.json", "-101.json"}, time: -150, period: 100},
		"Not Inclusive Period":              {files: []string{"1.json", "7.json"}, time: 2, period: 7},

		"Invalid Negative Period": {files: []string{"1.json", "7.json"}, time: 2, period: -7, wantSpecificErr: report.ErrInvalidPeriod},
		"Invalid Zero Period":     {files: []string{"1.json", "7.json"}, time: 2, period: 0, wantSpecificErr: report.ErrInvalidPeriod},

		"Invalid Dir": {period: 1, invalidDir: true, wantGenericErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir, err := setupTmpDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			if tc.invalidDir {
				dir = filepath.Join(dir, "invalid dir")
			}

			r, err := report.GetForPeriod(dir, time.Unix(tc.time, 0), tc.period)
			if tc.wantSpecificErr != nil {
				require.ErrorIs(t, err, tc.wantSpecificErr)
				return
			}
			if tc.wantGenericErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got := sanitizeReportPath(t, r, dir)

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetReportPath should return the most recent report within the period window")
		})
	}
}

func TestGetPerPeriod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       []string
		subDir      string
		subDirFiles []string
		period      int
		invalidDir  bool

		wantSpecificErr error
		wantGenericErr  bool
	}{
		"Empty Directory": {period: 500},
		"Files in subDir": {subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}, period: 500},

		"Invalid File Extension":   {files: []string{"1.txt", "2.txt"}, period: 500},
		"Invalid File Names":       {files: []string{"i-1.json", "i-2.json", "i-3.json", "test.json", "one.json"}, period: 500},
		"Mix of Valid and Invalid": {files: []string{"1.json", "2.json", "i-1.json", "i-2.json", "i-3.json", "test.json", "five.json"}, period: 500},

		"Get Newest of Period":             {files: []string{"1.json", "7.json"}, period: 100},
		"Multiple Consecutive Windows":     {files: []string{"1.json", "7.json", "101.json", "107.json", "201.json", "207.json"}, period: 100},
		"Multiple Non-Consecutive Windows": {files: []string{"1.json", "7.json", "101.json", "107.json", "251.json", "257.json"}, period: 50},
		"Get All Reports":                  {files: []string{"1.json", "2.json", "3.json", "101.json", "107.json", "251.json", "257.json"}, period: 1},

		"Invalid Negative Period": {files: []string{"1.json", "7.json"}, period: -7, wantSpecificErr: report.ErrInvalidPeriod},
		"Invalid Zero Period":     {files: []string{"1.json", "7.json"}, period: 0, wantSpecificErr: report.ErrInvalidPeriod},

		"Invalid Dir": {period: 1, invalidDir: true, wantGenericErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir, err := setupTmpDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			if tc.invalidDir {
				dir = filepath.Join(dir, "invalid dir")
			}

			reports, err := report.GetPerPeriod(dir, tc.period)
			if tc.wantSpecificErr != nil {
				require.ErrorIs(t, err, tc.wantSpecificErr)
				return
			}
			if tc.wantGenericErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got := make(map[int64]report.Report, len(reports))
			for n, r := range reports {
				got[n] = sanitizeReportPath(t, r, dir)
			}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetReports should return the most recent report within each period window")
		})
	}
}

func TestGetAll(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files       []string
		subDir      string
		subDirFiles []string
		invalidDir  bool

		wantErr bool
	}{
		"Empty Directory":          {},
		"Files in subDir":          {files: []string{"1.json", "2.json"}, subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}},
		"Invalid File Extension":   {files: []string{"1.txt", "2.txt"}},
		"Invalid File Names":       {files: []string{"i-1.json", "i-2.json", "i-3.json", "test.json", "one.json"}},
		"Mix of Valid and Invalid": {files: []string{"1.json", "2.json", "500.json", "i-1.json", "i-2.json", "i-3.json", "test.json", "five.json"}},

		"Invalid Dir": {invalidDir: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupTmpDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			if tc.invalidDir {
				dir = filepath.Join(dir, "invalid dir")
			}

			reports, err := report.GetAll(dir)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got := make([]report.Report, len(reports))
			for _, r := range reports {
				got = append(got, sanitizeReportPath(t, r, dir))
			}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetAllReports should return all reports in the directory")
		})
	}
}

func setupTmpDir(t *testing.T, files []string, subDir string, subDirFiles []string) (string, error) {
	t.Helper()

	dir := t.TempDir()

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

func sanitizeReportPath(t *testing.T, r report.Report, dir string) report.Report {
	t.Helper()
	if r.Path == "" {
		return r
	}

	fp, err := filepath.Rel(dir, r.Path)
	if err != nil {
		require.NoError(t, err, "failed to get relative path")
		return report.Report{}
	}
	return report.Report{Path: fp, Name: r.Name, TimeStamp: r.TimeStamp}
}
