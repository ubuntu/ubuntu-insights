// Package libinsights provides helpers shared library integration tests.
package libinsights

/*
#include <stdlib.h>
#ifdef SYSTEM_LIB
#include <insights/insights.h>
#include <insights/types.h>
#else
#include "insights.h"
#include "types.h"
#endif

extern void log_callback_c_wrapper(insights_log_level level, const char *msg);
*/
import "C"

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testsdetection"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

// Indicate if using system lib or local built lib.
var systemLib = false

func init() {
	testsdetection.MustBeTesting()
}

type testEnv struct {
	config      C.insights_config
	consentDir  string
	insightsDir string
	logPath     string
	cSource     *C.char
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	tempDir := t.TempDir()
	consentDir := filepath.Join(tempDir, "consent")
	insightsDir := filepath.Join(tempDir, "insights")
	logPath := filepath.Join(tempDir, "test.log")
	source := "integration-test-source"

	err := os.MkdirAll(consentDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(insightsDir, 0755)
	require.NoError(t, err)

	cConsentDir := C.CString(consentDir)
	cInsightsDir := C.CString(insightsDir)
	cSource := C.CString(source)

	config := C.insights_config{
		consent_dir:  cConsentDir,
		insights_dir: cInsightsDir,
	}

	env := &testEnv{
		config:      config,
		consentDir:  consentDir,
		insightsDir: insightsDir,
		logPath:     logPath,
		cSource:     cSource,
	}

	t.Cleanup(func() {
		C.free(unsafe.Pointer(cConsentDir))
		C.free(unsafe.Pointer(cInsightsDir))
		C.free(unsafe.Pointer(cSource))
	})

	return env
}

// TestSetConsent runs tests for setting consent state.
func TestSetConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		state           bool
		useSystemSource bool
	}{
		"Enable":        {state: true},
		"Disable":       {state: false},
		"System Enable": {state: true, useSystemSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := setupTestEnv(t)

			if tc.useSystemSource {
				newCSource := C.CString(constants.DefaultCollectSource)
				env.cSource = newCSource
				t.Cleanup(func() { C.free(unsafe.Pointer(newCSource)) })
			}

			// Ensure that we only modify the specified source's consent state.
			staticSource := C.CString("static-test-source")
			defer C.free(unsafe.Pointer(staticSource))
			cErr := C.insights_set_consent_state(&env.config, staticSource, C.bool(false))
			require.NoError(t, charToError(cErr), "Setup: failed to set consent for static source")

			cErr = C.insights_set_consent_state(&env.config, env.cSource, C.bool(tc.state))
			require.NoError(t, charToError(cErr), "Failed to set consent")

			states := validateConsent(t, env.consentDir)
			targetSource := "integration-test-source"
			if tc.useSystemSource {
				targetSource = "SYSTEM"
			}
			assert.Equal(t, tc.state, states[targetSource], "Consent state mismatch for %s", targetSource)
			assert.False(t, states["static-test-source"], "Consent state mismatch for static-test-source")
		})
	}
}

// TestGetConsent runs tests for getting consent state.
func TestGetConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		isNotSet        bool
		initialState    bool
		expectState     C.insights_consent_state
		useSystemSource bool
	}{
		"InitialEnable":        {initialState: true, expectState: C.INSIGHTS_CONSENT_TRUE},
		"InitialDisable":       {initialState: false, expectState: C.INSIGHTS_CONSENT_FALSE},
		"NotSet":               {isNotSet: true, expectState: C.INSIGHTS_CONSENT_UNKNOWN},
		"System InitialEnable": {initialState: true, expectState: C.INSIGHTS_CONSENT_TRUE, useSystemSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := setupTestEnv(t)

			if tc.useSystemSource {
				newCSource := C.CString(constants.DefaultCollectSource)
				env.cSource = newCSource
				t.Cleanup(func() { C.free(unsafe.Pointer(newCSource)) })
			}

			if !tc.isNotSet {
				cErr := C.insights_set_consent_state(&env.config, env.cSource, C.bool(tc.initialState))
				require.NoError(t, charToError(cErr), "Setup: failed to set initial consent state")
			}

			// Calling the function
			state := C.insights_get_consent_state(&env.config, env.cSource)
			assert.Equal(t, tc.expectState, state, "Consent state mismatch")
		})
	}
}

