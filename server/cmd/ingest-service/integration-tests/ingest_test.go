package ingest_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/config"
	ingestTestUtils "github.com/ubuntu/ubuntu-insights/server/internal/ingest/testutils"
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
		"Preexisting reports only": {
			validApps: []string{"linux", "windows"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 5},
				{app: "windows", reportType: validOptOut, count: 3},
				{app: "linux", reportType: invalidOptOut, count: 2},
				{app: "linux", reportType: empty, count: 1},
			},
		},
		"Preexisting and new reports": {
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
		"Legacy ubuntu report apps": {
			validApps: []string{"linux", "windows", "darwin", "ubuntu-report/distribution/desktop/version"},
			preReports: []reports{
				{app: "ubuntu-report/distribution/desktop/version", reportType: ubuntuReport, count: 3},
				{app: "ubuntu-report/distribution/desktop/version", reportType: validOptOut, count: 3},
				{app: "ubuntu-report/distribution/desktop/version", reportType: invalidOptOut, count: 2},
				{app: "ubuntu-report/distribution/desktop/version", reportType: empty, count: 1},
				{app: "ubuntu-report/distribution/desktop/bad-version", reportType: ubuntuReport, count: 1},
			},
			postReports: []reports{
				{app: "ubuntu-report/distribution/desktop/unknown-version", reportType: ubuntuReport, count: 1},
				{app: "ubuntu-report/unknown-distribution/desktop/version", reportType: ubuntuReport, count: 1},
				{app: "ubuntu-report/distribution/desktop/version", reportType: ubuntuReport, count: 2},
				{app: "ubuntu-report/distribution/desktop/version", reportType: validOptOut, count: 1},
				{app: "ubuntu-report/distribution/desktop/version", reportType: validV1, count: 1, delayAfter: 2},
			},
		},
		"Reports with unexpected fields": {
			validApps: []string{"linux", "windows", "darwin", "ubuntu-report/distribution/desktop/version"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 1},
				{app: "linux", reportType: invalidV1ExtraRoot, count: 1},
				{app: "linux", reportType: invalidV1ExtraSysInfo, count: 1},
				{app: "linux", reportType: invalidV1ExtraFields, count: 1},
				{app: "ubuntu-report/distribution/desktop/version", reportType: invalidOptOut, count: 1},
				{app: "windows", reportType: invalidOptOut, count: 1},
			},
		},
		"Reports with invalid JSON": {
			validApps: []string{"linux", "windows", "darwin", "ubuntu-report/distribution/desktop/version"},
			preReports: []reports{
				{app: "linux", reportType: validV1, count: 1},
				{app: "linux", reportType: invalidJSON, count: 1},
				{app: "ubuntu-report/distribution/desktop/version", reportType: invalidJSON, count: 1},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Start containers
			dbContainer := ingestTestUtils.StartPostgresContainer(t)
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
			ingestTestUtils.ApplyMigrations(t, dbContainer.DSN, filepath.Join(testutils.ProjectRoot(), "server", "migrations"))

			dst := t.TempDir()
			for _, report := range tc.preReports {
				makeReport(t, report.reportType, report.count, filepath.Join(dst, report.app), false)
			}

			daeConf := &config.Conf{
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
					"--reports-dir", dst,
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

			// Process remaining files
			remainingFiles := processDirectoryContents(dirContents)

			// Check the database for opt-out counts
			type reportCount struct {
				TotalReports  int
				OptOutReports int
				OptInReports  int
			}

			reportsCounts := make(map[string]reportCount)
			for _, app := range ingestTestUtils.DBListTables(t, dbContainer.DSN, "schema_migrations", "invalid_reports") {
				totalReports, optOutReports, optInReports := checkOptOutCounts(t, dbContainer.DSN, app)
				reportsCounts[app] = reportCount{
					TotalReports:  totalReports,
					OptOutReports: optOutReports,
					OptInReports:  optInReports,
				}

				fields := []string{"insights_version", "collection_time", "hardware", "software", "platform", "source_metrics"}
				if app == "ubuntu_report" {
					fields = []string{"report"}
				}
				validateOptOutEntries(t, dbContainer.DSN, app, fields...)
			}

			invalidReports := queryInvalidReports(t, dbContainer.DSN)

			results := struct {
				RemainingFiles map[string][]string
				ReportsCount   map[string]reportCount
				InvalidReports []invalidReportEntry
			}{
				RemainingFiles: remainingFiles,
				ReportsCount:   reportsCounts,
				InvalidReports: invalidReports,
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

// validateOptOutEntries checks that the opt-out field consistency is maintained in the specified table.
// For opt-out reports all fields listed should be NULL, while for opt-in reports,
// at least one of the fields should be NOT NULL.
func validateOptOutEntries(t *testing.T, dsn, tableName string, fields ...string) {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "failed to connect to the database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()), "failed to close the database connection")
	}()

	// Build SQL for checking violations
	// For optout=true: any field is NOT NULL
	optOutChecks := make([]string, len(fields))
	// For optout=false: all fields are NULL
	optInChecks := make([]string, len(fields))
	for i, f := range fields {
		optOutChecks[i] = f + " IS NOT NULL"
		optInChecks[i] = f + " IS NULL"
	}

	optOutQuery := `
        SELECT COUNT(*) FROM ` + tableName + `
        WHERE optout = true AND (` + strings.Join(optOutChecks, " OR ") + `)
    `
	optInQuery := `
        SELECT COUNT(*) FROM ` + tableName + `
        WHERE optout = false AND (` + strings.Join(optInChecks, " AND ") + `)
    `

	var optOutViolations, optInViolations int
	err = conn.QueryRow(t.Context(), optOutQuery).Scan(&optOutViolations)
	require.NoError(t, err, "failed to execute opt-out query")
	assert.Equal(t, 0, optOutViolations, "Opt-out reports should not have any consistency violations")

	err = conn.QueryRow(t.Context(), optInQuery).Scan(&optInViolations)
	require.NoError(t, err, "failed to execute opt-in query")
	assert.Equal(t, 0, optInViolations, "Opt-in reports should not have any consistency violations")
}

type invalidReportEntry struct {
	AppName   string
	RawReport string
}

// getInvalidReports queries the invalid_reports table and returns a sorted list of entries
// including the app_name and a hash of raw_report.
func queryInvalidReports(t *testing.T, dsn string) []invalidReportEntry {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err, "failed to connect to the database")
	defer func() {
		require.NoError(t, conn.Close(t.Context()), "failed to close the database connection")
	}()

	query := `
		SELECT app_name, raw_report
		FROM invalid_reports
		ORDER BY app_name, raw_report;
	`
	rows, err := conn.Query(t.Context(), query)
	require.NoError(t, err, "failed to execute query")

	var entries []invalidReportEntry
	for rows.Next() {
		var appName, rawReport string
		require.NoError(t, rows.Scan(&appName, &rawReport), "failed to scan row")
		entries = append(entries, invalidReportEntry{
			AppName:   appName,
			RawReport: fmt.Sprint(testutils.HashString(rawReport)),
		})
	}
	require.NoError(t, rows.Err(), "error occurred during rows iteration")

	return entries
}

type report int

const (
	empty report = iota
	validV1
	validOptOut
	invalidJSON
	invalidV1ExtraRoot
	invalidV1ExtraSysInfo
	invalidV1ExtraFields
	invalidOptOut
	ubuntuReport
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
	"collectionTime": 1747752692,
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
	case invalidJSON:
		rep = `{
this is invalid JSON`
	case invalidV1ExtraRoot:
		rep = `
		{
			"insightsVersion": "0.0.1~ppa5",
			"collectionTime": 1747752692,
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
			},
			"extraRoot": "This is an extra root field that should not be here"
		}`
	case invalidV1ExtraSysInfo:
		rep = `
		{
			"insightsVersion": "0.0.1~ppa5",
			"collectionTime": 1747752692,
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
				},
				"extraSysInfo": "This is an extra sysInfo field that should not be here"
			}
		}`
	case invalidV1ExtraFields:
		rep = `
		{
			"insightsVersion": "0.0.1~ppa5",
			"collectionTime": 1747752692,
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
				},
				"extraField1": "This is an extra field that should not be here",
			},
			"extraField2": "This is another extra field that should not be here"
		}`
	case invalidOptOut:
		rep = `
{
    "OptOut": true,
    "insightsVersion": "0.0.1~ppa5",
	"collectionTime": 1747752692,
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
	case ubuntuReport:
		rep = `
{
  "Version": "18.04",
  "OEM": {
    "Vendor": "Vendor Name",
    "Product": "4287CTO"
  },
  "BIOS": {
    "Vendor": "Vendor Name",
    "Version": "8DET52WW (1.27)"
  },
  "CPU": {
    "OpMode": "32-bit, 64-bit",
    "CPUs": "8",
    "Threads": "2",
    "Cores": "4",
    "Sockets": "1",
    "Vendor": "Genuine",
    "Family": "6",
    "Model": "158",
    "Stepping": "10",
    "Name": "Intius Corus i5-8300H CPU @ 2.30GHz",
    "Virtualization": "VT-x"
  },
  "Arch": "amd64",
  "GPU": [
    {
      "Vendor": "8086",
      "Model": "0126"
    }
  ],
  "RAM": 8,
  "Disks": [
    240.1,
    500.1
  ],
  "Partitions": [
    229.2,
    479.7
  ],
  "Screens": [
    {
      "Size": "277mmx156mm",
      "Resolution": "1366x768",
      "Frequency": "60.02"
    },
    {
      "Resolution": "1920x1080",
      "Frequency": "60.00"
    }
  ],
  "Autologin": false,
  "LivePatch": true,
  "Session": {
    "DE": "ubuntu:GNOME",
    "Name": "ubuntu",
    "Type": "x11"
  },
  "Language": "fr_FR",
  "Timezone": "Europe/Paris",
  "Install": {
    "Media": "Ubuntu 18.04 LTS \"Bionic Beaver\" - Alpha amd64 (20180305)",
    "Type": "GTK",
    "PartitionMethod": "use_device",
    "DownloadUpdates": true,
    "Language": "fr",
    "Minimal": false,
    "RestrictedAddons": false,
    "Stages": {
      "0": "language",
      "3": "language",
      "10": "console_setup",
      "15": "prepare",
      "25": "partman",
      "27": "start_install",
      "37": "timezone",
      "49": "usersetup",
      "57": "user_done",
      "829": "done"
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

// processDirectoryContents takes directory contents and returns a map of app names to sorted content.
func processDirectoryContents(dirContents map[string]string) map[string][]string {
	processedContents := make(map[string][]string)
	for path, content := range dirContents {
		// Parse path to get the app name
		appName := filepath.Base(filepath.Dir(path))
		if _, ok := processedContents[appName]; !ok {
			processedContents[appName] = make([]string, 0)
		}
		processedContents[appName] = append(processedContents[appName], content)
	}

	// Sort the content lists for consistency
	for appName, content := range processedContents {
		sort.Strings(content)
		processedContents[appName] = content
	}
	return processedContents
}
