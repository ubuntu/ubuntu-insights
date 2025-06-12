package uploader_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
	"github.com/ubuntu/ubuntu-insights/internal/uploader"
)

type reportType any

var (
	normal     reportType = struct{ Content string }{Content: "normal content"}
	optOut                = constants.OptOutJSON
	badContent            = `bad content`
)

var (
	cTrue    = testConsentChecker{consent: true}
	cFalse   = testConsentChecker{consent: false}
	cErr     = testConsentChecker{err: fmt.Errorf("consent error")}
	cErrTrue = testConsentChecker{consent: true, err: fmt.Errorf("consent error")}
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		consent testConsentChecker
		source  string
		minAge  uint
		dryRun  bool

		wantErr bool
	}{
		"Valid":        {consent: cTrue, source: "source", minAge: 5, dryRun: true},
		"Zero Min Age": {consent: cTrue, source: "source", minAge: 0},

		"Minage Overflow": {consent: cTrue, source: "source", minAge: math.MaxUint64, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := uploader.New(slog.Default(), tc.consent, "", tc.minAge, tc.dryRun)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpload(t *testing.T) {
	t.Parallel()

	const (
		mockTime        = 10
		defaultResponse = http.StatusAccepted
		source          = "source"
	)

	tests := map[string]struct {
		lFiles, uFiles map[string]reportType
		serverResponse int
		serverOffline  bool
		url            string
		rmLocal        bool
		noPerms        bool

		consent testConsentChecker
		minAge  uint
		dryRun  bool
		force   bool

		skipContentCheck bool
		wantErr          bool
	}{
		// Basic Tests
		"Does nothing when no reports to be uploaded":           {consent: cTrue},
		"Does nothing when the locals files dir does not exist": {consent: cTrue, rmLocal: true},
		"Finds and uploads single valid report":                 {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue},
		"Finds and uploads multiple valid reports":              {lFiles: map[string]reportType{"1.json": normal, "5.json": normal}, consent: cTrue},

		// Timestamp related tests
		"Ignores reports with future timestamps":               {lFiles: map[string]reportType{"1.json": normal, "11.json": normal}, consent: cTrue},
		"Ignores immature reports, but uploads mature reports": {lFiles: map[string]reportType{"1.json": normal, "9.json": normal}, consent: cTrue, minAge: 5},

		// Consent Tests
		"Errors when consent errors":                  {lFiles: map[string]reportType{"1.json": normal}, consent: cErr, wantErr: true},
		"Errors when consent errors and returns true": {lFiles: map[string]reportType{"1.json": normal}, consent: cErrTrue, wantErr: true},
		"Sends OptOut when consent is false":          {lFiles: map[string]reportType{"1.json": normal}, consent: cFalse},
		"Sends OptOut Payload when consent true":      {lFiles: map[string]reportType{"1.json": optOut}, consent: cTrue},
		"Sends OptOut payload when consent false":     {lFiles: map[string]reportType{"1.json": optOut}, consent: cFalse},

		// Force Tests
		"Force does not override consent false": {
			lFiles: map[string]reportType{"1.json": normal}, consent: cFalse, force: true},
		"Force overrides min age": {
			lFiles: map[string]reportType{"1.json": normal, "9.json": normal}, consent: cTrue, minAge: 5, force: true},
		"Force overrides duplicate reports": {
			lFiles: map[string]reportType{"1.json": normal}, uFiles: map[string]reportType{"1.json": badContent}, consent: cTrue, force: true},

		// Dry Run Tests
		"Upload does not send or cleanup when dry run": {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, dryRun: true},

		// Upload local report errors
		"Errors when local report is invalid JSON": {
			lFiles: map[string]reportType{"1.json": badContent}, consent: cTrue, wantErr: true},
		"Errors when a report has already been uploaded": {
			lFiles: map[string]reportType{"1.json": normal}, uFiles: map[string]reportType{"1.json": badContent}, consent: cTrue, wantErr: true},
		"Errors when not on Windows with bad file permissions": {
			lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, noPerms: true, wantErr: testutils.IsUnixNonRoot(), skipContentCheck: true},

		// Server errors
		"Errors when given a bad URL":               {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, url: "http://a b.com/", wantErr: true},
		"Errors when server returns a bad response": {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, serverResponse: http.StatusForbidden, wantErr: true},
		"Errors when the server is unreachable":     {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, serverOffline: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := setupTmpDir(t, tc.lFiles, tc.uFiles, source)

			if tc.serverResponse == 0 {
				tc.serverResponse = defaultResponse
			}

			if !tc.serverOffline {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.serverResponse)
				}))
				t.Cleanup(func() { ts.Close() })
				if tc.url == "" {
					tc.url = ts.URL
				}
			}

			localDir := filepath.Join(dir, source, constants.LocalFolder)
			if tc.rmLocal {
				require.NoError(t, os.RemoveAll(localDir), "Setup: failed to remove local directory")
			} else if tc.noPerms {
				require.NoError(t, os.Chmod(localDir, 0), "Setup: failed to remove local directory")
				t.Cleanup(func() { require.NoError(t, os.Chmod(localDir, 0750), "Cleanup: failed to restore permissions") }) //nolint:gosec //0750 is fine for folders
			}

			mgr, err := uploader.New(slog.Default(), tc.consent, dir, tc.minAge, tc.dryRun,
				uploader.WithBaseServerURL(tc.url), uploader.WithTimeProvider(uploader.MockTimeProvider{CurrentTime: mockTime}))
			require.NoError(t, err, "Setup: failed to create new uploader manager")

			err = mgr.Upload(source, tc.force)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.noPerms {
				//nolint:gosec //0750 is fine for folders
				require.NoError(t, os.Chmod(localDir, 0750), "Post: failed to restore permissions")
			}

			if tc.skipContentCheck {
				return
			}

			got, err := testutils.GetDirContents(t, dir, 3)
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.EqualValues(t, want, got)
		})
	}
}

