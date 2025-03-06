package insights_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

const defaultConsentFixture = "true-global"

func TestUpload(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources        []string
		config         string
		consentFixture string
		readOnlyFile   []string
		maxReports     uint
		time           int
		responseCode   int
		noServer       bool

		removeFiles []string

		wantExitCode int
	}{
		// True
		"True": {
			sources: []string{"True"},
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True DryRun": {
			sources: []string{"True"},
			config:  "dry.yaml",
		},
		"True DryRun Force": {
			sources: []string{"True"},
			config:  "dry-force.yaml",
		},
		"True DryRun Force MinAge": {
			sources: []string{"True"},
			config:  "dry-force-minAge.yaml",
			time:    2501,
		},
		"True Force": {
			sources: []string{"True"},
			config:  "force.yaml",
			removeFiles: []string{
				"True/local/2000.json",
			},
		},
		"True Force MinAge": {
			sources: []string{"True"},
			config:  "force-minAge.yaml",
			time:    2501,
			removeFiles: []string{
				"True/local/2000.json",
			},
		},
		"True MinAge": {
			sources: []string{"True"},
			config:  "minAge.yaml",
			time:    2501,
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True Global False": {
			sources:        []string{"True"},
			consentFixture: "false-global",
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True MaxReports": {
			sources:    []string{"True"},
			maxReports: 2,
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True DryRun MaxReports": {
			sources:    []string{"True"},
			config:     "dry.yaml",
			maxReports: 2,
		},
		"True MaxReports MinAge": {
			sources:    []string{"True"},
			config:     "minAge.yaml",
			maxReports: 2,
			time:       2501,
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True Duplicates": {
			sources: []string{"True"},
			removeFiles: []string{
				"True/local/2000.json",
			},
			wantExitCode: 1,
		},
		"True Bad Files": {
			sources: []string{"True"},
			removeFiles: []string{
				"True/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"True Bad Files Force": {
			sources: []string{"True"},
			removeFiles: []string{
				"True/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},

		// False
		"False": {
			sources: []string{"False"},
			removeFiles: []string{
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"False DryRun": {
			sources: []string{"False"},
			config:  "dry.yaml",
		},
		"False Force": {
			sources: []string{"False"},
			config:  "force.yaml",
			removeFiles: []string{
				"False/local/2000.json",
			},
		},
		"False MinAge": {
			sources: []string{"False"},
			config:  "minAge.yaml",
			time:    2501,
			removeFiles: []string{
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"False Duplicates": {
			sources: []string{"False"},
			removeFiles: []string{
				"False/local/2000.json",
			},
			wantExitCode: 1,
		},
		"False Bad Files": {
			sources: []string{"False"},
			removeFiles: []string{
				"False/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"False Bad Files Force": {
			sources: []string{"False"},
			removeFiles: []string{
				"False/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},

		// Unknown
		"Unknown": {
			sources: []string{"Unknown-A"},
			removeFiles: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown DryRun": {
			sources: []string{"Unknown-A"},
			config:  "dry.yaml",
		},
		"Unknown Force": {
			sources: []string{"Unknown-A"},
			config:  "force.yaml",
			removeFiles: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},

		// Unknown Global False
		"Unknown Global False": {
			sources:        []string{"Unknown-A"},
			consentFixture: "false-global",
			removeFiles: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown DryRun Global False": {
			sources:        []string{"Unknown-A"},
			config:         "dry.yaml",
			consentFixture: "false-global",
		},
		"Unknown Force Global False": {
			sources:        []string{"Unknown-A"},
			config:         "force.yaml",
			consentFixture: "false-global",
			removeFiles: []string{
				"Unknown-A/local/2000.json",
			},
		},

		// Multi Sources
		"Multi": {
			sources: []string{"True", "False"},
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"Multi DryRun": {
			sources: []string{"True", "False"},
			config:  "dry.yaml",
		},
		"Multi Force": {
			sources: []string{"True", "False"},
			config:  "force.yaml",
			removeFiles: []string{
				"True/local/2000.json",
				"False/local/2000.json",
			},
		},

		// All
		"All": {
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"All DryRun": {
			config: "dry.yaml",
		},
		"All Force": {
			config: "force.yaml",
			removeFiles: []string{
				"True/local/2000.json",
				"False/local/2000.json",
				"Unknown-A/local/2000.json",
			},
		},
		"All MinAge": {
			config: "minAge.yaml",
			time:   2501,
			removeFiles: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"All Duplicates": {
			removeFiles: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"All Bad Files": {
			removeFiles: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"All Bad Files Force": {
			removeFiles: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},

		// No Server
		"True No Server": {
			sources:      []string{"True"},
			noServer:     true,
			wantExitCode: 1,
		},
		"All No Server": {
			noServer:     true,
			wantExitCode: 1,
		},

		// Bad Response
		"True Bad Response": {
			sources:      []string{"True"},
			responseCode: http.StatusInternalServerError,
			wantExitCode: 1,
		},
		"All Bad Response": {
			responseCode: http.StatusInternalServerError,
			wantExitCode: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.responseCode == 0 {
				tc.responseCode = http.StatusOK
			}

			var mu sync.Mutex
			gotPayloads := make([]string, 0)
			server := newTestServer(t, echoPayloadHandler(tc.responseCode, &mu, &gotPayloads)).URL
			if tc.noServer {
				server = ""
			}

			if tc.consentFixture == "" {
				tc.consentFixture = defaultConsentFixture
			}

			paths := copyFixtures(t, tc.consentFixture)

			// Remove files
			for _, f := range tc.removeFiles {
				require.NoError(t, os.Remove(filepath.Join(paths.reports, f)), "Setup: failed to remove file")
			}

			for _, f := range tc.readOnlyFile {
				testutils.MakeReadOnly(t, filepath.Join(paths.reports, f))
			}

			consentContents, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "Setup: failed to get consent directory contents")

			smContents, err := testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "Setup: failed to get source metrics directory contents")

			// #nosec:G204 - we control the command arguments in tests
			cmd := exec.Command(cliPath, "upload")
			cmd.Args = append(cmd.Args, tc.sources...)
			if tc.config != "" {
				tc.config = filepath.Join("testdata", "configs", "upload", tc.config)
				cmd.Args = append(cmd.Args, "--config", tc.config)
			}
			cmd.Args = append(cmd.Args, "-vv")
			cmd.Args = append(cmd.Args, "--consent-dir", paths.consent)
			cmd.Args = append(cmd.Args, "--insights-dir", paths.reports)
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_SERVER_URL="+server)
			if tc.maxReports != 0 {
				cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_REPORTS="+fmt.Sprint(tc.maxReports))
			}
			if tc.time != 0 {
				cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_TIME="+fmt.Sprint(tc.time))
			}
			out, err := cmd.CombinedOutput()
			if tc.wantExitCode == 0 {
				require.NoError(t, err, "unexpected CLI error: %v\n%s", err, out)
			}
			assert.Equal(t, tc.wantExitCode, cmd.ProcessState.ExitCode(), "unexpected exit code: %v\n%s", err, out)

			// Check that the consent directory was not modified
			gotContents, err := testutils.GetDirContents(t, paths.consent, 3)
			require.NoError(t, err, "failed to get consent directory contents")
			assert.Equal(t, consentContents, gotContents)

			// Check that the source metrics directory was not modified
			gotContents, err = testutils.GetDirContents(t, paths.sourceMetrics, 3)
			require.NoError(t, err, "failed to get source metrics directory contents")
			assert.Equal(t, smContents, gotContents)

			type results struct {
				Payloads        []string
				ReportsContents map[string]string
			}

			var got results
			got.ReportsContents, err = testutils.GetDirContents(t, paths.reports, 3)
			require.NoError(t, err, "failed to get reports directory contents")
			got.Payloads = gotPayloads

			// Remove return carriage from payloads
			for i, payload := range got.Payloads {
				got.Payloads[i] = strings.ReplaceAll(payload, "\r", "")
			}

			sort.Strings(got.Payloads) // Sort to ignore Payload arrival order
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got)
		})
	}
}

func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// echoPayloadHandler is a handler that echoes the request body to a channel.
func echoPayloadHandler(responseCode int, mu *sync.Mutex, payloads *[]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send request body to channel
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}
		mu.Lock()
		*payloads = append(*payloads, string(body))
		mu.Unlock()

		// Ensure is JSON
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			http.Error(w, "expected Content-Type application/json", http.StatusBadRequest)
			return
		}

		if !json.Valid(body) {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		w.WriteHeader(responseCode)
	})
}