// TestCollect runs tests for collecting insights.
func TestCollect(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState    *bool // nil = unknown/default
		useSystemSource bool
		flags           func() C.insights_collect_flags
	}{
		"DryRun with Consent": {
			consentState: boolPtr(true),
			flags: func() C.insights_collect_flags {
				var f C.insights_collect_flags
				f.dry_run = C.bool(true)
				return f
			},
		},
		"Collect with Consent": {
			consentState: boolPtr(true),
			flags: func() C.insights_collect_flags {
				var f C.insights_collect_flags
				f.force = C.bool(true)
				return f
			},
		},
		"Collect OptOut": {
			consentState: boolPtr(false),
			flags: func() C.insights_collect_flags {
				var f C.insights_collect_flags
				f.force = C.bool(true)
				return f
			},
		},
		"System Collect with Consent": {
			consentState:    boolPtr(true),
			useSystemSource: true,
			flags: func() C.insights_collect_flags {
				var f C.insights_collect_flags
				f.force = C.bool(true)
				return f
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := setupTestEnv(t)

			if tc.useSystemSource {
				newCSource := C.CString(constants.DefaultCollectSource)
				env.cSource = newCSource
				t.Cleanup(func() { C.free(unsafe.Pointer(newCSource)) })
			}

			if tc.consentState != nil {
				cErr := C.insights_set_consent_state(&env.config, env.cSource, C.bool(*tc.consentState))
				require.NoError(t, charToError(cErr), "Setup: Failed to set consent")
			}

			flags := tc.flags()
			var report *C.char
			defer func() {
				if report != nil {
					C.free(unsafe.Pointer(report))
				}
			}()
			cErr := C.insights_collect(&env.config, env.cSource, &flags, &report)
			require.NoError(t, charToError(cErr), "Failed to collect")

			// Validation
			got := validateReports(t, env.insightsDir)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Collected reports do not match expected")
		})
	}
}

// TestCompileAndWrite runs tests for compiling and writing insights.
func TestCompileAndWrite(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState bool
	}{
		"With Consent":    {consentState: true},
		"Without Consent": {consentState: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := setupTestEnv(t)

			cErr := C.insights_set_consent_state(&env.config, env.cSource, C.bool(tc.consentState))
			require.NoError(t, charToError(cErr), "Setup: Failed to set consent")

			var compileFlags C.insights_compile_flags
			var report *C.char
			defer func() {
				if report != nil {
					C.free(unsafe.Pointer(report))
				}
			}()
			cErr = C.insights_compile(&env.config, &compileFlags, &report)
			require.NoError(t, charToError(cErr), "Failed to compile")

			var writeFlags C.insights_write_flags
			writeFlags.force = C.bool(true)

			cErr = C.insights_write(&env.config, env.cSource, report, &writeFlags)
			require.NoError(t, charToError(cErr), "Failed to write report")

			// Validation
			got := validateReports(t, env.insightsDir)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "Compiled reports do not match expected")
		})
	}
}

// TestUpload runs tests for uploading insights.
func TestUpload(t *testing.T) {
	tests := map[string]struct {
		consentState bool
		dryRun       bool
		expectError  bool
	}{
		"DryRun Upload": {
			dryRun:      true,
			expectError: false,
		},
		"Regular Upload": {
			dryRun:      false,
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if !tc.dryRun && systemLib {
				t.Skip("Skipping regular upload test with system lib")
			}

			testServer, testServerState := setupTestServer(t)
			defer testServer.Close()

			if !systemLib {
				setTestServerURL(testServer.URL)
			}

			env := setupTestEnv(t)

			// We need a report to upload.
			cErr := C.insights_set_consent_state(&env.config, env.cSource, C.bool(tc.consentState))
			require.NoError(t, charToError(cErr), "Setup failure")

			var cFlags C.insights_collect_flags
			cFlags.force = C.bool(true)
			cErr = C.insights_collect(&env.config, env.cSource, &cFlags, nil)
			require.NoError(t, charToError(cErr), "Setup failure")

			// Upload
			var uFlags C.insights_upload_flags
			uFlags.force = C.bool(true)
			uFlags.dry_run = C.bool(tc.dryRun)
			uFlags.min_age = 0

			sources := []*C.char{env.cSource}
			cErr = C.insights_upload(&env.config, &sources[0], C.size_t(1), &uFlags)

			if tc.expectError {
				require.Error(t, charToError(cErr), "Expected upload to fail")
				return
			}
			require.NoError(t, charToError(cErr), "Unexpected upload failed")

			if !tc.dryRun {
				testServerState.mu.Lock()
				got := testServerState.Received
				testServerState.mu.Unlock()
				want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
				assert.Equal(t, want, got, "Received reports do not match expected")
			}
		})
	}
}

