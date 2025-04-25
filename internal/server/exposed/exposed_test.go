package exposed_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/exposed"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

var defaultDaemonConfig = &exposed.DaemonConfig{
	ReadTimeout:    5 * time.Second,
	WriteTimeout:   10 * time.Second,
	RequestTimeout: 3 * time.Second,
	MaxHeaderBytes: 1 << 13, // 8 KB
	MaxUploadBytes: 1 << 17, // 128 KB
	RateLimitPS:    0.1,
	BurstLimit:     3,

	ListenHost: "localhost",
}

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

			daemonConfig := &exposed.DaemonConfig{
				ConfigPath: exposed.GenerateTestDaeConfig(t, &config.Conf{}),
			}

			cm := &testConfigManager{
				allowList: []string{"test"},
				baseDir:   t.TempDir(),
				loadErr:   tc.cmLoadErr,
			}

			s, err := daemonConfig.New(t.Context(), cm)
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
	cm := &testConfigManager{allowList: []string{defaultApp}}
	s := newForTest(t, cm, &dConf)

	t.Cleanup(func() {
		s.Quit(true)
	})
	go func() {
		err := s.Run()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	tests := map[string]struct {
		method      string
		path        string
		contentType string
		body        []byte
		wantStatus  int
		checkFile   bool
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
			wantStatus:  http.StatusAccepted,
			checkFile:   true,
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

			assert.Equal(t, tc.wantStatus, resp.StatusCode, "status")
			if tc.checkFile {
				app := tc.path[len("/upload/"):]
				files, err := os.ReadDir(filepath.Join(cm.baseDir, app))
				require.NoError(t, err)
				assert.Len(t, files, 1, "one file created")
				data, err := os.ReadFile(filepath.Join(cm.baseDir, app, files[0].Name()))
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
		dConf exposed.DaemonConfig
		cm    testConfigManager

		method      string
		path        string
		contentType string
		body        []byte
		wantStatus  int
		wantErr     bool
	}{
		"Version": {
			method:     http.MethodGet,
			path:       "/version",
			wantStatus: http.StatusOK,
		},
		"Basic Upload": {},

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

		// Bad Server Configurations
		"Bad Port": {
			dConf: func() exposed.DaemonConfig {
				d := *defaultDaemonConfig
				d.ListenPort = -1
				return d
			}(),
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

			if tc.dConf == (exposed.DaemonConfig{}) {
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

			s := newForTest(t, &tc.cm, &tc.dConf)
			defer s.Quit(false)

			runErr := make(chan error, 1)
			go func() {
				defer close(runErr)
				runErr <- s.Run()
			}()
			time.Sleep(100 * time.Millisecond)

			select {
			case err := <-runErr:
				if tc.wantErr {
					require.Error(t, err, "Run should fail")
					return
				}
				require.NoError(t, err, "Run should not fail")
			case <-time.After(1 * time.Second):
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
			if tc.wantStatus == http.StatusAccepted {
				app := filepath.Base(tc.path)
				contents, err := testutils.GetDirContents(t, filepath.Join(tc.cm.baseDir, app), 3)
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
	s := newForTest(t, cm, &dConf)
	defer s.Quit(true)

	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)
		serverErr <- s.Run()
	}()

	select {
	case err := <-serverErr:
		require.Fail(t, "Server should not have errored", err)
	case <-time.After(1 * time.Second):
	}
	require.True(t, testutils.PortOpen(t, dConf.ListenHost, dConf.ListenPort), "Server should be running on specified addr``")
	s.Quit(false)
	time.Sleep(500 * time.Millisecond) // Let server quit
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
	allowList   []string
	baseDir     string
	finishWatch bool
	loadErr     error
	watchErr    error
}

func (t testConfigManager) Load() error {
	return t.loadErr
}

func (t testConfigManager) Watch(ctx context.Context) error {
	// Simulate watching for changes
	if t.finishWatch {
		<-ctx.Done()
	}
	if t.watchErr != nil {
		return t.watchErr
	}

	// Block until the context is done
	<-ctx.Done()
	return nil
}

func (t testConfigManager) AllowList() []string {
	return t.allowList
}

func (t testConfigManager) BaseDir() string {
	return t.baseDir
}

func newForTest(t *testing.T, cm *testConfigManager, daemonConfig *exposed.DaemonConfig) *exposed.Server {
	t.Helper()

	if cm.baseDir == "" {
		cm.baseDir = t.TempDir()
	}

	if daemonConfig.ListenPort == 0 {
		daemonConfig.ListenPort = testutils.GetFreePort(t, daemonConfig.ListenHost, testutils.TCP)
	}

	if daemonConfig.ConfigPath == "" {
		daemonConfig.ConfigPath = exposed.GenerateTestDaeConfig(t, &config.Conf{
			BaseDir:     cm.BaseDir(),
			AllowedList: cm.AllowList(),
		})
	}

	s, err := daemonConfig.New(t.Context(), cm)
	require.NoError(t, err, "Setup: failed to create server")
	return s
}
