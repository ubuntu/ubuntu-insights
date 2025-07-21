package webservice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
	"github.com/ubuntu/ubuntu-insights/server/internal/webservice"
)

var defaultDaemonConfig = &webservice.StaticConfig{
	ReadTimeout:    5 * time.Second,
	WriteTimeout:   10 * time.Second,
	RequestTimeout: 3 * time.Second,
	MaxHeaderBytes: 1 << 13, // 8 KB
	MaxUploadBytes: 1 << 17, // 128 KB

	ListenHost: "localhost",
}

var muPortAcquire = sync.Mutex{}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmLoadErr error

		wantErr bool
	}{
		"Empty valid": {},
		"ConfigManager load error errors": {
			cmLoadErr: assert.AnError,
			wantErr:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			daemonConfig := webservice.StaticConfig{
				ConfigPath: webservice.GenerateTestDaemonConfig(t, &config.Conf{}),
				ReportsDir: t.TempDir(),
			}

			cm := &testConfigManager{
				allowList: []string{"test"},
				loadErr:   tc.cmLoadErr,
			}

			s, err := webservice.New(t.Context(), cm, daemonConfig)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, s)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, s)
		})
	}
}

func TestServeMulti(t *testing.T) {
	t.Parallel()
	const defaultApp = "goodapp"
	dConf := *defaultDaemonConfig
	cm := &testConfigManager{allowList: []string{defaultApp, "ubuntu-report/distribution/desktop/version"}}

	s := createServerAndWaitReady(t, cm, &dConf, false)

	tests := map[string]struct {
		method      string
		path        string
		contentType string
		body        []byte
		wantStatus  int
		checkDir    string
	}{
		"Version": {
			method:     http.MethodGet,
			path:       "/version",
			wantStatus: http.StatusOK,
		},
		"Path NotFound": {
			method:     http.MethodGet,
			path:       "/nope",
			wantStatus: http.StatusNotFound,
		},
		"Bad method MethodNotAllowed": {
			method:     http.MethodGet,
			path:       "/upload/goodapp",
			wantStatus: http.StatusMethodNotAllowed,
		},
		"Bad App ForbiddenApp": {
			method:      http.MethodPost,
			path:        "/upload/badapp",
			contentType: "application/json",
			body:        []byte(`{"foo":"bar"}`),
			wantStatus:  http.StatusForbidden,
		},
		"InvalidJSON BadRequest": {
			method:      http.MethodPost,
			path:        "/upload/goodapp",
			contentType: "application/json",
			body:        []byte(`not-json`),
			wantStatus:  http.StatusBadRequest,
		},
		"Valid Upload Accepted": {
			method:      http.MethodPost,
			path:        "/upload/goodapp",
			contentType: "application/json",
			body:        []byte(`{"foo":"bar"}`),
			checkDir:    defaultApp,
			wantStatus:  http.StatusAccepted,
		},
		"Ubuntu Report backwards compatibility": {
			method:      http.MethodPost,
			path:        "/distribution/desktop/version",
			contentType: "application/json",
			body:        []byte(`{"foo":"bar"}`),
			checkDir:    "ubuntu-report/distribution/desktop/version",
			wantStatus:  http.StatusOK,
		},
	}
	client := &http.Client{}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(tc.method, "http://"+s.Addr()+tc.path, bytes.NewReader(tc.body))
			require.NoError(t, err, "Setup: failed to create request")
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.wantStatus, resp.StatusCode, "Unexpected status response")
			if tc.checkDir != "" {
				files, err := os.ReadDir(filepath.Join(dConf.ReportsDir, tc.checkDir))
				require.NoError(t, err)
				assert.Len(t, files, 1, "one file created")
				data, err := os.ReadFile(filepath.Join(dConf.ReportsDir, tc.checkDir, files[0].Name()))
				assert.NoError(t, err)

				var got map[string]any
				assert.NoError(t, json.Unmarshal(data, &got))
				want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
				assert.Equal(t, want, got, "")
			}
		})
	}
}

