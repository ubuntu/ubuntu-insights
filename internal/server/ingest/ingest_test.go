package ingest_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/database"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

var testFixturesDir = filepath.Join("testdata", "fixtures")

func TestNew(t *testing.T) {
	tests := map[string]struct {
		cm       ingest.DConfigManager
		dbConfig database.Config
		sc       ingest.StaticConfig
		options  []ingest.Options

		wantErr bool
	}{
		"Successful creation": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			sc: ingest.StaticConfig{
				ReportsDir: t.TempDir(),
			},
			wantErr: false,
		},

		// Error cases
		"Config load failure": {
			cm:       &mockConfigManager{loadErr: errors.New("load error")},
			dbConfig: database.Config{},
			sc: ingest.StaticConfig{
				ReportsDir: t.TempDir(),
			},
			wantErr: true,
		},
		"Database connection failure": {
			cm:       &mockConfigManager{loadErr: nil},
			dbConfig: database.Config{},
			sc: ingest.StaticConfig{
				ReportsDir: t.TempDir(),
			},
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
			s, err := ingest.New(t.Context(), tc.cm, tc.dbConfig, tc.sc, tc.options...)
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
			cm: &mockConfigManager{allowList: []string{"SingleValid", "SingleInvalid", "OptOut", "MultiMixed",
				"ubuntu-report/distribution/desktop/version"}},
			dbConfig: &mockDBManager{},
		},
		"Legacy ubuntu report runs": {
			cm:       &mockConfigManager{allowList: []string{"ubuntu-report/distribution/desktop/version"}},
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

			sc := &ingest.StaticConfig{ReportsDir: ingest.CopyTestFixtures(t, tc.removeFiles)}
			s := newIngestService(t, tc.cm, tc.dbConfig, sc)
			runErr := run(t, s)

			// Allow time for the service to start and run
			if runWait(t, runErr, tc.wantErr, 4*time.Second) {
				return
			}

			waitForUploaderToBeIdle(t, tc.dbConfig, 2*time.Second, 15*time.Second)

			if tc.reAddFiles {
				// Re-add files to the directory
				err := testutils.CopyDir(t, testFixturesDir, sc.ReportsDir)
				require.NoError(t, err, "Setup: failed to re-copy test fixtures")

				waitForUploaderToBeIdle(t, tc.dbConfig, 8*time.Second, 20*time.Second)
				runWait(t, runErr, false, 500*time.Millisecond)
			}

			// Simulate a graceful shutdown
			gracefulShutdown(t, s, runErr)
			checkRunResults(t, tc.dbConfig, *sc)
		})
	}
}

// Tests the addition of a new valid app to the allow list
// and verifies that the app is processed correctly.
func TestRunNewApp(t *testing.T) {
	t.Parallel()

	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}
	sc := &ingest.StaticConfig{ReportsDir: ingest.CopyTestFixtures(t, nil)}

	s := newIngestService(t, cm, db, sc)
	runErr := run(t, s)

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
	checkRunResults(t, db, *sc)
}

// TestRunRemoveApp tests the removal of an app from the allow list
// and verifies that the app is no longer processed.
func TestRunRemoveApp(t *testing.T) {
	t.Parallel()

	cm := &mockConfigManager{
		allowList: []string{"SingleValid", "MultiMixed"},

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}
	sc := &ingest.StaticConfig{ReportsDir: ingest.CopyTestFixtures(t, nil)}

	s := newIngestService(t, cm, db, sc)
	runErr := run(t, s)

	// Allow time for the service to start and run
	waitForUploaderToBeIdle(t, db, 2*time.Second, 20*time.Second)
	runWait(t, runErr, false, 500*time.Millisecond)

	// Add MultiMixed to the allow list (send burst of signals)
	cm.SetAllowList([]string{"SingleValid"})
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}
	cm.reloadCh <- struct{}{}

	runWait(t, runErr, false, 10*time.Second)

	err := testutils.CopyDir(t, testFixturesDir, sc.ReportsDir)
	require.NoError(t, err, "Setup: failed to re-copy test fixtures")
	waitForUploaderToBeIdle(t, db, 4*time.Second, 20*time.Second)
	runWait(t, runErr, false, 500*time.Millisecond)

	gracefulShutdown(t, s, runErr)
	checkRunResults(t, db, *sc)
}

