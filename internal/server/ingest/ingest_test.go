package ingest_test

import (
	"context"
	"errors"
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

func TestRunExistingSingle(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cm       mockConfigManager
		dbConfig mockDBManager
		options  []ingest.Options

		removeFiles []string

		wantErr bool
	}{
		"Successful run": {
			cm:       mockConfigManager{loadErr: nil},
			dbConfig: mockDBManager{},
			wantErr:  false,
		},
		"Config load failure": {
			cm:       mockConfigManager{loadErr: errors.New("load error")},
			dbConfig: mockDBManager{},
			wantErr:  true,
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
					return &tc.dbConfig, nil
				}),
			}
			s, err := ingest.New(t.Context(), tc.cm, database.Config{}, opts...)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			runErr := make(chan error, 1)
			go func() {
				defer close(runErr)
				err := s.Run()
				if err != nil {
					runErr <- err
				}
			}()

			// Allow time for the service to start and run
			select {
			case err := <-runErr:
				if tc.wantErr {
					require.Error(t, err, "Expected error but got nil")
					return
				}
				// Unexpected early close
				require.Fail(t, "Service closed unexpectedly: %v", err)
			case <-time.After(5 * time.Second):
			}

			// Simulate a graceful shutdown
			s.Quit(false)
			select {
			case err := <-runErr:
				require.NoError(t, err, "Expected no error but got: %v", err)
			case <-time.After(3 * time.Second):
				require.Fail(t, "Service did not close within the expected time")
			}

			// Check if the files were processed and uploaded to the database
			remainingFiles, err := testutils.GetDirContents(t, tc.cm.BaseDir(), 4)
			require.NoError(t, err, "Failed to get directory contents")

			got := struct {
				RemainingFiles map[string]string
				UploadedFiles  map[string][]*models.TargetModel
			}{
				RemainingFiles: remainingFiles,
				UploadedFiles:  tc.dbConfig.data,
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want.RemainingFiles, got.RemainingFiles, "Remaining files do not match")
			assert.Equal(t, want.UploadedFiles, got.UploadedFiles, "Uploaded files do not match")
		})
	}
}

type mockConfigManager struct {
	baseDir   string
	allowList []string

	earlyClose bool
	loadErr    error
	watchErr   error
}

func (m mockConfigManager) Load() error {
	return m.loadErr
}

func (m mockConfigManager) Watch(ctx context.Context) (<-chan struct{}, <-chan error, error) {
	if m.watchErr != nil {
		return nil, nil, m.watchErr
	}
	reloadCh := make(chan struct{})
	errCh := make(chan error)

	if m.earlyClose {
		close(reloadCh)
		close(errCh)
	}
	return reloadCh, errCh, nil
}

func (m mockConfigManager) AllowList() []string {
	return m.allowList
}

func (m mockConfigManager) BaseDir() string {
	return m.baseDir
}

type mockDBManager struct {
	closeErr  error
	uploadErr error
	data      map[string][]*models.TargetModel // Fake in-memory database
}

func newMockDBManager() *mockDBManager {
	return &mockDBManager{
		data: make(map[string][]*models.TargetModel),
	}
}

func (m *mockDBManager) Upload(ctx context.Context, app string, data *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	// Simulate storing the data in the fake database
	m.data[app] = append(m.data[app], data)
	return nil
}

func (m *mockDBManager) Close() error {
	return m.closeErr
}
