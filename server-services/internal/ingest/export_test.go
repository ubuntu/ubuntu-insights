package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/shared/testutils"
)

type (
	DConfigManager = dConfigManager
	DProcessor     = dProcessor
)

// CopyTestFixtures copies test fixtures to a temporary directory for testing.
func CopyTestFixtures(t *testing.T, removeFiles []string) string {
	t.Helper()

	dst := filepath.Join(t.TempDir(), "fixtures")
	err := testutils.CopyDir(t, filepath.Join("testdata", "fixtures"), dst)
	require.NoError(t, err, "Setup: failed to copy test fixtures")

	for _, file := range removeFiles {
		err = os.Remove(filepath.Join(dst, file))
		require.NoError(t, err, "Setup: failed to remove file %s", file)
	}

	return dst
}

// WorkerNames returns the app names of active workers.
func (s *Service) WorkerNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	names := make([]string, 0, len(s.workers))
	for name := range s.workers {
		names = append(names, name)
	}
	return names
}