// Global logger file pointer setup for callback tests.
var gLogFile *os.File
var gLogMu sync.Mutex

func writeLog(format string, args ...any) {
	gLogMu.Lock()
	defer gLogMu.Unlock()
	if gLogFile != nil {
		fmt.Fprintf(gLogFile, format, args...)
	}
}

// Exported function for C to call back into Go
//
//export goLogCallback
func goLogCallback(level C.insights_log_level, msg *C.char) {
	goLogMsg := C.GoString(msg)
	writeLog("[LIBINSIGHTS][%d] %s\n", int(level), goLogMsg)
}

// TestCallback runs tests for log callback behavior.
func TestCallback(t *testing.T) {
	tests := map[string]struct {
		useCallback bool
	}{
		"WithCallback":    {useCallback: true},
		"WithoutCallback": {useCallback: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Run the test in a subprocess to verify that no output is printed to stdout/stderr
			// when the callback is set.
			arg := "false"
			if tc.useCallback {
				arg = "true"
			}
			args := testutils.SetupFakeCmdArgs("TestHelperCallbackWorkWrapper", arg)
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Env = append(os.Environ(), "GO_HELPER_PROCESS=1")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "Helper process failed: %s", out)

			output := string(out)
			if tc.useCallback {
				// We expect the C library output to be suppressed, so stdout should be relatively clean (just test runner output)
				assert.NotContains(t, output, "[LIBINSIGHTS]", "Found Go callback logs in stdout/stderr, expected them to be suppressed")
				assert.NotContains(t, output, "insights", "Found insights logs in stdout/stderr, expected them to be suppressed")
				return
			}
			// Without callback, we expect logs to go to stdout/stderr
			assert.Contains(t, strings.ToLower(output), "insights", "Expected insights logs in stdout")
		})
	}
}

// TestHelperCallbackWork is a helper function to be run in a subprocess to test logging callback behavior.
func TestHelperCallbackWork(t *testing.T) {
	args, err := testutils.GetFakeCmdArgs()
	if err != nil {
		return
	}
	require.Len(t, args, 1)
	useCallback := args[0] == "true"

	env := setupTestEnv(t)
	env.config.verbose = C.bool(true)

	// Setup Log File locally for this test
	f, err := os.Create(env.logPath)
	require.NoError(t, err)
	gLogMu.Lock()
	gLogFile = f
	gLogMu.Unlock()
	t.Cleanup(func() {
		gLogMu.Lock()
		if gLogFile != nil {
			gLogFile.Close()
			gLogFile = nil
		}
		gLogMu.Unlock()
	})

	if useCallback {
		C.insights_set_log_callback(C.insights_logger_callback(C.log_callback_c_wrapper))
		t.Cleanup(func() {
			C.insights_set_log_callback(nil)
		})
	}

	// Trigger logs using collect with dry run (should log stuff but not write report)
	var cFlags C.insights_collect_flags
	// Ensure we generate some logs
	cFlags.dry_run = C.bool(true)
	cFlags.force = C.bool(true)

	// We just want to trigger logging, result doesn't matter much
	C.insights_collect(&env.config, env.cSource, &cFlags, nil)

	// Validate logs in file
	counts := countLogLevels(t, env.logPath)

	totalLogs := 0
	for _, count := range counts {
		totalLogs += count
	}

	if useCallback {
		assert.Positive(t, totalLogs, "Expected callback to handle some logs")
		// We expect at least Info (2) or Debug (3) logs usually.
		t.Logf("Log counts by level: %v", counts)
	} else {
		assert.Equal(t, 0, totalLogs, "Expected no logs in file when callback is unset")
	}
}

