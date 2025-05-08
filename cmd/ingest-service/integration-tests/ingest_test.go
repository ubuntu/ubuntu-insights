package ingest_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/fileutils"
	"github.com/ubuntu/ubuntu-insights/internal/server/shared/config"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestIngestService(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "linux" {
		t.Skip("Skipping test on non-linux OS")
	}

	type reports struct {
		app        string
		reportType report
		count      int
		delayAfter int
	}

	tests := map[string]struct {
		validApps   []string
		preReports  []reports // Reports to be created before starting the daemon
		postReports []reports // Reports to be created after starting the daemon
	}{
		"Prexisting reports only": {
			validApps: []string{"linux", "windows"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2},
				{app: "linux", reportType: empty, count: 1},
			},
		},
		"Prexisting and new reports": {
			validApps: []string{"linux", "windows"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2},
				{app: "linux", reportType: empty, count: 1},
			},
			postReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "linux", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2, delayAfter: 2},
				{app: "linux", reportType: empty, count: 1},
			},
		},
		"New reports only": {
			validApps: []string{"linux", "windows"},
			postReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2},
				{app: "linux", reportType: empty, count: 1, delayAfter: 2},
			},
		},
		"Only linux valid": {
			validApps: []string{"linux"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2},
				{app: "linux", reportType: empty, count: 1, delayAfter: 2},
			},
			postReports: []reports{
				{app: "windows", reportType: validV1, count: 5},
				{app: "linux", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2, delayAfter: 2},
			},
		},
		"All valid apps": {
			validApps: []string{"linux", "windows", "darwin"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "darwin", reportType: validOptOut, count: 3},
			},
			postReports: []reports{
				{app: "linux", reportType: validOptOut, count: 2},
				{app: "windows", reportType: validV1, count: 3},
				{app: "darwin", reportType: validV1, count: 3},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Start containers
			dbContainer := StartPostgresContainer(t)
			defer func() {
				if err := dbContainer.Stop(t.Context()); err != nil {
					t.Errorf("Teardown: failed to stop dbContainer: %v", err)
				}
			}()
			dbLogs, err := dbContainer.Container.Logs(t.Context())
			require.NoError(t, err, "Setup: failed to get dbContainer logs")
			go func() {
				scanner := bufio.NewScanner(dbLogs)
				for scanner.Scan() {
					t.Logf("dbContainer logs: %s", scanner.Text())
				}
			}()

			require.NoError(t, dbContainer.IsReady(t, 5*time.Second, 10), "Setup: dbContainer was not ready in time")
			ApplyMigrations(t, dbContainer.DSN, filepath.Join(testutils.ProjectRoot(), "migrations"))

			dst := t.TempDir()
			for _, report := range tc.preReports {
				makeReport(t, report.reportType, report.count, filepath.Join(dst, report.app), false)
			}

			daeConf := &config.Conf{
				BaseDir:     dst,
				AllowedList: tc.validApps,
			}
			configPath := generateTestDaemonConfig(t, daeConf)

			ctx, cancel := context.WithCancel(t.Context())
			// #nosec:G204 - we control the command arguments in tests
			go func() {
				r, w := io.Pipe()
				cmd := exec.CommandContext(ctx,
					cliPath,
					"--daemon-config", configPath,
					"--db-host", dbContainer.Host,
					"--db-port", dbContainer.Port,
					"--db-user", dbContainer.User,
					"--db-password", dbContainer.Password,
					"--db-name", dbContainer.Name,
					"-vv")

				// Redirect command output to the pipe
				cmd.Stdout = w
				cmd.Stderr = w

				// Log the output in real-time
				go func() {
					scanner := bufio.NewScanner(r)
					for scanner.Scan() {
						t.Logf("CLI Output: %s", scanner.Text())
					}
				}()

				// Run the command
				if err := cmd.Run(); err != nil {
					// Ignored killed error
					if ctx.Err() == context.Canceled {
						return
					}
					t.Errorf("unexpected CLI error: %v", err)
				}

				// Close the writer to signal the end of output
				_ = w.Close()
			}()

			// Allow it to run for a while
			time.Sleep(2 * time.Second)

			for _, report := range tc.postReports {
				makeReport(t, report.reportType, report.count, filepath.Join(dst, report.app), true)
				if report.delayAfter > 0 {
					time.Sleep(time.Duration(report.delayAfter) * time.Second)
				}
			}
			time.Sleep(5 * time.Second)
			// Send signal to stop the daemon
			cancel()

			// Check the dirContents of data directory
			dirContents, err := testutils.GetDirContents(t, dst, 3)
			require.NoError(t, err, "failed to get directory contents")
			// Remove the filenames from the map
			remainingFiles := make(map[string][]string)
			for path, content := range dirContents {
				// Parse path to get the app name
				appName := filepath.Base(filepath.Dir(path))
				if _, ok := remainingFiles[appName]; !ok {
					remainingFiles[appName] = make([]string, 0)
				}
				remainingFiles[appName] = append(remainingFiles[appName], content)
			}

			// Sort the content lists for consistency
			for appName, content := range remainingFiles {
				sort.Strings(content)
				remainingFiles[appName] = content
			}

			// Check the database for opt-out counts
			type reportCount struct {
				TotalReports  int
				OptOutReports int
				OptInReports  int
			}

			reportsCounts := make(map[string]reportCount)
			for _, app := range listTables(t, dbContainer.DSN) {
				totalReports, optOutReports, optInReports := checkOptOutCounts(t, dbContainer.DSN, app)
				reportsCounts[app] = reportCount{
					TotalReports:  totalReports,
					OptOutReports: optOutReports,
					OptInReports:  optInReports,
				}
			}

			results := struct {
				RemainingFiles map[string][]string
				ReportsCount   map[string]reportCount
			}{
				RemainingFiles: remainingFiles,
				ReportsCount:   reportsCounts,
			}

			got, err := json.MarshalIndent(results, "", "  ")
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGolden(t, string(got))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Unexpected results after processing files")
		})
	}
}

