package processor_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/fileutils"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/server/internal/common/constants"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/models"
	"github.com/ubuntu/ubuntu-insights/server/internal/ingest/processor"
)

var testFixturesDir = filepath.Join("testdata", "fixtures")

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		baseDir                 string
		preRegisteredCollectors []prometheus.Collector
		wantErr                 bool
	}{
		"Valid base directory": {
			baseDir: t.TempDir(),
		},
		"Valid non-existent base directory": {
			baseDir: filepath.Join(t.TempDir(), "non-existent"),
		},
		"Non-empty registry": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "test_counter",
					},
					[]string{"label"},
				),
			},
		},

		// Error cases
		"Empty base directory": {
			baseDir: "",
			wantErr: true,
		},
		"Invalid base directory": {
			baseDir: string([]byte{0}),
			wantErr: true,
		},
		"ingest_processor_files_processed_total already registered": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "ingest_processor_files_processed_total",
					},
					[]string{"app", "result"},
				),
			},
			wantErr: true,
		},
		"ingest_processor_process_duration_seconds already registered": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewHistogramVec(
					prometheus.HistogramOpts{
						Name: "ingest_processor_process_duration_seconds",
					},
					[]string{"app"},
				),
			},
			wantErr: true,
		},
		"ingest_processor_cache_size already registered": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "ingest_processor_cache_size",
					},
					[]string{"app"},
				),
			},
			wantErr: true,
		},
		"ingest_processor_cache_size_bytes already registered": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "ingest_processor_cache_size_bytes",
					},
					[]string{"app"},
				),
			},
			wantErr: true,
		},
		"ingest_processor_errors_total already registered": {
			baseDir: t.TempDir(),
			preRegisteredCollectors: []prometheus.Collector{
				prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: "ingest_processor_errors_total",
					},
					[]string{"app"},
				),
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			registry := prometheus.NewRegistry()
			for _, collector := range tc.preRegisteredCollectors {
				require.NoError(t, registry.Register(collector), "Setup: Failed to register pre-existing collector")
			}

			p, err := processor.New(tc.baseDir, nil, registry)

			if tc.wantErr {
				require.Error(t, err, "Expected error for test case: %s", name)
				return
			}
			require.NoError(t, err, "Unexpected error for test case: %s", name)
			require.NotNil(t, p, "Processor should not be nil for test case: %s", name)
		})
	}
}

func TestProcessFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		app string
		db  mockDBManager

		delay       time.Duration
		earlyCancel bool

		wantErr error
	}{
		"Mixed files process as expected": {
			app: "MultiMixed",

			delay: 5 * time.Second,
		},

		"Legacy ubuntu report files process as expected": {
			app:   "ubuntu-report/distribution/desktop/version",
			delay: 5 * time.Second,
		},

		// Error cases
		"Upload errors do not remove processed files": {
			app: "MultiMixed",
			db: mockDBManager{
				uploadErr: errors.New("requested upload error"),
			},
			delay: 5 * time.Second,

			wantErr: processor.ErrDatabaseErrors,
		},

		"Instant context cancellation errors": {
			app:         "MultiMixed",
			earlyCancel: true,

			wantErr: context.Canceled,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixtureDir := filepath.Join(testFixturesDir, tc.app)
			dst := t.TempDir()
			require.NoError(t, testutils.CopyDir(t, fixtureDir, filepath.Join(dst, tc.app)), "Setup: failed to copy fixture directory")

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(func() {
				cancel()
			})

			if tc.earlyCancel {
				cancel()
			}
			registry := prometheus.NewRegistry()
			p, err := processor.New(dst, &tc.db, registry)
			require.NoError(t, err, "Setup: Failed to create processor")
			errCh := make(chan error, 1)
			go func() {
				defer close(errCh)
				errCh <- p.Process(ctx, tc.app)
			}()

			select {
			case err = <-errCh:
			case <-time.After(45 * time.Second):
				require.Fail(t, "Test timed out waiting for processing to finish")
			}

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			remainingFiles, err := testutils.GetDirHashedContents(t, dst, 4)
			require.NoError(t, err, "Failed to get directory contents")

			referenceHashes, err := testutils.GetDirHashedContents(t, filepath.Join(testFixturesDir, tc.app), 3)
			require.NoError(t, err, "Failed to get reference directory contents")

			// Ensure "ingest_processor_process_duration_seconds" is registered
			assert.NotEqual(t, 0, testutil.CollectAndCount(registry, "ingest_processor_process_duration_seconds"),
				"Expected 'ingest_processor_process_duration_seconds' metric to be registered")

			// Don't check "ingest_processor_process_duration_seconds" as it may vary
			metrics, err := testutil.CollectAndFormat(registry, expfmt.TypeTextPlain,
				"ingest_processor_files_processed_total",
				"ingest_processor_cache_size",
				"ingest_processor_cache_size_bytes",
				"ingest_processor_errors_total")
			require.NoError(t, err, "Failed to gather metrics")

			results := struct {
				RemainingFiles      map[string]string
				UploadedFiles       map[string][]*models.TargetModel
				UploadedLegacyFiles map[string][]*models.LegacyTargetModel
				InvalidReports      map[string][]string
				ReferenceHashes     map[string]string
				Metrics             string
			}{
				RemainingFiles:      remainingFiles,
				UploadedFiles:       tc.db.reports,
				UploadedLegacyFiles: tc.db.legacyReports,
				InvalidReports:      tc.db.invalidReports,
				ReferenceHashes:     referenceHashes,
				Metrics:             string(metrics),
			}

			got, err := json.MarshalIndent(results, "", "  ")
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGolden(t, string(got))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Unexpected results after processing files")
		})
	}
}

