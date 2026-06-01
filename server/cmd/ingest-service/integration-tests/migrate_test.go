package ingest_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	serverTestUtils "github.com/ubuntu/ubuntu-insights/server/internal/ingest/testutils"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	trueMigrationsDir := filepath.Join(serverTestUtils.ModuleRoot(), "migrations")
	fakeMigration := filepath.Join(tmpDir, "fake.sql")
	require.NoError(t, os.WriteFile(fakeMigration, []byte(""), 0600), "Setup: couldn't write fake migration file")

	tests := map[string]struct {
		args                  []string
		noDatabase            bool
		preAppliedMigrations  bool
		preGolangMigrateState bool
		preGolangMigrateDirty bool

		wantExitCode int
	}{
		"Basic Migration": {
			args:         []string{"migrate", trueMigrationsDir},
			wantExitCode: 0,
		},
		"Pre-applied Migrations": {
			args:                 []string{"migrate", trueMigrationsDir},
			preAppliedMigrations: true,
			wantExitCode:         0,
		},
		"Bootstrap from golang-migrate": {
			args:                  []string{"migrate", trueMigrationsDir},
			preGolangMigrateState: true,
			wantExitCode:          0,
		},
		"Bootstrap from golang-migrate dirty state": {
			args:                  []string{"migrate", trueMigrationsDir},
			preGolangMigrateDirty: true,
			wantExitCode:          1,
		},

		// Usage Error Cases
		"Missing Path": {
			args:         []string{"migrate"},
			wantExitCode: 2,
		},
		"Extra Arguments": {
			args:         []string{"migrate", trueMigrationsDir, "extra"},
			wantExitCode: 2,
		},
		"Non-existent Path": {
			args:         []string{"migrate", filepath.Join(tmpDir, "non-existent-folder")},
			wantExitCode: 2,
		},
		"Path to File": {
			args:         []string{"migrate", fakeMigration},
			wantExitCode: 2,
		},

		// Error Cases
		"Empty Migrations Directory": {
			args:         []string{"migrate", tmpDir},
			wantExitCode: 1,
		},
		"No Database": {
			args:         []string{"migrate", trueMigrationsDir},
			noDatabase:   true,
			wantExitCode: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := tc.args
			// Start containers
			db := &serverTestUtils.PostgresContainer{}
			if !tc.noDatabase {
				db = serverTestUtils.StartPostgresContainer(t)
				defer func() {
					if err := db.Stop(t.Context()); err != nil {
						t.Errorf("Teardown: failed to stop dbContainer: %v", err)
					}
				}()

				require.NoError(t, db.IsReady(t, 5*time.Second, 10), "Setup: dbContainer was not ready in time")

				if tc.preAppliedMigrations {
					serverTestUtils.ApplyMigrations(t, db.DSN, trueMigrationsDir)
				}
				if tc.preGolangMigrateState {
					seedGolangMigrateState(t, db.DSN, trueMigrationsDir, false)
				}
				if tc.preGolangMigrateDirty {
					seedGolangMigrateState(t, db.DSN, trueMigrationsDir, true)
				}

				args = append(args,
					"--db-host", db.Host,
					"--db-port", db.Port,
					"--db-user", db.User,
					"--db-password", db.Password,
					"--db-name", db.Name,
					"-vv")
			}

			// #nosec:G204 - we control the command arguments in tests
			cmd := exec.CommandContext(t.Context(),
				cliPath,
				args...)

			// Run the command
			out, err := cmd.CombinedOutput()
			if tc.wantExitCode == 0 {
				require.NoError(t, err, "unexpected CLI error: %v\n%s", err, out)

				got := serverTestUtils.DBListTables(t, db.DSN)
				want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
				require.ElementsMatch(t, want, got, "Run should create the expected tables in the database")
			}
			assert.Equal(t, tc.wantExitCode, cmd.ProcessState.ExitCode(), "unexpected exit code: %v\n%s", err, out)
		})
	}
}

// seedGolangMigrateState loads an exact pg_dump of a database previously managed
// by golang-migrate into the test database. The dump was captured from a real
// Postgres instance after golang-migrate v4.19.1 applied all 8 migrations.
//
// If dirty is true, the schema_migrations row is updated to simulate a failed
// migration that left the database in a dirty state.
func seedGolangMigrateState(t *testing.T, dsn string, _ string, dirty bool) {
	t.Helper()

	dumpPath := filepath.Join(serverTestUtils.ModuleRoot(), "internal", "ingest", "migration", "testdata", "golang_migrate_dump.sql")
	dumpSQL, err := os.ReadFile(dumpPath)
	require.NoError(t, err, "Setup: failed to read golang_migrate_dump.sql")

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "Setup: failed to connect to database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()))
	}()

	_, err = conn.Exec(t.Context(), string(dumpSQL))
	require.NoError(t, err, "Setup: failed to apply golang_migrate_dump.sql")

	if dirty {
		_, err = conn.Exec(t.Context(), `UPDATE schema_migrations SET dirty = true`)
		require.NoError(t, err, "Setup: failed to set dirty state")
	}
}