// generateTestDaemonConfig generates a temporary daemon config file for testing.
func generateTestDaemonConfig(t *testing.T, daeConf *config.Conf) string {
	t.Helper()

	d, err := json.Marshal(daeConf)
	require.NoError(t, err, "Setup: failed to marshal dynamic server config for tests")
	daeConfPath := filepath.Join(t.TempDir(), "daemon-testconfig.yaml")
	require.NoError(t, os.WriteFile(daeConfPath, d, 0600), "Setup: failed to write dynamic config for tests")

	return daeConfPath
}

func listTables(t *testing.T, dsn string) []string {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "failed to connect to the database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()), "failed to close the database connection")
	}()

	query := `
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_type = 'BASE TABLE'
          AND table_name NOT IN ('schema_migrations');`

	rows, err := conn.Query(t.Context(), query)
	require.NoError(t, err, "failed to execute query")

	var tables []string
	for rows.Next() {
		var tableName string
		require.NoError(t, rows.Scan(&tableName), "failed to scan table name")
		tables = append(tables, tableName)
	}

	require.NoError(t, rows.Err(), "error occurred during rows iteration")
	return tables
}

func checkOptOutCounts(t *testing.T, dsn string, tableName string) (totalReports, optOutReports, optInReports int) {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "failed to connect to the database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()), "failed to close the database connection")
	}()

	// Query to count total reports, opt-out reports, and opt-in reports
	query := `
        SELECT
            COUNT(*) AS total_reports,
            COUNT(CASE WHEN optout = true THEN 1 END) AS opt_out_reports,
            COUNT(CASE WHEN optout = false THEN 1 END) AS opt_in_reports
        FROM ` + tableName

	err = conn.QueryRow(t.Context(), query).Scan(&totalReports, &optOutReports, &optInReports)
	require.NoError(t, err, "failed to execute query")

	return totalReports, optOutReports, optInReports
}

type report int

const (
	empty report = iota
	validV1
	validOptOut
	invalidOptOut
)

