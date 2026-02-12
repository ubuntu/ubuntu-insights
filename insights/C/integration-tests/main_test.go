package libinsights_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	constantstestutils "github.com/ubuntu/ubuntu-insights/insights/internal/constants/testutils"
)

const (
	reportStartMarker = "REPORT_START"
	reportEndMarker   = "REPORT_END"
)

var (
	testDriverPath string
	systemLib      bool
)

func TestMain(m *testing.M) {
	constantstestutils.Normalize()

	// Get directories
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get cwd: %v", err)
	}

	// Assuming cwd is insights/C/integration-tests
	// We want insights/C root
	cDir := filepath.Dir(cwd)
	driverDir := filepath.Join(cwd, "test-driver")
	generatedDir := filepath.Join(cwd, "generated")

	buildDir, err := os.MkdirTemp("", "insights-test-driver-*")
	if err != nil {
		log.Fatalf("Failed to create temporary build directory: %v", err)
	}
	defer os.RemoveAll(buildDir)

	// 1. Generate shared library if not using system lib
	if !systemLib {
		log.Println("Generating shared libraries via go generate...")
		cmd := exec.Command("go", "generate", ".")
		cmd.Dir = cDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to generate shared libraries: %v", err)
		}
	}

	// 2. Build the test driver
	log.Println("Building test driver...")

	exeName := "test-driver"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	testDriverPath = filepath.Join(buildDir, exeName)

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}

	cflags := []string{"-Wall", "-Wextra"}
	ldflags := []string{}

	if systemLib {
		cflags = append(cflags, "-DSYSTEM_LIB")
		ldflags = append(ldflags, "-linsights")
	} else {
		cflags = append(cflags, fmt.Sprintf("-I%s", generatedDir))

		// Link directly against the versioned shared library file
		var libName string
		switch runtime.GOOS {
		case "linux":
			libName = "libinsights.so.0"
		case "darwin":
			libName = "libinsights.0.dylib"
		case "windows":
			libName = "libinsights.dll"
		default:
			log.Fatalf("Unsupported OS: %s", runtime.GOOS)
		}

		ldflags = append(ldflags, filepath.Join(generatedDir, libName))

		if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
			ldflags = append(ldflags, fmt.Sprintf("-Wl,-rpath,%s", generatedDir))
		}
	}

	args := append(cflags, "-o", testDriverPath, "main.c")
	args = append(args, ldflags...)

	// #nosec:G204 - we control the command arguments in tests
	buildCmd := exec.Command(cc, args...)
	buildCmd.Dir = driverDir
	buildCmd.Env = os.Environ()

	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build test driver: %v", err)
	}

	// On Windows, if using local lib, we copy the .dll next to the executable
	if !systemLib && runtime.GOOS == "windows" {
		dllSrc := filepath.Join(generatedDir, "libinsights.dll")
		dllDst := filepath.Join(buildDir, "libinsights.dll")
		input, err := os.ReadFile(dllSrc)
		if err == nil {
			if err := os.WriteFile(dllDst, input, 0600); err != nil {
				log.Printf("Warning: failed to write dll to %s: %v", dllDst, err)
			}
		} else {
			log.Printf("Warning: could not read dll from %s: %v", dllSrc, err)
		}
	}

	m.Run()
}

type testFixture struct {
	consentDir  string
	insightsDir string
	logPath     string
	source      string
	uploadURL   string
	makePanic   bool
	printReport bool
}

func setupTestFixture(t *testing.T) *testFixture {
	t.Helper()
	tempDir := t.TempDir()
	consentDir := filepath.Join(tempDir, "consent")
	insightsDir := filepath.Join(tempDir, "insights")
	logPath := filepath.Join(tempDir, "test.log")
	source := "integration-test-source"

	err := os.MkdirAll(consentDir, 0750)
	require.NoError(t, err)
	err = os.MkdirAll(insightsDir, 0750)
	require.NoError(t, err)

	return &testFixture{
		consentDir:  consentDir,
		insightsDir: insightsDir,
		logPath:     logPath,
		source:      source,
	}
}

