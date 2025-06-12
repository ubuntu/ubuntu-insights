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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

const defaultConsentFixture = "true-global"

func TestUpload(t *testing.T) {
	t.Parallel()

	const (
		baseRetryPeriod = 100 * time.Millisecond
		maxRetryPeriod  = 4 * time.Second
		maxAttempts     = 4
		responseTimeout = 2 * time.Second
	)

	tests := map[string]struct {
		sources        []string
		config         string
		consentFixture string
		readOnlyFile   []string
		maxReports     uint
		time           int

		// Server options
		badCount            int
		initialResponseCode int // If < 0, the server will not respond
		responseCode        int
		noServer            bool

		removeReports       []string
		removeGlobalConsent bool

		wantExitCode int
	}{
		// True
		"True-Uploads reports from the specified source": {
			sources: []string{"True"},
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-Causes nothing to happen with DryRun": {
			sources: []string{"True"},
			config:  "dry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-DryRun causes nothing to happen with Force": {
			sources: []string{"True"},
			config:  "dry-force.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-DryRun causes nothing to happen with Force MinAge": {
			sources: []string{"True"},
			config:  "dry-force-minAge.yaml",
			time:    2501,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-Force uploads duplicate files": {
			sources: []string{"True"},
			config:  "force.yaml",
			removeReports: []string{
				"True/local/2000.json",
			},
		},
		"True-Force overrides MinAge": {
			sources: []string{"True"},
			config:  "force-minAge.yaml",
			time:    2501,
			removeReports: []string{
				"True/local/2000.json",
			},
		},
		"True-MinAge skips immature reports": {
			sources: []string{"True"},
			config:  "minAge.yaml",
			time:    2501,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-Prioritizes local consent over Global False": {
			sources:        []string{"True"},
			consentFixture: "false-global",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-Removes old reports when over MaxReports limit": {
			sources:    []string{"True"},
			maxReports: 2,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-DryRun does not trigger cleanup with MaxReports": {
			sources:    []string{"True"},
			config:     "dry.yaml",
			maxReports: 2,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-MaxReports and MinAge are respected": {
			sources:    []string{"True"},
			config:     "minAge.yaml",
			maxReports: 2,
			time:       2501,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
			},
		},
		"True-Errors when encountering duplicates": {
			sources: []string{"True"},
			removeReports: []string{
				"True/local/2000.json",
			},
			wantExitCode: 1,
		},
		"True-Errors when encountering bad files": {
			sources: []string{"True"},
			removeReports: []string{
				"True/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"True-Force errors when encountering bad files": {
			sources: []string{"True"},
			removeReports: []string{
				"True/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},
		"True-DryRun errors when encountering bad files": {
			sources: []string{"True"},
			removeReports: []string{
				"True/uploaded/1000.json",
			},
			config:       "dry.yaml",
			wantExitCode: 1,
		},
		"True-DryRun errors when encountering duplicate files": {
			sources: []string{"True"},
			removeReports: []string{
				"True/local/2000.json",
			},
			config:       "dry.yaml",
			wantExitCode: 1,
		},

		// False
		"False-Respects consent": {
			sources: []string{"False"},
			removeReports: []string{
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"False-DryRun causes nothing to happen": {
			sources: []string{"False"},
			config:  "dry.yaml",
			removeReports: []string{
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"False-Force respects consent and uploads duplicate files": {
			sources: []string{"False"},
			config:  "force.yaml",
			removeReports: []string{
				"False/local/2000.json",
			},
		},
		"False-MinAge skips immature reports": {
			sources: []string{"False"},
			config:  "minAge.yaml",
			time:    2501,
			removeReports: []string{
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"False Duplicates": {
			sources: []string{"False"},
			removeReports: []string{
				"False/local/2000.json",
			},
			wantExitCode: 1,
		},
		"False Bad Files": {
			sources: []string{"False"},
			removeReports: []string{
				"False/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"False Bad Files Force": {
			sources: []string{"False"},
			removeReports: []string{
				"False/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},

		// Unknown
		"Unknown-Consent falls back to global when not set": {
			sources: []string{"Unknown-A"},
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown-DryRun causes nothing to happen": {
			sources: []string{"Unknown-A"},
			config:  "dry.yaml",
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown-Force uploads duplicate files": {
			sources: []string{"Unknown-A"},
			config:  "force.yaml",
			removeReports: []string{
				"Unknown-A/local/2000.json",
			},
		},

		// Unknown Global False
		"Unknown-Consent falls back to global when not set and respects no consent": {
			sources:        []string{"Unknown-A"},
			consentFixture: "false-global",
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown-DryRun causes nothing to happen with false consent": {
			sources:        []string{"Unknown-A"},
			config:         "dry.yaml",
			consentFixture: "false-global",
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"Unknown-Force respects consent and uploads duplicate files": {
			sources:        []string{"Unknown-A"},
			config:         "force.yaml",
			consentFixture: "false-global",
			removeReports: []string{
				"Unknown-A/local/2000.json",
			},
		},

		// Unknown Global Not Set
		"Unknown-Exits 0 when no consent set and does nothing": {
			sources: []string{"Unknown-A"},
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
			removeGlobalConsent: true,
		},

		// Multi Sources
		"Multi-Uploads all reports from all specified sources": {
			sources: []string{"True", "False"},
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"Multi-DryRun causes nothing to happen": {
			sources: []string{"True", "False"},
			config:  "dry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
		},
		"Multi-Force uploads duplicate files": {
			sources: []string{"True", "False"},
			config:  "force.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"False/local/2000.json",
			},
		},

		// All
		"All-Uploads all reports for all sources detected": {
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"All-DryRun causes nothing to happen": {
			config: "dry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"All-Force uploads duplicate files": {
			config: "force.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"False/local/2000.json",
				"Unknown-A/local/2000.json",
			},
		},
		"All-Immature reports are skipped": {
			config: "minAge.yaml",
			time:   2501,
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
			},
		},
		"All-Errors when encountering duplicate files": {
			removeReports: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"All-Errors when encountering bad files": {
			removeReports: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			wantExitCode: 1,
		},
		"All-Force errors when encountering bad files": {
			removeReports: []string{
				"True/uploaded/1000.json",
				"False/uploaded/1000.json",
				"Unknown-A/uploaded/1000.json",
			},
			config:       "force.yaml",
			wantExitCode: 1,
		},

		// No Server Response
		"True-No response from server errors": {
			sources:      []string{"True"},
			noServer:     true,
			wantExitCode: 1,
		},
		"All-No response from server errors": {
			noServer:     true,
			wantExitCode: 1,
		},

		// Bad Response
		"True-Bad response from server errors": {
			sources:      []string{"True"},
			responseCode: http.StatusInternalServerError,
			wantExitCode: 1,
		},
		"All-Bad response from server errors": {
			responseCode: http.StatusInternalServerError,
			wantExitCode: 1,
		},

		// Exponential Backoff (Retry) Tests
		"Exponential backoff exits 0 when no consent set and does nothing": {
			sources: []string{"Unknown-A", "False"},
			config:  "retry.yaml",
			removeReports: []string{
				"Unknown-A/local/2000.json",
				"Unknown-A/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
			removeGlobalConsent: true,

			initialResponseCode: http.StatusInternalServerError,
			badCount:            2,
			wantExitCode:        0,
		},
		"Exponential backoff retries when bad response code": {
			sources: []string{"True", "False"},
			config:  "retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},

			initialResponseCode: http.StatusInternalServerError,
			badCount:            3,
			wantExitCode:        0,
		},
		"Exponential backoff retries when no response": {
			sources: []string{"True", "False"},
			config:  "retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},

			initialResponseCode: -1,
			badCount:            3,
			wantExitCode:        0,
		},
		"Exponential backoff does nothing when dry-run": {
			sources: []string{"True", "False"},
			config:  "dry-retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},

			initialResponseCode: http.StatusInternalServerError,
			badCount:            500,
			wantExitCode:        0,
		},
		"Exponential backoff overwrites duplicate reports with force": {
			sources: []string{"True", "False"},
			config:  "force-retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"False/local/2000.json",
			},

			initialResponseCode: http.StatusInternalServerError,
			badCount:            3,
			wantExitCode:        0,
		},
		// Exponential backoff (Retry) erroring tests
		"Exponential backoff gives up after too many bad response codes": {
			sources: []string{"True", "False"},
			config:  "retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
			initialResponseCode: http.StatusInternalServerError,
			badCount:            500,
			wantExitCode:        1,
		},
		"Exponential backoff gives up after too many response timeouts": {
			sources: []string{"True", "False"},
			config:  "retry.yaml",
			removeReports: []string{
				"True/local/2000.json",
				"True/uploaded/1000.json",
				"False/local/2000.json",
				"False/uploaded/1000.json",
			},
			initialResponseCode: -1,
			badCount:            500,
			wantExitCode:        1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.responseCode == 0 {
				tc.responseCode = http.StatusAccepted
			}

			var mu sync.Mutex
			gotPayloads := make([]string, 0)
			s := httptest.NewUnstartedServer(echoPayloadHandler(tc.responseCode, &mu, &gotPayloads))
			if tc.initialResponseCode != 0 {
				s = httptest.NewUnstartedServer(echoBackoffPayloadHandler(tc.initialResponseCode, tc.responseCode, &tc.badCount, responseTimeout*2, &mu, &gotPayloads))
			}
			if !tc.noServer {
				t.Cleanup(s.Close)
				s.Start()
			}
			server := s.URL

			if tc.consentFixture == "" {
				tc.consentFixture = defaultConsentFixture
			}

			paths := copyFixtures(t, tc.consentFixture)

			// Remove files
			for _, f := range tc.removeReports {
				require.NoError(t, os.Remove(filepath.Join(paths.reports, f)), "Setup: failed to remove file")
			}
			if tc.removeGlobalConsent {
				require.NoError(t, os.Remove(filepath.Join(paths.consent, "consent.toml")), "Setup: failed to remove global consent file")
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
			cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_BASE_RETRY_PERIOD="+baseRetryPeriod.String())
			cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_RETRY_PERIOD="+maxRetryPeriod.String())
			cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_ATTEMPTS="+fmt.Sprint(maxAttempts))
			cmd.Env = append(cmd.Env, "UBUNTU_INSIGHTS_INTEGRATIONTESTS_RESPONSE_TIMEOUT="+responseTimeout.String())
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

// echoBackoffPayloadHandler is like echoPayloadHandler, but for the first badCount requests, it returns an initial bad response code.
func echoBackoffPayloadHandler(initialResponseCode, responseCode int, badCount *int, sleepTime time.Duration, mu *sync.Mutex, payloads *[]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if *badCount > 0 {
			*badCount--
			mu.Unlock()
			// Unresponsive server if initialResponseCode < 0
			if initialResponseCode < 0 {
				time.Sleep(sleepTime)
				return
			}
			w.WriteHeader(initialResponseCode)
			return
		}
		defer mu.Unlock()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}

		*payloads = append(*payloads, string(body))
		w.WriteHeader(responseCode)
	})
}