func BenchmarkProcessFiles(b *testing.B) {
	dir := b.TempDir()
	appDir := filepath.Join(dir, "Benchmark")

	// Suppress logs for cleaner benchmark output.
	slog.SetLogLoggerLevel(slog.LevelError)
	require.NoError(b, os.Mkdir(appDir, 0o700), "Setup: Failed to create app directory")

	db := &mockDBManager{}
	proc, err := processor.New(dir, db, prometheus.NewRegistry())
	require.NoError(b, err, "Setup: Failed to create processor")

	fixtures := filepath.Join(testFixturesDir, "MultiMixed")

	optOutReport, err := os.ReadFile(filepath.Join(fixtures, "optout.json"))
	require.NoError(b, err, "Setup: Failed to read opt-out report fixture")

	validReport, err := os.ReadFile(filepath.Join(fixtures, "valid_1.json"))
	require.NoError(b, err, "Setup: Failed to read valid report fixture")

	invalidReport, err := os.ReadFile(filepath.Join(fixtures, "invalid_1.json"))
	require.NoError(b, err, "Setup: Failed to read invalid report fixture")

	for b.Loop() {
		b.StopTimer()
		entries, err := os.ReadDir(appDir)
		require.NoError(b, err, "Setup: Failed to read app directory")
		require.Empty(b, entries, "Setup: App directory is not empty at start of benchmark loop")

		for range 2000 {
			// 6000 entries per loop.
			require.NoError(b, fileutils.AtomicWrite(filepath.Join(appDir, fmt.Sprintf("%s.json", uuid.NewString())), validReport), "Setup: Failed to write valid report")
			require.NoError(b, fileutils.AtomicWrite(filepath.Join(appDir, fmt.Sprintf("%s.json", uuid.NewString())), invalidReport), "Setup: Failed to write invalid report")
			require.NoError(b, fileutils.AtomicWrite(filepath.Join(appDir, fmt.Sprintf("%s.json", uuid.NewString())), optOutReport), "Setup: Failed to write opt-out report")
		}
		b.StartTimer()

		err = proc.Process(context.Background(), "Benchmark")
		require.NoError(b, err, "Processing failed")
	}
}

type mockDBManager struct {
	uploadErr      error
	reports        map[string][]*models.TargetModel       // Fake in-memory database
	legacyReports  map[string][]*models.LegacyTargetModel // Fake in-memory legacy reports
	invalidReports map[string][]string                    // Fake in-memory invalid reports
}

func (m *mockDBManager) Upload(ctx context.Context, id, app string, report *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.reports == nil {
		m.reports = make(map[string][]*models.TargetModel)
	}

	if err := uuid.Validate(id); err != nil {
		return errors.New("invalid UUID provided")
	}

	// Simulate storing the data in the fake database
	m.reports[app] = append(m.reports[app], report)
	return nil
}

func (m *mockDBManager) UploadLegacy(ctx context.Context, id, distribution, version string, report *models.LegacyTargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.legacyReports == nil {
		m.legacyReports = make(map[string][]*models.LegacyTargetModel)
	}

	if err := uuid.Validate(id); err != nil {
		return errors.New("invalid UUID provided")
	}

	// Simulate storing the legacy report in the fake database
	key := constants.LegacyReportTag + "/" + distribution + "/desktop/" + version
	m.legacyReports[key] = append(m.legacyReports[key], report)
	return nil
}

func (m *mockDBManager) UploadInvalid(ctx context.Context, id, app, rawReport string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.invalidReports == nil {
		m.invalidReports = make(map[string][]string)
	}

	if err := uuid.Validate(id); err != nil {
		return errors.New("invalid UUID provided")
	}

	m.invalidReports[app] = append(m.invalidReports[app], fmt.Sprint(testutils.HashString(rawReport)))
	return nil
}