func makeReport(t *testing.T, reportType report, count int, reportDir string, atomicWrite bool) {
	t.Helper()

	require.NoError(t, os.MkdirAll(reportDir, 0750), "Setup: failed to create report directory")
	rep := ""
	switch reportType {
	case empty:
		rep = `{}`
	case validV1:
		rep = `
{
    "insightsVersion": "0.0.1~ppa5",
    "systemInfo": {
        "hardware": {
            "product": {
                "family": "My Product Family",
                "name": "My Product Name",
                "vendor": "My Product Vendor"
            },
            "cpu": {
                "name": "9 1200SX",
                "vendor": "Authentic",
                "architecture": "x86_64",
                "cpus": 16,
                "sockets": 1,
                "coresPerSocket": 8,
                "threadsPerCore": 2
            },
            "gpus": [
                {
                    "device": "0x0294",
                    "vendor": "0x10df",
                    "driver": "gpu"
                },
                {
                    "device": "0x03ec",
                    "vendor": "0x1003",
                    "driver": "gpu"
                }
            ],
            "memory": {
                "size": 23247
            },
            "disks": [
                {
                    "size": 1887436,
                    "type": "disk",
                    "children": [
                        {
                            "size": 750,
                            "type": "part"
                        },
                        {
                            "size": 260,
                            "type": "part"
                        },
                        {
                            "size": 16,
                            "type": "part"
                        },
                        {
                            "size": 1887436,
                            "type": "part"
                        },
                        {
                            "size": 869,
                            "type": "part"
                        },
                        {
                            "size": 54988,
                            "type": "part"
                        }
                    ]
                }
            ],
            "screens": [
                {
                    "size": "600mm x 340mm",
                    "resolution": "2560x1440",
                    "refreshRate": "143.83"
                },
                {
                    "size": "300mm x 190mm",
                    "resolution": "1704x1065",
                    "refreshRate": "119.91"
                }
            ]
        },
        "software": {
            "os": {
                "family": "linux",
                "distribution": "Ubuntu",
                "version": "24.04"
            },
            "timezone": "EDT",
            "language": "en_US",
            "bios": {
                "vendor": "Bios Vendor",
                "version": "Bios Version"
            }
        },
        "platform": {
            "desktop": {
                "desktopEnvironment": "ubuntu:GNOME",
                "sessionName": "ubuntu",
                "sessionType": "wayland"
            },
            "proAttached": true
        }
    }
}`
	case validOptOut:
		rep = `
{
    "OptOut": true
}`
	case invalidOptOut:
		rep = `
{
    "OptOut": true,
    "insightsVersion": "0.0.1~ppa5",
    "systemInfo": {
        "hardware": {
            "product": {
                "family": "My Product Family",
                "name": "My Product Name",
                "vendor": "My Product Vendor"
            },
            "cpu": {
                "name": "9 1200SX",
                "vendor": "Authentic",
                "architecture": "x86_64",
                "cpus": 16,
                "sockets": 1,
                "coresPerSocket": 8,
                "threadsPerCore": 2
            },
            "gpus": [
                {
                    "device": "0x0294",
                    "vendor": "0x10df",
                    "driver": "gpu"
                },
                {
                    "device": "0x03ec",
                    "vendor": "0x1003",
                    "driver": "gpu"
                }
            ],
            "memory": {
                "size": 23247
            },
            "disks": [
                {
                    "size": 1887436,
                    "type": "disk",
                    "children": [
                        {
                            "size": 750,
                            "type": "part"
                        },
                        {
                            "size": 260,
                            "type": "part"
                        },
                        {
                            "size": 16,
                            "type": "part"
                        },
                        {
                            "size": 1887436,
                            "type": "part"
                        },
                        {
                            "size": 869,
                            "type": "part"
                        },
                        {
                            "size": 54988,
                            "type": "part"
                        }
                    ]
                }
            ],
            "screens": [
                {
                    "size": "600mm x 340mm",
                    "resolution": "2560x1440",
                    "refreshRate": "143.83"
                },
                {
                    "size": "300mm x 190mm",
                    "resolution": "1704x1065",
                    "refreshRate": "119.91"
                }
            ]
        },
        "software": {
            "os": {
                "family": "linux",
                "distribution": "Ubuntu",
                "version": "24.04"
            },
            "timezone": "EDT",
            "language": "en_US",
            "bios": {
                "vendor": "Bios Vendor",
                "version": "Bios Version"
            }
        },
        "platform": {
            "desktop": {
                "desktopEnvironment": "ubuntu:GNOME",
                "sessionName": "ubuntu",
                "sessionType": "wayland"
            },
            "proAttached": true
        }
    }
}`
	}

	// Write the report to a file, with uuid name and .json extension
	for range count {
		uuid := uuid.New()
		fileName := uuid.String() + ".json"
		filePath := filepath.Join(reportDir, fileName)

		if atomicWrite {
			require.NoError(t, fileutils.AtomicWrite(filePath, []byte(rep)), "Setup: failed to write report file")
		} else {
			require.NoError(t, os.WriteFile(filePath, []byte(rep), 0600), "Setup: failed to write report file")
		}
	}
}
