package ingest_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		cm       ingest.DConfigManager
		dbConfig database.Config
		options  []ingest.Options

		wantErr bool
	}{
		"Successful creation": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			options:  nil,
			wantErr:  false,
		},
		"Config load failure": {
			cm:       &mockConfigManager{loadErr: errors.New("load error")},
			dbConfig: database.Config{},
			options:  nil,
			wantErr:  true,
		},
		"Database connection failure": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			options: []ingest.Options{
				ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
					return nil, errors.New("db connect error")
				}),
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := ingest.New(t.Context(), tc.cm, tc.dbConfig, tc.options...)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, s, "Returned service should not be nil")
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cm       *mockConfigManager
		dbConfig *mockDBManager

		removeFiles []string
		reAddFiles  bool

		wantErr bool
	}{
		"SingleValid runs": {
			cm:       &mockConfigManager{allowList: []string{"SingleValid"}},
			dbConfig: &mockDBManager{},
		},
		"SingleInvalid deletes invalid": {
			cm:       &mockConfigManager{allowList: []string{"SingleInvalid"}},
			dbConfig: &mockDBManager{},
		},
		"OptOut runs": {
			cm:       &mockConfigManager{allowList: []string{"OptOut"}},
			dbConfig: &mockDBManager{},
		},
		"MultiMixed runs": {
			cm:       &mockConfigManager{allowList: []string{"MultiMixed"}},
			dbConfig: &mockDBManager{},
		},
		"All apps runs": {
			cm:       &mockConfigManager{allowList: []string{"SingleValid", "SingleInvalid", "OptOut", "MultiMixed"}},
			dbConfig: &mockDBManager{},
		},

		// Re-add files during run
		"Re-add files": {
			cm:         &mockConfigManager{allowList: []string{"MultiMixed"}},
			dbConfig:   &mockDBManager{},
			reAddFiles: true,
		},

		// Error cases
		"Config watch failure errors": {
			cm:       &mockConfigManager{watchErr: errors.New("Requested watch error")},
			dbConfig: &mockDBManager{},
			wantErr:  true,
		},
		"Delayed config watch failure errors": {
			cm:       &mockConfigManager{delayedWatchErr: errors.New("Delayed watch error")},
			dbConfig: &mockDBManager{},
			wantErr:  true,
		},
		"DB upload failure errors": {
			cm:       &mockConfigManager{allowList: []string{"SingleValid"}},
			dbConfig: &mockDBManager{uploadErr: errors.New("Upload error")},
			wantErr:  true,
		},
		"DB close failure errors": {
			cm:       &mockConfigManager{allowList: []string{"SingleValid"}},
			dbConfig: &mockDBManager{closeErr: errors.New("Close error")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dst := ingest.CopyTestFixtures(t, tc.removeFiles)
			if tc.cm.baseDir == "" {
				tc.cm.baseDir = dst
			}

			opts := []ingest.Options{
				ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
					return tc.dbConfig, nil
				}),
			}
			s, err := ingest.New(t.Context(), tc.cm, database.Config{}, opts...)
			require.NoError(t, err, "Setup: Failed to create service")

			runErr := make(chan error, 1)
			go func() {
				defer close(runErr)
				err := s.Run()
				if err != nil {
					runErr <- err
				}
			}()

			// Allow time for the service to start and run
			if runWait(t, runErr, tc.wantErr, 4*time.Second) {
				return
			}

			waitForUploaderToBeIdle(t, tc.dbConfig, 2*time.Second, 15*time.Second)

			if tc.reAddFiles {
				// Re-add files to the directory
				err := testutils.CopyDir(t, filepath.Join("testdata", "fixtures"), dst)
				require.NoError(t, err, "Setup: failed to re-copy test fixtures")

				waitForUploaderToBeIdle(t, tc.dbConfig, 5*time.Second, 20*time.Second)
				runWait(t, runErr, false, 500*time.Millisecond)
			}

			// Simulate a graceful shutdown
			gracefulShutdown(t, s, runErr)
			checkRunResults(t, tc.cm, tc.dbConfig)
		})
	}
}

