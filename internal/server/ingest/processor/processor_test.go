package processor_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/models"
	"github.com/ubuntu/ubuntu-insights/internal/server/ingest/processor"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestProcessFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		app string
		db  mockDBManager

		delay         time.Duration
		skipFileCheck bool
		earlyCancel   bool

		wantErr error
	}{
		"Mixed files process as expected": {
			app: "MultiMixed",

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
			app:           "MultiMixed",
			earlyCancel:   true,
			skipFileCheck: true,

			wantErr: context.Canceled,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixtureDir := filepath.Join("..", "testdata", "fixtures", tc.app)
			dst := t.TempDir()
			require.NoError(t, testutils.CopyDir(t, fixtureDir, filepath.Join(dst, tc.app)), "Setup: failed to copy fixture directory")

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(func() {
				cancel()
			})

			if tc.earlyCancel {
				cancel()
			}
			invalidFilesDir := filepath.Join(t.TempDir(), "invalid-files")
			errCh := make(chan error, 1)
			go func() {
				defer close(errCh)
				errCh <- processor.ProcessFiles(ctx, filepath.Join(dst, tc.app), &tc.db, invalidFilesDir)
			}()

			err := <-errCh
			if tc.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tc.skipFileCheck {
				return
			}

			remainingFiles, err := testutils.GetDirHashedContents(t, dst, 4)
			require.NoError(t, err, "Failed to get directory contents")

			invalidFiles, err := testutils.GetDirHashedContents(t, invalidFilesDir, 4)
			require.NoError(t, err, "Failed to get invalid directory contents")

			results := struct {
				RemainingFiles map[string]string
				UploadedFiles  map[string][]*models.TargetModel
				InvalidFiles   map[string]string
			}{
				RemainingFiles: remainingFiles,
				UploadedFiles:  tc.db.data,
				InvalidFiles:   invalidFiles,
			}

			got, err := json.MarshalIndent(results, "", "  ")
			require.NoError(t, err)
			want := testutils.LoadWithUpdateFromGolden(t, string(got))
			assert.Equal(t, strings.ReplaceAll(want, "\r\n", "\n"), string(got), "Unexpected results after processing files")
		})
	}
}

type mockDBManager struct {
	uploadErr error
	data      map[string][]*models.TargetModel // Fake in-memory database
}

func (m *mockDBManager) Upload(ctx context.Context, app string, data *models.TargetModel) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}

	if m.data == nil {
		m.data = make(map[string][]*models.TargetModel)
	}

	// Simulate storing the data in the fake database
	m.data[app] = append(m.data[app], data)
	return nil
}
