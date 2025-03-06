package uploader

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/report"
)

func TestUploadBadFile(t *testing.T) {
	t.Parallel()
	basicContent := `{"Content":true, "string": "string"}`
	badContent := `bad content`

	tests := map[string]struct {
		fName        string
		fileContents string
		missingFile  bool
		fileIsDir    bool
		url          string
		consent      bool

		rNewErr bool
		wantErr bool
	}{
		"Ok":           {fName: "0.json", fileContents: basicContent, wantErr: false},
		"Missing File": {fName: "0.json", fileContents: basicContent, missingFile: true, wantErr: true},
		"File Is Dir":  {fName: "0.json", fileIsDir: true, wantErr: true},
		"Non-numeric":  {fName: "not-numeric.json", fileContents: basicContent, rNewErr: true},
		"Bad File Ext": {fName: "0.txt", fileContents: basicContent, rNewErr: true},
		"Bad Contents": {fName: "0.json", fileContents: badContent, wantErr: true},
		"Bad URL":      {fName: "0.json", fileContents: basicContent, url: "http://bad host:1234", wantErr: true},

		"Ok Consent":           {fName: "0.json", fileContents: basicContent, consent: true, wantErr: false},
		"Missing File Consent": {fName: "0.json", fileContents: basicContent, missingFile: true, consent: true, wantErr: true},
		"File Is Dir Consent":  {fName: "0.json", fileIsDir: true, consent: true, wantErr: true},
		"Non-numeric Consent":  {fName: "not-numeric.json", fileContents: basicContent, consent: true, rNewErr: true},
		"Bad File Ext Consent": {fName: "0.txt", fileContents: basicContent, consent: true, rNewErr: true},
		"Bad Contents Consent": {fName: "0.json", fileContents: badContent, consent: true, wantErr: true},
		"Bad URL Consent":      {fName: "0.json", fileContents: basicContent, url: "http://bad host:1234", consent: true, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.False(t, tc.missingFile && tc.fileIsDir, "Test case cannot have both missing file and file is dir")

			dir := t.TempDir()

			um := &Uploader{
				collectedDir: filepath.Join(dir, constants.LocalFolder),
				uploadedDir:  filepath.Join(dir, constants.UploadedFolder),
				minAge:       0,
				timeProvider: MockTimeProvider{CurrentTime: 0},
			}

			require.NoError(t, os.Mkdir(um.collectedDir, 0750), "Setup: failed to create uploaded folder")
			require.NoError(t, os.Mkdir(um.uploadedDir, 0750), "Setup: failed to create collected folder")
			fPath := filepath.Join(um.collectedDir, tc.fName)

			if !tc.missingFile && !tc.fileIsDir {
				require.NoError(t, fileutils.AtomicWrite(fPath, []byte(tc.fileContents)), "Setup: failed to create report file")
			}
			if tc.fileIsDir {
				require.NoError(t, os.Mkdir(fPath, 0750), "Setup: failed to create directory")
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(func() { ts.Close() })

			if tc.url == "" {
				tc.url = ts.URL
			}
			r, err := report.New(fPath)
			if tc.rNewErr {
				require.Error(t, err, "Setup: failed to create report object")
				return
			}
			require.NoError(t, err, "Setup: failed to create report object")
			err = um.upload(r, tc.url, tc.consent, false)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSend(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		url            string
		noServer       bool
		serverResponse int

		wantErr bool
	}{
		"No Server":    {noServer: true, wantErr: true},
		"Bad URL":      {url: "http://local host:1234", serverResponse: http.StatusOK, wantErr: true},
		"Bad Response": {serverResponse: http.StatusForbidden, wantErr: true},

		"Success": {serverResponse: http.StatusOK},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.serverResponse)
			}))
			t.Cleanup(func() { ts.Close() })

			if tc.url == "" {
				tc.url = ts.URL
			}
			if tc.noServer {
				ts.Close()
			}
			um := &Uploader{
				responseTimeout: 0,
			}
			err := um.send(tc.url, []byte("payload"))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