func TestRunAfterQuitErrors(t *testing.T) {
	t.Parallel()

	cm := &mockConfigManager{
		allowList: []string{"SingleValid"},

		reloadCh: make(chan struct{}),
		errCh:    make(chan error),
	}
	db := &mockDBManager{}
	s := newIngestService(t, cm, db, &ingest.StaticConfig{ReportsDir: ingest.CopyTestFixtures(t, nil)})
	defer s.Quit(true)

	runErr := run(t, s)

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

type mockDBManager struct {
	closeErr       error
	uploadErr      error
	reports        map[string][]*models.TargetModel       // Fake in-memory database
	legacyReports  map[string][]*models.LegacyTargetModel // Fake in-memory legacy reports
	invalidReports map[string][]string                    // Fake in-memory invalid reports
	mu             sync.Mutex                             // Mutex to protect access to the data map
	lastUploadTime time.Time                              // Time of the last upload
}

func (m *mockDBManager) Upload(ctx context.Context, app string, report *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	m.mu.Lock() // Lock the mutex before accessing the data map
	defer m.mu.Unlock()

	if m.reports == nil {
		m.reports = make(map[string][]*models.TargetModel)
	}

	// Simulate storing the data in the fake database
	m.reports[app] = append(m.reports[app], report)
	m.lastUploadTime = time.Now()
	return nil
}

func (m *mockDBManager) UploadLegacy(ctx context.Context, distribution, version string, report *models.LegacyTargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	m.mu.Lock() // Lock the mutex before accessing the legacyReports map
	defer m.mu.Unlock()

	if m.legacyReports == nil {
		m.legacyReports = make(map[string][]*models.LegacyTargetModel)
	}

	// Simulate storing the legacy report in the fake database
	key := constants.LegacyReportTag + "/" + distribution + "/desktop/" + version
	m.legacyReports[key] = append(m.legacyReports[key], report)
	m.lastUploadTime = time.Now()
	return nil
}

func (m *mockDBManager) UploadInvalid(ctx context.Context, id, app, rawReport string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.invalidReports == nil {
		m.invalidReports = make(map[string][]string)
	}

	if err := uuid.Validate(id); err != nil {
		return errors.New("invalid UUID format")
	}

	// Simulate storing the invalid report in the fake database
	rawReport = strings.ReplaceAll(rawReport, "\r\n", "\n") // Fix for Windows line endings
	reportHash := sha256.Sum256([]byte(rawReport))
	m.invalidReports[app] = append(m.invalidReports[app], hex.EncodeToString(reportHash[:]))
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

// run is a helper function which runs the service in a separate goroutine
// and returns a channel to receive any errors that occur during the run.
//
// The channel is closed when the run is complete.
func run(t *testing.T, s *ingest.Service) chan error {
	t.Helper()

	runErr := make(chan error, 1)
	go func() {
		defer close(runErr)
		err := s.Run()
		if err != nil {
			runErr <- err
		}
	}()

	return runErr
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
		require.Fail(t, "Service has not returned after 2 seconds")
	}
}

// checkRunResults is a helper function which checks the results of the run
// and compares them to the expected results.
func checkRunResults(t *testing.T, db *mockDBManager, sc ingest.StaticConfig) {
	t.Helper()

	db.mu.Lock()
	defer db.mu.Unlock()

	remainingFiles, err := testutils.GetDirHashedContents(t, sc.ReportsDir, 4)
	require.NoError(t, err, "Failed to get directory contents")

	referenceHashes, err := testutils.GetDirHashedContents(t, testFixturesDir, 4)
	require.NoError(t, err, "Failed to get reference directory contents")

	results := struct {
		RemainingFiles      map[string]string
		UploadedFiles       map[string][]*models.TargetModel
		UploadedLegacyFiles map[string][]*models.LegacyTargetModel
		InvalidFiles        map[string][]string
		ReferenceHashes     map[string]string
	}{
		RemainingFiles:      remainingFiles,
		UploadedFiles:       db.reports,
		UploadedLegacyFiles: db.legacyReports,
		InvalidFiles:        db.invalidReports,
		ReferenceHashes:     referenceHashes,
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

// newIngestService is a helper function which creates a new ingest service for testing purposes.
func newIngestService(t *testing.T, cm *mockConfigManager, db *mockDBManager, sc *ingest.StaticConfig) (s *ingest.Service) {
	t.Helper()

	opts := []ingest.Options{
		ingest.WithDBConnect(func(ctx context.Context, cfg database.Config) (ingest.DBManager, error) {
			return db, nil
		}),
	}
	s, err := ingest.New(t.Context(), cm, database.Config{}, *sc, opts...)
	require.NoError(t, err, "Setup: Failed to create service")

	return s
}
