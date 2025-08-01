package report_test

import (
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/report"
)

func TestGetPeriodStart(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		period uint32
		time   int64
	}{
		"Valid Period":             {period: 500, time: 100000},
		"Negative Time:":           {period: 500, time: -100000},
		"Non-Multiple Time":        {period: 500, time: 1051},
		"Zero Period Returns Time": {period: 0, time: 500},
		"Zero Period and Max Time": {period: 0, time: math.MaxInt64},
		"Zero Period and Min Time": {period: 0, time: math.MinInt64},
		"Max Period and Max Time":  {period: math.MaxUint32, time: math.MaxInt64},
		"Max Period and Min Time":  {period: math.MaxUint32, time: math.MinInt64},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := report.GetPeriodStart(tc.period, time.Unix(tc.time, 0))

			require.IsType(t, int64(0), got)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetPeriodStart should return the expect start of the period window")
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
		period      uint32
		invalidDir  bool

		wantErr bool
	}{
		"Empty Directory":        {time: 1, period: 500},
		"Files in subDir":        {subDir: "subdir", subDirFiles: []string{"1.json", "2.json"}, time: 1, period: 500},
		"Empty subDir":           {subDir: "subdir", time: 1, period: 500},
		"Invalid File Extension": {files: []string{"1.txt", "2.txt"}, time: 1, period: 500},
		"Invalid File Names":     {files: []string{"i-1.json", "i-2.json", "i-3.json", "test.json", "one.json"}, time: -100, period: 500},

		"Specific Time Single Valid Report": {files: []string{"1.json", "2.json"}, time: 2, period: 1},
		"Negative Timestamp":                {files: []string{"-100.json", "-101.json"}, time: -150, period: 100},
		"Not Inclusive Period":              {files: []string{"1.json", "7.json"}, time: 2, period: 7},
		"Lexical Order Check":               {files: []string{"5.json", "20.json"}, time: 10, period: 20},
		"Zero Period Returns Nothing":       {files: []string{"1.json", "7.json"}, time: 7, period: 0},

		// Error cases
		"Invalid Dir":        {period: 1, invalidDir: true, wantErr: true},
		"Max time overflows": {period: 1, time: math.MaxInt64, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir, err := setupNoDataDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			if tc.invalidDir {
				dir = filepath.Join(dir, "invalid dir")
			}

			r, err := report.GetForPeriod(slog.Default(), dir, time.Unix(tc.time, 0), tc.period)
			if tc.wantErr {
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
		"Lexical Order Check":      {files: []string{"500.json", "2.json", "120.json", "1.json", "0.json", "51230.json", "-1234.json", "121.json"}},

		"Invalid Dir": {invalidDir: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, err := setupNoDataDir(t, tc.files, tc.subDir, tc.subDirFiles)
			require.NoError(t, err, "Setup: failed to setup temporary directory")
			if tc.invalidDir {
				dir = filepath.Join(dir, "invalid dir")
			}

			reports, err := report.GetAll(slog.Default(), dir)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got := make([]report.Report, 0, len(reports))
			for _, r := range reports {
				got = append(got, sanitizeReportPath(t, r, dir))
			}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetAllReports should return all reports in the directory")
		})
	}
}