func extractReport(output string) string {
	s := strings.Index(output, reportStartMarker)
	if s == -1 {
		return output // fallback
	}
	s += len(reportStartMarker)

	e := strings.Index(output[s:], reportEndMarker)
	if e == -1 {
		return strings.TrimSpace(output[s:])
	}
	return strings.TrimSpace(output[s : s+e])
}

func runDriver(t *testing.T, fixture *testFixture, args ...string) (string, error) {
	t.Helper()
	baseArgs := []string{
		"--consent-dir", fixture.consentDir,
		"--insights-dir", fixture.insightsDir,
	}
	allArgs := append(baseArgs, args...)

	if fixture.printReport {
		// pass --print-report if it's a collect or compile command
		// We scan args to see if "collect" or "compile" is the command
		for _, arg := range args {
			if arg == "collect" || arg == "compile" {
				allArgs = append(allArgs, "--print-report")
				break
			}
		}
	}

	// #nosec:G204 - we control the command arguments in tests
	cmd := exec.Command(testDriverPath, allArgs...)

	// Pass the upload URL if needed
	cmd.Env = os.Environ()
	if fixture.uploadURL != "" {
		cmd.Env = append(cmd.Env, "INSIGHTS_TEST_UPLOAD_URL="+fixture.uploadURL)
	}
	if fixture.makePanic {
		cmd.Env = append(cmd.Env, "INSIGHTS_TEST_MAKE_PANIC=true")
	}

	out, err := cmd.CombinedOutput()
	output := string(out)
	t.Logf("Command output: %s", output)
	if err != nil {
		return strings.TrimSpace(output), err
	}

	output = strings.TrimSpace(output)
	// If command is collect or compile, output might contain report wrapped
	if fixture.printReport && strings.Contains(output, reportStartMarker) {
		return extractReport(output), nil
	}
	return output, nil
}

func boolPtr(b bool) *bool {
	return &b
}

type ReportCounts struct {
	OptOut  int
	Regular int
}

func validateConsent(t *testing.T, consentDir string) map[string]bool {
	t.Helper()
	consentStates := make(map[string]bool)

	cEntries, err := os.ReadDir(consentDir)
	if os.IsNotExist(err) {
		return consentStates
	}
	require.NoError(t, err, "Failed to read consent directory")
	for _, entry := range cEntries {
		require.False(t, entry.IsDir(), "Consent entry %s is a directory, expected file", entry.Name())
		name := entry.Name()
		source, found := strings.CutSuffix(name, constants.ConsentFilenameSuffix)
		require.True(t, found, "Consent file name %s does not match expected pattern", name)
		require.NotEmpty(t, source, "Failed to infer source from consent file name %s", name)

		// Normalize for system lib autopkgtest
		if source == runtime.GOOS {
			source = constants.PlatformSource
		}

		var c consent.CFile
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
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		rel, err := filepath.Rel(insightsDir, path)
		require.NoError(t, err)

		parts := strings.Split(rel, string(os.PathSeparator))
		require.Len(t, parts, 3, "Unexpected report path structure: %s", rel)
		source := parts[0]

		// Normalize for system lib autopkgtest
		if source == runtime.GOOS {
			source = constants.PlatformSource
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
	if os.IsNotExist(err) {
		return reports
	}
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

		// Normalize for system lib autopkgtest
		if source == runtime.GOOS {
			source = constants.PlatformSource
		}

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

func countLogLevels(t *testing.T, logPath string) map[int]int {
	t.Helper()
	f, err := os.Open(logPath)
	if os.IsNotExist(err) {
		return map[int]int{}
	}
	require.NoError(t, err)
	defer f.Close()

	counts := make(map[int]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		var level int
		if n, _ := fmt.Sscanf(line, "[LIBINSIGHTS][%d]", &level); n == 1 {
			counts[level]++
		}
	}
	require.NoError(t, scanner.Err(), "Failed to scan log file")
	return counts
}