func countLogLevels(t *testing.T, logPath string) map[int]int {
	t.Helper()
	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close()

	counts := make(map[int]int)
	re := regexp.MustCompile(`\[LIBINSIGHTS\]\[(\d+)\]`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			var level int
			_, err := fmt.Sscanf(matches[1], "%d", &level)
			require.NoError(t, err, "Failed to parse log level")
			counts[level]++
		}
	}
	return counts
}

func boolPtr(b bool) *bool {
	return &b
}

// ReportCounts holds counts of different report types.
type ReportCounts struct {
	OptOut  int
	Regular int
}

func validateConsent(t *testing.T, consentDir string) map[string]bool {
	t.Helper()
	consentStates := make(map[string]bool)

	cEntries, err := os.ReadDir(consentDir)
	require.NoError(t, err, "Failed to read consent directory")
	for _, entry := range cEntries {
		assert.False(t, entry.IsDir(), "Consent entry %s is a directory, expected file", entry.Name())
		name := entry.Name()
		var source string
		if name == "consent.toml" {
			source = "DEFAULT"
		} else if before, ok := strings.CutSuffix(name, constants.ConsentSourceBaseSeparator+constants.DefaultConsentFileName); ok {
			source = before
			if before == constants.DefaultCollectSource {
				source = "SYSTEM"
			}
		}
		require.NotEmpty(t, source, "Failed to infer source from consent file name %s", name)

		var c struct {
			ConsentState bool `toml:"consent_state"`
		}
		_, err := toml.DecodeFile(filepath.Join(consentDir, name), &c)
		if assert.NoError(t, err, "Failed to decode consent file %s", name) {
			consentStates[source] = c.ConsentState
		}
	}

	return consentStates
}

func validateReports(t *testing.T, insightsDir string) map[string]ReportCounts {
	t.Helper()
	reports := make(map[string]ReportCounts)

	err := filepath.WalkDir(insightsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		rel, err := filepath.Rel(insightsDir, path)
		require.NoError(t, err)

		// Should be of the form <source>/subdir/report.json
		parts := strings.Split(rel, string(os.PathSeparator))
		require.Len(t, parts, 3, "Unexpected report path structure: %s", rel)
		source := parts[0]

		if source == constants.DefaultCollectSource {
			source = "SYSTEM"
		}

		content, err := os.ReadFile(path)
		require.NoError(t, err)

		var payload struct {
			OptOut bool
		}
		err = json.Unmarshal(content, &payload)
		assert.NoError(t, err, "Invalid JSON in %s", path)

		c := reports[source]
		if payload.OptOut {
			c.OptOut++
		} else {
			c.Regular++
		}
		reports[source] = c
		return nil
	})
	require.NoError(t, err, "Failed to walk insights dir")

	return reports
}

type testServerState struct {
	mu            sync.Mutex
	Received      map[string]ReportCounts
	TotalRequests int
}

func setupTestServer(t *testing.T) (*httptest.Server, *testServerState) {
	t.Helper()
	state := &testServerState{
		Received: make(map[string]ReportCounts),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		state.mu.Lock()
		defer state.mu.Unlock()
		state.TotalRequests++

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Logf("TestServer: Failed to read body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var optOutStruct struct {
			OptOut bool `json:"OptOut"`
		}

		if err := json.Unmarshal(body, &optOutStruct); err != nil {
			t.Logf("TestServer: Failed to unmarshal body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		source := path.Base(r.URL.Path)
		counts := state.Received[source]
		if optOutStruct.OptOut {
			counts.OptOut++
		} else {
			counts.Regular++
		}
		state.Received[source] = counts

		w.WriteHeader(http.StatusAccepted)
	}))

	return server, state
}

// charToError converts a C string error to a Go error and frees the C string.
func charToError(cErr *C.char) error {
	if cErr == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cErr))
	return fmt.Errorf("%s", C.GoString(cErr))
}