func TestMarkAsProcessed(t *testing.T) {
	t.Parallel()

	type got struct {
		Report   report.Report
		SrcFiles map[string]string
		DstFiles map[string]string
	}

	tests := map[string]struct {
		srcFile map[string]string
		dstFile map[string]string

		fileName     string
		data         []byte
		srcFilePerms os.FileMode
		dstFilePerms os.FileMode

		wantErr bool
	}{
		"Basic Move": {
			srcFile:      map[string]string{"1.json": `{"test": true}`},
			dstFile:      map[string]string{},
			fileName:     "1.json",
			data:         []byte(`{"test": true}`),
			srcFilePerms: os.FileMode(0o600),
			dstFilePerms: os.FileMode(0o600),
			wantErr:      false,
		},
		"Basic Move New Data": {
			srcFile:      map[string]string{"1.json": `{"test": true}`},
			dstFile:      map[string]string{},
			fileName:     "1.json",
			data:         []byte("new data"),
			srcFilePerms: os.FileMode(0o600),
			dstFilePerms: os.FileMode(0o600),
			wantErr:      false,
		},
		"Basic Move Overwrite": {
			srcFile:      map[string]string{"1.json": `{"test": true}`},
			dstFile:      map[string]string{"1.json": "old data"},
			fileName:     "1.json",
			data:         []byte("new data"),
			srcFilePerms: os.FileMode(0o600),
			dstFilePerms: os.FileMode(0o600),
			wantErr:      false,
		}, "SrcPerm None": {
			srcFile:      map[string]string{"1.json": `{"test": true}`},
			dstFile:      map[string]string{},
			fileName:     "1.json",
			data:         []byte(`{"test": true}`),
			srcFilePerms: os.FileMode(000),
			dstFilePerms: os.FileMode(0o600),
			wantErr:      testutils.IsUnixNonRoot(),
		}, "DstPerm None": {
			srcFile:      map[string]string{"1.json": `{"test": true}`},
			dstFile:      map[string]string{"1.json": "old data"},
			fileName:     "1.json",
			data:         []byte("new data"),
			srcFilePerms: os.FileMode(0o600),
			dstFilePerms: os.FileMode(000),
			wantErr:      runtime.GOOS == "windows",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rootDir, srcDir, dstDir := setupProcessingDirs(t)

			setupBasicDir(t, tc.srcFile, tc.srcFilePerms, srcDir)
			setupBasicDir(t, tc.dstFile, tc.dstFilePerms, dstDir)

			r, err := report.New(filepath.Join(srcDir, tc.fileName))
			require.NoError(t, err, "Setup: failed to create report object")

			r, err = r.MarkAsProcessed(dstDir, tc.data)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			dstDirContents, err := testutils.GetDirContents(t, dstDir, 2)
			require.NoError(t, err, "failed to get directory contents")

			srcDirContents, err := testutils.GetDirContents(t, srcDir, 2)
			require.NoError(t, err, "failed to get directory contents")

			r = sanitizeReportPath(t, r, rootDir)
			got := got{Report: r, SrcFiles: srcDirContents, DstFiles: dstDirContents}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.EqualExportedValues(t, want, got, "MarkAsProcessed should move the report to the processed directory")
		})
	}
}

func TestUndoProcessed(t *testing.T) {
	t.Parallel()

	type got struct {
		Report   report.Report
		SrcFiles map[string]string
		DstFiles map[string]string
	}

	tests := map[string]struct {
		srcFile map[string]string
		dstFile map[string]string

		fileName string
		data     []byte

		wantErr bool
	}{
		"Basic Move": {
			srcFile:  map[string]string{"1.json": `{"test": true}`},
			dstFile:  map[string]string{},
			fileName: "1.json",
			data:     []byte(`"new data"`),
			wantErr:  false,
		},
		"Basic Move New Data": {
			srcFile:  map[string]string{"1.json": `{"test": true}`},
			dstFile:  map[string]string{},
			fileName: "1.json",
			data:     []byte("new data"),
			wantErr:  false,
		},
		"Basic Move Overwrite": {
			srcFile:  map[string]string{"1.json": `{"test": true}`},
			dstFile:  map[string]string{"1.json": "old data"},
			fileName: "1.json",
			data:     []byte("new data"),
			wantErr:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rootDir, srcDir, dstDir := setupProcessingDirs(t)

			setupBasicDir(t, tc.srcFile, 0600, srcDir)
			setupBasicDir(t, tc.dstFile, 0600, dstDir)

			r, err := report.New(filepath.Join(srcDir, tc.fileName))
			require.NoError(t, err, "Setup: failed to create report object")

			r, err = r.MarkAsProcessed(dstDir, tc.data)
			require.NoError(t, err, "Setup: failed to mark report as processed")

			r, err = r.UndoProcessed()
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			dstDirContents, err := testutils.GetDirContents(t, dstDir, 2)
			require.NoError(t, err, "failed to get directory contents")

			srcDirContents, err := testutils.GetDirContents(t, srcDir, 2)
			require.NoError(t, err, "failed to get directory contents")

			r = sanitizeReportPath(t, r, rootDir)
			got := got{Report: r, SrcFiles: srcDirContents, DstFiles: dstDirContents}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.EqualExportedValues(t, want, got, "UndoProcessed should move the report to the processed directory")
		})
	}
}

func TestUndoProcessedNoStash(t *testing.T) {
	t.Parallel()

	r, err := report.New("1.json")
	require.NoError(t, err, "Setup: failed to create report object")

	_, err = r.UndoProcessed()
	require.Error(t, err, "UndoProcessed should return an error if the report has not been marked as processed")
}

func TestMarkAsProcessedNoFile(t *testing.T) {
	t.Parallel()

	_, srcDir, dstDir := setupProcessingDirs(t)
	r, err := report.New(filepath.Join(srcDir, "1.json"))
	require.NoError(t, err, "Setup: failed to create report object")

	_, err = r.MarkAsProcessed(dstDir, []byte(`"new data"`))
	require.Error(t, err, "MarkAsProcessed should return an error if the report file does not exist")
}