// Tests the addition of a new valid app to the allow list
// and verifies that the app is processed correctly.
func TestRunNewApp(t *testing.T) {
	t.Parallel()

	dst := ingest.CopyTestFixtures(t, nil)
	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},
		baseDir:   dst,

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}

	opts := []ingest.Options{
		ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
			return db, nil
		}),
	}
	s, err := ingest.New(t.Context(), cm, database.Config{}, opts...)
	require.NoError(t, err, "Setup: Failed to create service")

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()

	// Allow time for the service to start and run
	runWait(t, runErr, false, 3*time.Second)

	// Add MultiMixed to the allow list (send burst of signals)
	cm.SetAllowList(append(cm.AllowList(), "MultiMixed"))
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}

	runWait(t, runErr, false, 15*time.Second)
	waitForUploaderToBeIdle(t, db, 4*time.Second, 20*time.Second)

	gracefulShutdown(t, s, runErr)
	checkRunResults(t, cm, db)
}

// TestRunRemoveApp tests the removal of an app from the allow list
// and verifies that the app is no longer processed.
func TestRunRemoveApp(t *testing.T) {
	t.Parallel()

	dst := ingest.CopyTestFixtures(t, nil)
	cm := &mockConfigManager{
		allowList: []string{"SingleValid", "MultiMixed"},
		baseDir:   dst,

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}

	opts := []ingest.Options{
		ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
			return db, nil
		}),
	}
	s, err := ingest.New(t.Context(), cm, database.Config{}, opts...)
	require.NoError(t, err, "Setup: Failed to create service")

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()

	// Allow time for the service to start and run
	waitForUploaderToBeIdle(t, db, 2*time.Second, 20*time.Second)
	runWait(t, runErr, false, 500*time.Millisecond)

	// Add MultiMixed to the allow list (send burst of signals)
	cm.SetAllowList([]string{"SingleValid"})
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}

	runWait(t, runErr, false, 10*time.Second)

	err = testutils.CopyDir(t, filepath.Join("testdata", "fixtures"), dst)
	require.NoError(t, err, "Setup: failed to re-copy test fixtures")
	waitForUploaderToBeIdle(t, db, 4*time.Second, 20*time.Second)
	runWait(t, runErr, false, 500*time.Millisecond)

	gracefulShutdown(t, s, runErr)
	checkRunResults(t, cm, db)
}

func TestRunAfterQuitErrors(t *testing.T) {
	t.Parallel()

	dst := ingest.CopyTestFixtures(t, nil)
	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},
		baseDir:   dst,

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}

	opts := []ingest.Options{
		ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
			return db, nil
		}),
	}
	s, err := ingest.New(t.Context(), cm, database.Config{}, opts...)
	require.NoError(t, err, "Setup: Failed to create service")
	defer s.Quit(true)

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()

	runWait(t, runErr, false, 3*time.Second)
	gracefulShutdown(t, s, runErr)

	runErr = make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()
	runWait(t, runErr, true, 3*time.Second)
}

type mockConfigManager struct {
	baseDir   string
	allowList []string

	earlyClose      bool
	loadErr         error
	watchErr        error
	delayedWatchErr error

	reloadCh chan struct{}
	errCh    chan error

	mu sync.RWMutex // Mutex to protect access to the allowList
}

func (m *mockConfigManager) Load() error {
	return m.loadErr
}

func (m *mockConfigManager) Watch(ctx context.Context) (<-chan struct{}, <-chan error, error) {
	if m.watchErr != nil {
		return nil, nil, m.watchErr
	}

	if m.reloadCh == nil {
		m.reloadCh = make(chan struct{})
	}

	if m.errCh == nil {
		m.errCh = make(chan error)
	}

	if m.earlyClose {
		close(m.reloadCh)
		close(m.errCh)
	} else if m.delayedWatchErr != nil {
		go func() {
			time.Sleep(2 * time.Second)
			m.errCh <- m.delayedWatchErr
		}()
	}
	return m.reloadCh, m.errCh, nil
}