func TestBackoffUpload(t *testing.T) {
	t.Parallel()

	const (
		mockTime        = 10
		defaultResponse = http.StatusAccepted
		source          = "source"
	)

	tests := map[string]struct {
		lFiles, uFiles  map[string]reportType
		initialResponse int // If initial response is 0 or lower, the server will not respond
		badCount        int // Number of initialResponses the server will send before an OK response
		serverOffline   bool

		rmLocal       bool // Remove the local directory
		readOnlyFiles []string

		consent testConsentChecker // Default cTrue
		minAge  uint
		dryRun  bool
		force   bool

		skipContentCheck bool
		wantErr          bool
	}{
		// Basic Tests
		"Does nothing when no reports to be uploaded":           {consent: cTrue},
		"Does nothing when the locals files dir does not exist": {consent: cTrue, rmLocal: true},
		"Finds and uploads single valid report":                 {consent: cTrue, lFiles: map[string]reportType{"1.json": normal}},
		"Finds and uploads multiple valid reports":              {consent: cTrue, lFiles: map[string]reportType{"1.json": normal, "2.json": optOut, "5.json": normal}},
		"Respects consent and sends OptOut":                     {consent: cFalse, lFiles: map[string]reportType{"1.json": normal, "2.json": optOut}},

		"Dry run does not send": {consent: cTrue, lFiles: map[string]reportType{"1.json": normal}, dryRun: true, serverOffline: true},

		// Timeout Tests
		"Retries when no response": {
			consent: cTrue, lFiles: map[string]reportType{"1.json": normal}, initialResponse: -1, badCount: 2},
		"Retries when server returns a bad response": {
			consent: cTrue, lFiles: map[string]reportType{"1.json": normal}, initialResponse: http.StatusForbidden, badCount: 2},

		// Timeout Error Tests
		"Gives up after too many no responses retries": {
			consent: cTrue, lFiles: map[string]reportType{"1.json": normal}, serverOffline: true, wantErr: true},
		"Gives up after too many bad responses": {
			consent: cTrue, lFiles: map[string]reportType{"1.json": normal}, initialResponse: http.StatusForbidden, badCount: 500, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := setupTmpDir(t, tc.lFiles, tc.uFiles, source)

			if tc.initialResponse == 0 {
				tc.initialResponse = defaultResponse
			}

			var bc atomic.Int64
			bc.Store(int64(tc.badCount))
			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bc.Add(-1) >= 0 {
					// Unresponsive server if initialResponseCode < 0
					if tc.initialResponse < 0 {
						time.Sleep(4 * time.Second)
						return
					}
					w.WriteHeader(tc.initialResponse)
					return
				}
				w.WriteHeader(http.StatusAccepted)
			}))
			if !tc.serverOffline {
				t.Cleanup(func() { ts.Close() })
				ts.Start()
			}
			url := ts.URL

			localDir := filepath.Join(dir, source, constants.LocalFolder)
			if tc.rmLocal {
				require.NoError(t, os.RemoveAll(localDir), "Setup: failed to remove local directory")
			}

			for _, file := range tc.readOnlyFiles {
				testutils.MakeReadOnly(t, filepath.Join(dir, source, file))
			}

			mgr, err := uploader.New(slog.Default(), tc.consent, dir, tc.minAge, tc.dryRun,
				uploader.WithBaseServerURL(url),
				uploader.WithTimeProvider(uploader.MockTimeProvider{CurrentTime: mockTime}),
				uploader.WithInitialRetryPeriod(100*time.Millisecond),
				uploader.WithMaxRetryPeriod(4*time.Second),
				uploader.WithResponseTimeout(2*time.Second),
				uploader.WithMaxAttempts(4))
			require.NoError(t, err, "Setup: failed to create new uploader manager")

			err = mgr.BackoffUpload(source, tc.force)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got, err := testutils.GetDirContents(t, dir, 3)
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.EqualValues(t, want, got)
		})
	}
}