func TestRunSingle(t *testing.T) {
	t.Parallel()

	const defaultApp = "goodapp"

	tests := map[string]struct {
		dConf webservice.StaticConfig
		cm    testConfigManager

		method      string
		path        string
		contentType string
		body        []byte

		checkDir   string
		wantStatus int
		wantErr    bool
	}{
		"Version": {
			method:     http.MethodGet,
			path:       "/version",
			wantStatus: http.StatusOK,
		},
		"Basic Upload": {checkDir: defaultApp},
		"Ubuntu Report backwards compatibility": {
			method:      http.MethodPost,
			path:        "/distribution/desktop/version",
			contentType: "application/json",
			cm: testConfigManager{
				allowList: []string{defaultApp,
					"ubuntu-report/distribution/desktop/version"},
			},
			checkDir:   "ubuntu-report/distribution/desktop/version",
			wantStatus: http.StatusOK,
		},

		// Bad Requests
		"Bad App StatusForbidden": {
			path:       "/upload/badapp",
			wantStatus: http.StatusForbidden,
		},
		"Bad JSON StatusBadRequest": {
			body:       []byte(`not-json`),
			wantStatus: http.StatusBadRequest,
		},
		"Bad Method StatusMethodNotAllowed": {
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
		"Bad Path StatusNotFound": {
			path:       "/unknown-path",
			wantStatus: http.StatusNotFound,
		},
		"Bad legacy path StatusNotFound": {
			method:      http.MethodPost,
			path:        "/distribution/desktop/badapp",
			contentType: "application/json",
			cm: testConfigManager{
				allowList: []string{defaultApp,
					"ubuntu-report/distribution/desktop/version"},
			},
			wantStatus: http.StatusForbidden,
		},

		// Bad Server Configurations
		"Bad Port": {
			dConf: func() webservice.StaticConfig {
				d := *defaultDaemonConfig
				d.ListenPort = -1
				return d
			}(),
			wantErr: true,
		},
		"New Watcher Error": {
			cm: testConfigManager{
				allowList:     []string{defaultApp},
				newWatcherErr: fmt.Errorf("requested watch error"),
			},
			wantErr: true,
		},
		"Watch Error": {
			cm: testConfigManager{
				allowList: []string{defaultApp},
				watchErr:  fmt.Errorf("requested watch error"),
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.dConf == (webservice.StaticConfig{}) {
				tc.dConf = *defaultDaemonConfig
			}

			if tc.method == "" {
				tc.method = http.MethodPost
			}
			if tc.path == "" {
				tc.path = "/upload/" + defaultApp
			}
			if tc.contentType == "" {
				tc.contentType = "application/json"
			}
			if tc.body == nil {
				tc.body = []byte(`{"foo":"bar"}`)
			}
			if tc.wantStatus == 0 {
				tc.wantStatus = http.StatusAccepted
			}
			if tc.cm.allowList == nil {
				tc.cm.allowList = []string{defaultApp}
			}

			s := createServerAndWaitReady(t, &tc.cm, &tc.dConf, tc.wantErr)
			if tc.wantErr {
				return // If we expect an error and createServerAndWaitReady returns, we can stop here
			}

			req, err := http.NewRequest(tc.method, "http://"+s.Addr()+tc.path, bytes.NewReader(tc.body))
			require.NoError(t, err, "Setup: failed to create request")
			req.Header.Set("Content-Type", tc.contentType)
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.wantStatus, resp.StatusCode, "status")

			// Check files and file content, ignore uuid name
			if tc.checkDir != "" {
				contents, err := testutils.GetDirContents(t, filepath.Join(tc.dConf.ReportsDir, tc.checkDir), 3)
				require.NoError(t, err)
				assert.Len(t, contents, 1, "one file created")
				var got string
				for _, v := range contents {
					got = v
					break
				}
				want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
				assert.Equal(t, want, got, "unexpected, file content")
			}
		})
	}
}

