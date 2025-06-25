package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/database"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/models"
)

func TestConnect(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config database.Config

		wantErr bool
	}{
		"valid config": {
			config: database.Config{
				Host: "localhost",
				Port: 5432,
			},
			wantErr: false,
		},
		"bad port errors": {
			config: database.Config{
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mgr, err := database.Connect(t.Context(), tc.config, database.WithNewPool(mockNewDBPool(t, mockDBPool{})))
			if (err != nil) != tc.wantErr {
				t.Fatalf("Connect() error = %v, wantErr %v", err, tc.wantErr)
			}
			if mgr != nil {
				mgr.Close()
			}
		})
	}
}

func TestUpload(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id         string
		data       *models.TargetModel
		earlyClose bool
		execErr    error

		wantErr bool
	}{
		"successful exec": {id: uuid.NewString()},
		"opt-out successful exec": {
			data: &models.TargetModel{
				OptOut: true,
			},
		},

		// Error cases
		"exec error": {
			execErr: fmt.Errorf("error requested by test"),
			wantErr: true,
		},
		"errors if pool is nil or closed": {
			earlyClose: true,
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dbPool := mockDBPool{
				execErr: tc.execErr,
			}

			mgr, err := database.Connect(t.Context(), database.Config{}, database.WithNewPool(mockNewDBPool(t, dbPool)))
			require.NoError(t, err, "Setup: Connect() error")
			defer mgr.Close()

			if tc.earlyClose {
				require.NoError(t, mgr.Close(), "Setup: failed to close database connection")
			}

			if tc.data == nil {
				tc.data = &models.TargetModel{}
			}

			err = mgr.Upload(t.Context(), tc.id, "test", tc.data)
			if tc.wantErr {
				require.Error(t, err, "Upload() error")
				return
			}
			require.NoError(t, err, "Upload() error")
		})
	}
}

func TestUploadLegacy(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id         string
		report     *models.LegacyTargetModel
		earlyClose bool
		execErr    error

		wantErr bool
	}{
		"successful exec": {id: uuid.NewString()},
		"opt-out successful exec": {
			report: &models.LegacyTargetModel{
				OptOut: true,
			},
		},

		// Error cases
		"exec error": {
			execErr: fmt.Errorf("error requested by test"),
			wantErr: true,
		},
		"errors if pool is nil or closed": {
			earlyClose: true,
			wantErr:    true,
		}}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dbPool := mockDBPool{
				execErr: tc.execErr,
			}

			mgr, err := database.Connect(t.Context(), database.Config{}, database.WithNewPool(mockNewDBPool(t, dbPool)))
			require.NoError(t, err, "Setup: Connect() error")
			defer mgr.Close()

			if tc.earlyClose {
				require.NoError(t, mgr.Close(), "Setup: failed to close database connection")
			}

			if tc.report == nil {
				tc.report = &models.LegacyTargetModel{}
			}

			err = mgr.UploadLegacy(t.Context(), tc.id, "test-distribution", "test-version", tc.report)
			if tc.wantErr {
				require.Error(t, err, "Expected error on UploadLegacy() but got none")
				return
			}
			require.NoError(t, err, "Unexpected error on UploadLegacy()")
		})
	}
}

func TestUploadInvalid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id         string
		app        string
		rawReport  string
		earlyClose bool
		execErr    error

		wantErr bool
	}{
		"successful exec": {
			id:        uuid.NewString(),
			app:       "test-app",
			rawReport: "raw report data",
		},
		"empty exec": {},

		// Error cases
		"exec error": {
			execErr: fmt.Errorf("error requested by test"),
			wantErr: true,
		},
		"errors if pool is nil or closed": {
			earlyClose: true,
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dbPool := mockDBPool{
				execErr: tc.execErr,
			}

			mgr, err := database.Connect(t.Context(), database.Config{}, database.WithNewPool(mockNewDBPool(t, dbPool)))
			require.NoError(t, err, "Setup: Connect() error")
			defer mgr.Close()

			if tc.earlyClose {
				require.NoError(t, mgr.Close(), "Setup: failed to close database connection")
			}

			err = mgr.UploadInvalid(t.Context(), tc.id, tc.app, tc.rawReport)
			if tc.wantErr {
				require.Error(t, err, "UploadInvalid() error")
				return
			}
			require.NoError(t, err, "UploadInvalid() error")
		})
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		closeDelay time.Duration

		wantErr bool
	}{
		"successful close": {
			closeDelay: 0,
			wantErr:    false,
		},
		"delayed close": {
			closeDelay: 1 * time.Second,
			wantErr:    false,
		},
		"blocking close": {
			closeDelay: 15 * time.Second,
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dbPool := mockDBPool{
				closeDelay: tc.closeDelay,
			}

			mgr, err := database.Connect(t.Context(), database.Config{}, database.WithNewPool(mockNewDBPool(t, dbPool)))
			require.NoError(t, err, "Setup: Connect() error")
			defer mgr.Close()

			err = mgr.Close()
			if tc.wantErr {
				require.Error(t, err, "expected error on close")
				return
			}
			require.NoError(t, err, "Close() error")

			// No error after second close
			require.NoError(t, mgr.Close(), "Close should not error on second call")
		})
	}
}

func mockNewDBPool(t *testing.T, dbPool mockDBPool) func(ctx context.Context, dsn string) (database.DBPool, error) {
	t.Helper()
	return func(ctx context.Context, dsn string) (database.DBPool, error) {
		// If dsn port is negative, simulate a connection error
		_, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, err
		}

		return dbPool, nil
	}
}

type mockDBPool struct {
	execErr    error
	closeDelay time.Duration
}

func (m mockDBPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, m.execErr
}

func (m mockDBPool) Close() {
	if m.closeDelay > 0 {
		time.Sleep(m.closeDelay)
	}
}