func TestGetAllSources(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		folders       []string
		files         []string
		subDirs       []string
		subFiles      []string
		noFolderPerms bool
		noFilePerms   bool

		wantErr bool
	}{
		"Empty":                                {},
		"Single Source":                        {folders: []string{"source"}},
		"Multiple Sources":                     {folders: []string{"source1", "source2"}},
		"Source with Files":                    {folders: []string{"source"}, files: []string{"1.json", "2.json"}},
		"Source with Subdirectories":           {folders: []string{"source"}, subDirs: []string{"sub1", "sub2"}},
		"Source with Subdirectories and Files": {folders: []string{"source"}, subDirs: []string{"sub1", "sub2"}, subFiles: []string{"1.json", "2.json"}},

		"Source with No Folder Perms": {folders: []string{"source"}, noFolderPerms: true, wantErr: testutils.IsUnixNonRoot()},
		"Source with No File Perms":   {folders: []string{"source"}, files: []string{"1.json", "2.json"}, noFilePerms: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			folderPerms := os.FileMode(0750)
			if tc.noFolderPerms {
				folderPerms = 0
			}
			filePerms := os.FileMode(0600)
			if tc.noFilePerms {
				filePerms = 0
			}

			for _, folder := range tc.folders {
				fPath := filepath.Join(dir, folder)
				require.NoError(t, os.Mkdir(fPath, folderPerms), "Setup: failed to create source directory")

				for _, subDir := range tc.subDirs {
					sdPath := filepath.Join(fPath, subDir)
					require.NoError(t, os.Mkdir(sdPath, folderPerms), "Setup: failed to create subdirectory")
					t.Cleanup(func() { assert.NoError(t, os.Chmod(sdPath, 0750), "Cleanup: Failed to restore folder perms") }) //nolint:gosec //0750 is fine for folders
				}

				for _, file := range tc.files {
					fPath := filepath.Join(fPath, file)
					require.NoError(t, os.WriteFile(fPath, []byte{}, filePerms), "Setup: failed to create file")
					t.Cleanup(func() { assert.NoError(t, os.Chmod(fPath, 0600), "Cleanup: Failed to restore file perms") })
				}
			}

			for _, file := range tc.files {
				fPath := filepath.Join(dir, file)
				require.NoError(t, os.WriteFile(fPath, []byte{}, filePerms), "Setup: failed to create file")
				t.Cleanup(func() { assert.NoError(t, os.Chmod(fPath, 0600), "Cleanup: Failed to restore file perms") })
			}

			got, err := uploader.GetAllSources(dir)
			if tc.wantErr {
				require.Error(t, err, "GetAllSources should return an error")
				return
			}
			require.NoError(t, err, "GetAllSources should not return an error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetAllSources should return expected sources")
		})
	}
}

func setupTmpDir(t *testing.T, localFiles, uploadedFiles map[string]reportType, source string) string {
	t.Helper()
	dir := t.TempDir()

	localDir := filepath.Join(dir, source, constants.LocalFolder)
	uploadedDir := filepath.Join(dir, source, constants.UploadedFolder)
	require.NoError(t, os.MkdirAll(localDir, 0750), "Setup: failed to create local directory")
	require.NoError(t, os.MkdirAll(uploadedDir, 0750), "Setup: failed to create uploaded directory")

	writeFiles(t, localDir, localFiles)
	writeFiles(t, uploadedDir, uploadedFiles)

	return dir
}

func writeFiles(t *testing.T, targetDir string, files map[string]reportType) {
	t.Helper()
	for file, content := range files {
		var data []byte
		var err error

		switch v := content.(type) {
		case string:
			data = []byte(v)
		default:
			data, err = json.Marshal(content)
			require.NoError(t, err, "Setup: failed to marshal sample data")
		}
		require.NoError(t, fileutils.AtomicWrite(filepath.Join(targetDir, file), data), "Setup: failed to write file")
	}
}

type testConsentChecker struct {
	consent bool
	err     error
}

func (m testConsentChecker) HasConsent(source string) (bool, error) {
	return m.consent, m.err
}