func TestUndoProcessedNoFile(t *testing.T) {
	t.Parallel()

	_, srcDir, dstDir := setupProcessingDirs(t)
	reportPath := filepath.Join(srcDir, "1.json")
	require.NoError(t, os.WriteFile(reportPath, []byte(`{"test": true}`), 0600), "Setup: failed to write report file")
	r, err := report.New(reportPath)
	require.NoError(t, err, "Setup: failed to create report object")

	r, err = r.MarkAsProcessed(dstDir, []byte(`"new data"`))
	require.NoError(t, err, "Setup: failed to mark report as processed")

	require.NoError(t, os.Remove(r.Path), "Setup: failed to remove report file")

	_, err = r.UndoProcessed()
	require.Error(t, err, "UndoProcessed should return an error if the report file does not exist")
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files map[string]string
		file  string

		wantErr bool
	}{
		"Basic Read":     {files: map[string]string{"1.json": `{"test": true}`}, file: "1.json"},
		"Multiple Files": {files: map[string]string{"1.json": `{"test": true}`, "2.json": `{"test": false}`}, file: "1.json"},

		"Empty File":   {files: map[string]string{"1.json": ""}, file: "1.json", wantErr: true},
		"Invalid JSON": {files: map[string]string{"1.json": `{"test":::`}, file: "1.json", wantErr: true},
		"No File":      {files: map[string]string{}, file: "1.json", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, srcDir, _ := setupProcessingDirs(t)

			setupBasicDir(t, tc.files, 0600, srcDir)

			r, err := report.New(filepath.Join(srcDir, tc.file))
			require.NoError(t, err, "Setup: failed to create report object")

			data, err := r.ReadJSON()
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got := string(data)
			want := testutils.LoadWithUpdateFromGolden(t, got)
			require.Equal(t, want, got, "ReadJSON should return the data from the report file")
		})
	}
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files      []string
		maxReports uint32
		noDir      bool

		wantErr bool
	}{
		"Empty Directory": {maxReports: 5},
		"Less than maxReports": {
			files:      []string{"1.json", "2.json", "3.json", "4.json"},
			maxReports: 5,
		},
		"Equal to maxReports": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json"},
			maxReports: 5,
		},
		"More than maxReports": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json", "6.json"},
			maxReports: 5,
		},
		"Unordered Creation": {
			files:      []string{"-100.json", "-2.json", "0.json", "150.json", "151.json", "120.json", "121.json", "122.json"},
			maxReports: 5,
		},
		"Non-Report Files": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json", "6.json", "7.txt", "8.txt", "9.txt", "seven.json"},
			maxReports: 6,
		},
		"Zero maxReports": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json", "6.json"},
			maxReports: 0,
		},
		"Bad Path": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json", "6.json"},
			maxReports: 5,
			noDir:      true,
			wantErr:    true,
		},
		"Max Reports Overflow": {
			files:      []string{"1.json", "2.json", "3.json", "4.json", "5.json", "6.json"},
			maxReports: math.MaxInt32 + 1,
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			baseDir := t.TempDir()

			for _, file := range tc.files {
				path := filepath.Join(baseDir, file)
				require.NoError(t, os.WriteFile(path, []byte(`{"test": true}`), 0o600), "Setup: failed to write report file")
			}

			dir := baseDir
			if tc.noDir {
				dir = filepath.Join(baseDir, "invalid dir")
			}
			err := report.Cleanup(slog.Default(), dir, tc.maxReports)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got, err := testutils.GetDirContents(t, baseDir, 2)
			require.NoError(t, err, "failed to get directory contents")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "Cleanup should remove the oldest reports, keeping the most recent maxReports")
		})
	}
}

func setupProcessingDirs(t *testing.T) (rootDir, srcDir, dstDir string) {
	t.Helper()
	rootDir = t.TempDir()
	srcDir = filepath.Join(rootDir, "src")
	dstDir = filepath.Join(rootDir, "dst")
	err := os.MkdirAll(srcDir, 0700)
	require.NoError(t, err, "Setup: failed to create source directory")
	err = os.MkdirAll(dstDir, 0700)
	require.NoError(t, err, "Setup: failed to create destination directory")
	return rootDir, srcDir, dstDir
}

func setupBasicDir(t *testing.T, files map[string]string, perms os.FileMode, dir string) {
	t.Helper()
	for file, data := range files {
		path := filepath.Join(dir, file)
		err := os.WriteFile(path, []byte(data), perms)
		require.NoError(t, err, "Setup: failed to write file")
		t.Cleanup(func() {
			_ = os.Chmod(path, os.FileMode(0600))
		})
	}
}

func setupNoDataDir(t *testing.T, files []string, subDir string, subDirFiles []string) (string, error) {
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
	return report.Report{Path: filepath.ToSlash(fp), Name: r.Name, TimeStamp: r.TimeStamp}
}
