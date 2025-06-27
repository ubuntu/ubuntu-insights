package daemon_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/cmd/ingest-service/daemon"
	serverTestUtils "github.com/ubuntu/ubuntu-insights/server/internal/common/testutils"
	ingestTestUtils "github.com/ubuntu/ubuntu-insights/server/internal/ingest/testutils"
)

func TestMigrateRequiresDirArgument(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Make a fake file in dir
	fakeMigration := filepath.Join(dir, "fake.sql")
	require.NoError(t, os.WriteFile(fakeMigration, []byte(""), 0600), "Setup: couldn't write fake migration file")
	trueMigrationsDir := filepath.Join(serverTestUtils.ModuleRoot(), "migrations")

	tests := map[string]struct {
		args               []string
		noDatabase         bool
		preApplyMigrations bool

		wantErr      bool
		wantUsageErr bool
	}{
		"basic migration": {
			args: []string{trueMigrationsDir},
		},
		"pre-applied migrations": {
			args:               []string{trueMigrationsDir},
			preApplyMigrations: true,
		},

		// Usage Error Cases
		"no path": {
			wantErr:      true,
			wantUsageErr: true,
		},
		"non-existent path": {
			args:         []string{filepath.Join(dir, "non-existent-folder")},
			wantErr:      true,
			wantUsageErr: true,
		},
		"path to file": {
			args:         []string{fakeMigration},
			wantErr:      true,
			wantUsageErr: true,
		},

		// Error Cases
		"no database": {
			args:       []string{trueMigrationsDir},
			noDatabase: true,
			wantErr:    true,
		},
		"empty migrations directory": {
			args:    []string{dir},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := &ingestTestUtils.PostgresContainer{}
			if !tc.noDatabase {
				db = ingestTestUtils.StartPostgresContainer(t)

				tc.args = append(tc.args,
					"--db-host", db.Host,
					"--db-port", db.Port,
					"--db-user", db.User,
					"--db-password", db.Password,
					"--db-name", db.Name,
					"-vv")

				require.NoError(t, db.IsReady(t, 5*time.Second, 10), "Setup: dbContainer was not ready in time")
				if tc.preApplyMigrations {
					ingestTestUtils.ApplyMigrations(t, db.DSN, trueMigrationsDir)
				}
			}

			a, err := daemon.New()
			require.NoError(t, err, "Setup: New should not return an error")
			args := append([]string{"migrate"}, tc.args...)
			a.SetArgs(args...)

			err = a.Run()
			require.Equal(t, tc.wantUsageErr, a.UsageError(), "Run should return a usage error if expected")
			if tc.wantErr {
				require.Error(t, err, "Run should return an error")
				return
			}
			require.NoError(t, err, "Run should not return an error")

			// Get list of database tables
			got := ingestTestUtils.DBListTables(t, db.DSN)
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.ElementsMatch(t, want, got, "Run should create the expected tables in the database")
		})
	}
}
