package uploader_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

type mockTimeProvider struct {
	currentTime int64
}

func (m mockTimeProvider) NowUnix() int64 {
	return m.currentTime
}

var (
	cmSErr     = testConsentManager{sErr: fmt.Errorf("consent error")}
	cmTrueSErr = testConsentManager{sState: true, gState: true, sErr: fmt.Errorf("consent error")}
	cmGErr     = testConsentManager{gErr: fmt.Errorf("consent error")}
	cmTrueGErr = testConsentManager{gState: true, gErr: fmt.Errorf("consent error")}
	cmTrue     = testConsentManager{sState: true, gState: true}
	cmFalse    = testConsentManager{sState: false, gState: false}
	cmSTrue    = testConsentManager{sState: true, gState: false}
	cmGTrue    = testConsentManager{sState: false, gState: true}
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cm     testConsentManager
		source string
		minAge uint
		dryRun bool

		wantErr bool
	}{
		"Valid":        {cm: cmTrue, source: "source", minAge: 5, dryRun: true},
		"Zero Min Age": {cm: cmTrue, source: "source", minAge: 0},

		"Empty Source":    {cm: cmTrue, source: "", wantErr: true},
		"Minage Overflow": {cm: cmTrue, source: "source", minAge: math.MaxUint64, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := uploader.New(tc.cm, tc.source, tc.minAge, tc.dryRun)
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

	const mockTime = 10

	tests := map[string]struct {
		localFiles, uploadedFiles map[string]reportType
		dummy                     bool
		serverResponse            int
		serverOffline             bool
		url                       string
		invalidDir                bool

		cm     testConsentManager
		minAge uint
		dryRun bool
		force  bool

		wantErr bool
	}{
		"No Reports":            {cm: cmTrue, serverResponse: http.StatusOK},
		"No Reports with Dummy": {dummy: true, cm: cmTrue, serverResponse: http.StatusOK},
		"Single Upload":         {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, serverResponse: http.StatusOK},
		"Multi Upload":          {localFiles: map[string]reportType{"1.json": normal, "5.json": normal}, cm: cmTrue, serverResponse: http.StatusOK},
		"Min Age":               {localFiles: map[string]reportType{"1.json": normal, "9.json": normal}, cm: cmTrue, minAge: 5, serverResponse: http.StatusOK},
		"Future Timestamp":      {localFiles: map[string]reportType{"1.json": normal, "11.json": normal}, cm: cmTrue, serverResponse: http.StatusOK},
		"Duplicate Upload":      {localFiles: map[string]reportType{"1.json": normal}, uploadedFiles: map[string]reportType{"1.json": badContent}, cm: cmTrue, serverResponse: http.StatusAccepted},
		"Bad Content":           {localFiles: map[string]reportType{"1.json": badContent}, cm: cmTrue, serverResponse: http.StatusOK},

		"Consent Manager Source Error":              {localFiles: map[string]reportType{"1.json": normal}, cm: cmSErr, serverResponse: http.StatusOK, wantErr: true},
		"Consent Manager Source Error with True":    {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrueSErr, serverResponse: http.StatusOK, wantErr: true},
		"Consent Manager Global Error":              {localFiles: map[string]reportType{"1.json": normal}, cm: cmGErr, serverResponse: http.StatusOK, wantErr: true},
		"Consent Manager Global Error with True":    {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrueGErr, serverResponse: http.StatusOK, wantErr: true},
		"Consent Manager False":                     {localFiles: map[string]reportType{"1.json": normal}, cm: cmFalse, serverResponse: http.StatusOK},
		"Consent Manager Global True, Source False": {localFiles: map[string]reportType{"1.json": normal}, cm: cmGTrue, serverResponse: http.StatusOK},
		"Consent Manager Global False, Source True": {localFiles: map[string]reportType{"1.json": normal}, cm: cmSTrue, serverResponse: http.StatusOK},

		"Force CM False":  {localFiles: map[string]reportType{"1.json": normal}, cm: cmFalse, force: true, serverResponse: http.StatusOK},
		"Force Min Age":   {localFiles: map[string]reportType{"1.json": normal, "9.json": normal}, cm: cmTrue, minAge: 5, force: true, serverResponse: http.StatusOK},
		"Force Duplicate": {localFiles: map[string]reportType{"1.json": normal}, uploadedFiles: map[string]reportType{"1.json": badContent}, cm: cmTrue, force: true, serverResponse: http.StatusOK},

		"OptOut Payload CM True":  {localFiles: map[string]reportType{"1.json": optOut}, cm: cmTrue, serverResponse: http.StatusOK},
		"OptOut Payload CM False": {localFiles: map[string]reportType{"1.json": optOut}, cm: cmFalse, serverResponse: http.StatusOK},

		"Dry run": {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, dryRun: true},

		"Bad URL":        {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, url: "http://a b.com/", wantErr: true},
		"Bad Response":   {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, serverResponse: http.StatusForbidden},
		"Offline Server": {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, serverOffline: true},

		"Invalid Directory": {localFiles: map[string]reportType{"1.json": normal}, cm: cmTrue, invalidDir: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := setupTmpDir(t, tc.localFiles, tc.uploadedFiles, tc.dummy)

			if !tc.serverOffline {
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.serverResponse)
				}))
				t.Cleanup(func() { ts.Close() })
				if tc.url == "" {
					tc.url = ts.URL
				}
			}

			if tc.invalidDir {
				require.NoError(t, os.RemoveAll(filepath.Join(dir, "local")), "Setup: failed to remove local directory")
			}

			mgr, err := uploader.New(tc.cm, "source", tc.minAge, tc.dryRun,
				uploader.WithBaseServerURL(tc.url), uploader.WithCachePath(dir), uploader.WithTimeProvider(mockTimeProvider{currentTime: mockTime}))
			require.NoError(t, err, "Setup: failed to create new uploader manager")

			err = mgr.Upload(tc.force)
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

func setupTmpDir(t *testing.T, localFiles, uploadedFiles map[string]reportType, dummy bool) string {
	t.Helper()
	dir := t.TempDir()

	localDir := filepath.Join(dir, "local")
	uploadedDir := filepath.Join(dir, "uploaded")
	require.NoError(t, os.Mkdir(localDir, 0750), "Setup: failed to create local directory")
	require.NoError(t, os.Mkdir(uploadedDir, 0750), "Setup: failed to create uploaded directory")

	if dummy {
		copyDummyData(t, "testdata/test_source", dir, localDir, uploadedDir)
	}

	writeFiles(t, localDir, localFiles)
	writeFiles(t, uploadedDir, uploadedFiles)

	return dir
}

func copyDummyData(t *testing.T, sourceDir, dir, localDir, uploadedDir string) {
	t.Helper()
	require.NoError(t, testutils.CopyDir(sourceDir, dir), "Setup: failed to copy dummy data to temporary directory")
	require.NoError(t, testutils.CopyDir(sourceDir, localDir), "Setup: failed to copy dummy data to local")
	require.NoError(t, testutils.CopyDir(sourceDir, uploadedDir), "Setup: failed to copy dummy data to uploaded")
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

type testConsentManager struct {
	sState bool
	gState bool
	sErr   error
	gErr   error
}

func (m testConsentManager) GetConsentState(source string) (bool, error) {
	if source != "" {
		return m.sState, m.sErr
	}
	return m.gState, m.gErr
}