func (m *mockConfigManager) AllowList() []string {
	m.mu.RLock() // Lock for reading
	defer m.mu.RUnlock()
	return m.allowList
}

func (m *mockConfigManager) SetAllowList(newAllowList []string) {
	m.mu.Lock() // Lock for writing
	defer m.mu.Unlock()
	m.allowList = newAllowList
}

func (m *mockConfigManager) BaseDir() string {
	return m.baseDir
}

type mockDBManager struct {
	closeErr       error
	uploadErr      error
	data           map[string][]*models.TargetModel // Fake in-memory database
	mu             sync.Mutex                       // Mutex to protect access to the data map
	lastUploadTime time.Time                        // Time of the last upload
}

func (m *mockDBManager) Upload(ctx context.Context, app string, data *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	m.mu.Lock() // Lock the mutex before accessing the data map
	defer m.mu.Unlock()

	if m.data == nil {
		m.data = make(map[string][]*models.TargetModel)
	}

	// Simulate storing the data in the fake database
	m.data[app] = append(m.data[app], data)
	m.lastUploadTime = time.Now()
	return nil
}

func (m *mockDBManager) IsUploaderActiveWithin(duration time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return time.Since(m.lastUploadTime) <= duration
}

func (m *mockDBManager) Close() error {
	return m.closeErr
}

// runWait is a helper function which waits a specified duration, unless an error signal is received.
//
// If an error is received, and expectErr is true, it returns true.
func runWait(t *testing.T, runErr chan error, expectErr bool, duration time.Duration) bool {
	t.Helper()

	select {
	case err := <-runErr:
		if expectErr {
			require.Error(t, err, "Expected error but got nil")
			return true
		}
		// Unexpected early close
		require.Fail(t, "Service closed unexpectedly: %v", err)
	case <-time.After(duration):
	}

	return false
}

// gracefulShutdown is a helper function which simulates a graceful shutdown of the service.
// If the service does not shutdown within 8 seconds, it fails the test.
// If runErr receives an error during shutdown, it fails the test.
func gracefulShutdown(t *testing.T, s *ingest.Service, runErr chan error) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		s.Quit(false)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		require.Fail(t, "Service failed to shutdown gracefully within 8 seconds")
	}

	// Check for any errors during shutdown
	select {
	case err := <-runErr:
		require.NoError(t, err, "Service failed to shutdown gracefully")
	case <-time.After(2 * time.Second):
		require.Fail(t, "Service has not returned aftery 2 seconds")
	}
}

// checkRunResults is a helper function which checks the results of the run
// and compares them to the expected results.
func checkRunResults(t *testing.T, cm *mockConfigManager, db *mockDBManager) {
	t.Helper()

	db.mu.Lock()
	defer db.mu.Unlock()

	remainingFiles, err := testutils.GetDirHashedContents(t, cm.BaseDir(), 4)
	require.NoError(t, err, "Failed to get directory contents")

	results := struct {
		RemainingFiles map[string]string
		UploadedFiles  map[string][]*models.TargetModel
	}{
		RemainingFiles: remainingFiles,
		UploadedFiles:  db.data,
	}

	got, err := json.MarshalIndent(results, "", "  ")
	require.NoError(t, err)
	want := testutils.LoadWithUpdateFromGolden(t, string(got))
	assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Unexpected results after processing files")
}

// waitForUploaderToBeIdle is a helper function which waits for the uploader to be idle for a specified duration.
//
// If it does not become idle within the timeout, it fails the test.
func waitForUploaderToBeIdle(t *testing.T, db *mockDBManager, idleDuration time.Duration, timeout time.Duration) {
	t.Helper()

	start := time.Now()
	for time.Since(start) < timeout {
		if !db.IsUploaderActiveWithin(idleDuration) {
			return
		}
		time.Sleep(500 * time.Millisecond) // Small delay between checks
	}

	require.Fail(t, "Uploader did not become idle within the timeout")
}
