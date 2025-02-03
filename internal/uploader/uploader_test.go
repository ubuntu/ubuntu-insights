package uploader_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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

		"Empty Source":    {consent: cTrue, source: "", wantErr: true},
		"Minage Overflow": {consent: cTrue, source: "source", minAge: math.MaxUint64, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := uploader.New(tc.consent, tc.source, tc.minAge, tc.dryRun)
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
		defaultResponse = http.StatusOK
		source          = "source"
	)

	tests := map[string]struct {
		lFiles, uFiles map[string]reportType
		dummy          bool
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
		"No Reports":            {consent: cTrue},
		"No Reports with Dummy": {dummy: true, consent: cTrue},
		"Single Upload":         {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue},
		"Multi Upload":          {lFiles: map[string]reportType{"1.json": normal, "5.json": normal}, consent: cTrue},
		"Min Age":               {lFiles: map[string]reportType{"1.json": normal, "9.json": normal}, consent: cTrue, minAge: 5},
		"Future Timestamp":      {lFiles: map[string]reportType{"1.json": normal, "11.json": normal}, consent: cTrue},
		"Duplicate Upload":      {lFiles: map[string]reportType{"1.json": normal}, uFiles: map[string]reportType{"1.json": badContent}, consent: cTrue},
		"Bad Content":           {lFiles: map[string]reportType{"1.json": badContent}, consent: cTrue},
		"No Directory":          {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, rmLocal: true},

		"Consent Manager Source Error":           {lFiles: map[string]reportType{"1.json": normal}, consent: cErr, wantErr: true},
		"Consent Manager Source Error with True": {lFiles: map[string]reportType{"1.json": normal}, consent: cErrTrue, wantErr: true},
		"Consent Manager False":                  {lFiles: map[string]reportType{"1.json": normal}, consent: cFalse},

		"Force CM False":  {lFiles: map[string]reportType{"1.json": normal}, consent: cFalse, force: true},
		"Force Min Age":   {lFiles: map[string]reportType{"1.json": normal, "9.json": normal}, consent: cTrue, minAge: 5, force: true},
		"Force Duplicate": {lFiles: map[string]reportType{"1.json": normal}, uFiles: map[string]reportType{"1.json": badContent}, consent: cTrue, force: true},

		"OptOut Payload CM True":  {lFiles: map[string]reportType{"1.json": optOut}, consent: cTrue},
		"OptOut Payload CM False": {lFiles: map[string]reportType{"1.json": optOut}, consent: cFalse},

		"Dry run": {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, dryRun: true},

		"Bad URL":        {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, url: "http://a b.com/", wantErr: true},
		"Bad Response":   {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, serverResponse: http.StatusForbidden},
		"Offline Server": {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, serverOffline: true},
		"No Permissions": {lFiles: map[string]reportType{"1.json": normal}, consent: cTrue, noPerms: true, wantErr: runtime.GOOS != "windows", skipContentCheck: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := setupTmpDir(t, tc.lFiles, tc.uFiles, source, tc.dummy)

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

			mgr, err := uploader.New(tc.consent, source, tc.minAge, tc.dryRun,
				uploader.WithBaseServerURL(tc.url), uploader.WithCachePath(dir), uploader.WithTimeProvider(uploader.MockTimeProvider{CurrentTime: mockTime}))
			require.NoError(t, err, "Setup: failed to create new uploader manager")

			err = mgr.Upload(tc.force)
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

func setupTmpDir(t *testing.T, localFiles, uploadedFiles map[string]reportType, source string, dummy bool) string {
	t.Helper()
	dir := t.TempDir()

	localDir := filepath.Join(dir, source, constants.LocalFolder)
	uploadedDir := filepath.Join(dir, source, constants.UploadedFolder)
	require.NoError(t, os.MkdirAll(localDir, 0750), "Setup: failed to create local directory")
	require.NoError(t, os.MkdirAll(uploadedDir, 0750), "Setup: failed to create uploaded directory")

	if dummy {
		copyDummyData(t, "testdata/test_source", dir, localDir, uploadedDir)
	}

	writeFiles(t, localDir, localFiles)
	writeFiles(t, uploadedDir, uploadedFiles)

	return dir
}

func copyDummyData(t *testing.T, sourceDir, dir, localDir, uploadedDir string) {
	t.Helper()
	require.NoError(t, testutils.CopyDir(t, sourceDir, dir), "Setup: failed to copy dummy data to temporary directory")
	require.NoError(t, testutils.CopyDir(t, sourceDir, localDir), "Setup: failed to copy dummy data to local")
	require.NoError(t, testutils.CopyDir(t, sourceDir, uploadedDir), "Setup: failed to copy dummy data to uploaded")
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