func TestRunAfterQuitErrors(t *testing.T) {
	t.Parallel()

	dConf := *defaultDaemonConfig
	cm := &testConfigManager{allowList: []string{}}

	s := createServerAndWaitReady(t, cm, &dConf, false)

	s.Quit(false)
	testutils.WaitForPortClosed(t, dConf.ListenHost, dConf.ListenPort, 3*time.Second)

	serverErr2 := make(chan error, 1)
	go func() {
		defer close(serverErr2)
		serverErr2 <- s.Run()
	}()

	select {
	case err := <-serverErr2:
		require.Error(t, err, "Server should have errored after second run")
	case <-time.After(1 * time.Second):
		require.Fail(t, "Server should have errored after second run")
	}

	require.False(t, testutils.PortOpen(t, dConf.ListenHost, dConf.ListenPort), "Server should not be running after second (failed) run")
}

type testConfigManager struct {
	allowList     []string
	finishWatch   bool
	loadErr       error
	newWatcherErr error
	watchErr      error
}

func (t testConfigManager) Load() error {
	return t.loadErr
}

func (t testConfigManager) Watch(ctx context.Context) (<-chan struct{}, <-chan error, error) {
	// Simulate watching for changes
	if t.finishWatch {
		<-ctx.Done()
	}
	if t.newWatcherErr != nil {
		return nil, nil, t.newWatcherErr
	}

	eventsChan := make(chan struct{})
	errorsChan := make(chan error)
	go func() {
		defer close(eventsChan)
		defer close(errorsChan)

		if t.watchErr != nil {
			errorsChan <- t.watchErr
			return
		}

		// Block until the context is done
		<-ctx.Done()
	}()

	return eventsChan, errorsChan, nil
}

func (t testConfigManager) AllowList() []string {
	return t.allowList
}

func (t testConfigManager) AllowSet() map[string]struct{} {
	allowSet := make(map[string]struct{}, len(t.allowList))
	for _, name := range t.allowList {
		allowSet[name] = struct{}{}
	}
	return allowSet
}

func newForTest(t *testing.T, cm *testConfigManager, daemonConfig *webservice.StaticConfig) *webservice.Server {
	t.Helper()

	if daemonConfig.ReportsDir == "" {
		daemonConfig.ReportsDir = t.TempDir()
	}

	if daemonConfig.ListenPort == 0 {
		daemonConfig.ListenPort = testutils.GetFreePort(t, daemonConfig.ListenHost, testutils.TCP)
	}

	if daemonConfig.ConfigPath == "" {
		daemonConfig.ConfigPath = webservice.GenerateTestDaemonConfig(t, &config.Conf{
			AllowedList: cm.AllowList(),
		})
	}

	s, err := webservice.New(t.Context(), cm, *daemonConfig)
	require.NoError(t, err, "Setup: failed to create server")
	return s
}

// createServerAndWaitReady initializes and starts a webservice server for testing.
// It waits for the server to be ready and returns the server instance.
// If expectErr is true, it expects the server to fail to start and returns the server instance anyway.
// If expectErr is false, it ensures the server starts successfully and is ready to accept requests.
func createServerAndWaitReady(t *testing.T, cm *testConfigManager, daemonConfig *webservice.StaticConfig, expectErr bool) *webservice.Server {
	t.Helper()

	muPortAcquire.Lock()
	defer muPortAcquire.Unlock()

	s := newForTest(t, cm, daemonConfig)
	t.Cleanup(func() {
		s.Quit(true)
	})

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		runErr <- s.Run()
	}()

	select {
	case err := <-runErr:
		if expectErr {
			require.Error(t, err, "Run should fail")
			return s
		}
		require.NoError(t, err, "Run should not fail")
	case <-time.After(1 * time.Second):
		require.False(t, expectErr, "Expected Run to fail with error, but it did not")
		waitServerReady(t, s)
	}

	require.True(t, testutils.PortOpen(t, daemonConfig.ListenHost, daemonConfig.ListenPort), "Server should be running on specified address")

	return s
}

func waitServerReady(t *testing.T, s *webservice.Server) {
	t.Helper()

	const (
		timeout  = 5 * time.Second
		interval = 50 * time.Millisecond
	)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + s.Addr() + "/version")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}

		time.Sleep(interval)
	}

	require.True(t, time.Now().Before(deadline), "Setup: Server did not become ready in time")
}
