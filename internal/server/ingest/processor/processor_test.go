package processor_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/processor"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

var testFixutresDir = filepath.Join("..", "testdata", "fixtures")

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

		"Upload errors do not remove processed files": {
			app: "MultiMixed",
			db: mockDBManager{
				uploadErr: errors.New("requested upload error"),
			},
			delay: 5 * time.Second,
		},

		// Error cases
		"Instant context cancellation errors": {
			app:         "MultiMixed",
			earlyCancel: true,

			wantErr: context.Canceled,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixtureDir := filepath.Join(testFixutresDir, tc.app)
			dst := t.TempDir()
			require.NoError(t, testutils.CopyDir(t, fixtureDir, filepath.Join(dst, tc.app)), "Setup: failed to copy fixture directory")

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(func() {
				cancel()
			})

			if tc.earlyCancel {
				cancel()
			}
			p := processor.New(dst, &tc.db)
			errCh := make(chan error, 1)
			go func() {
				defer close(errCh)
				errCh <- p.Process(ctx, tc.app)
			}()

			var err error
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

			referenceHashes, err := testutils.GetDirHashedContents(t, filepath.Join(testFixutresDir, tc.app), 3)
			require.NoError(t, err, "Failed to get reference directory contents")

			results := struct {
				RemainingFiles      map[string]string
				UploadedFiles       map[string][]*models.TargetModel
				UploadedLegacyFiles map[string][]*models.LegacyTargetModel
				InvalidReports      map[string][]string
				ReferenceHashes     map[string]string
			}{
				RemainingFiles:      remainingFiles,
				UploadedFiles:       tc.db.reports,
				UploadedLegacyFiles: tc.db.legacyReports,
				InvalidReports:      tc.db.invalidReports,
				ReferenceHashes:     referenceHashes,
			}

			got, err := json.MarshalIndent(results, "", "  ")
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGolden(t, string(got))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Unexpected results after processing files")
		})
	}
}

type mockDBManager struct {
	uploadErr      error
	reports        map[string][]*models.TargetModel       // Fake in-memory database
	legacyReports  map[string][]*models.LegacyTargetModel // Fake in-memory legacy reports
	invalidReports map[string][]string                    // Fake in-memory invalid reports
}

func (m *mockDBManager) Upload(ctx context.Context, app string, report *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.reports == nil {
		m.reports = make(map[string][]*models.TargetModel)
	}

	// Simulate storing the data in the fake database
	m.reports[app] = append(m.reports[app], report)
	return nil
}

func (m *mockDBManager) UploadLegacy(ctx context.Context, distribution, version string, report *models.LegacyTargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.legacyReports == nil {
		m.legacyReports = make(map[string][]*models.LegacyTargetModel)
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
