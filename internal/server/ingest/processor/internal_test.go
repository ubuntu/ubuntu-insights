// These tests are for unit testing of specific edge cases.
package processor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostProcessingMissingInvalidDir(t *testing.T) {
	t.Parallel()

	// Write temporary test file
	testFile := t.TempDir() + "/test.json"
	err := os.WriteFile(testFile, []byte(`{"foo": "bar"}`), 0600)
	require.NoError(t, err, "Setup: failed to write test file")

	// Ensure there isn't a panic!
	postProcessing(testFile, errors.New("Some error"), filepath.Join(t.TempDir(), "invalid"))

	// Assert that testFile doesn't exist
	_, err = os.Stat(testFile)
	require.True(t, os.IsNotExist(err), "Test: expected test file to be deleted")
}

func TestPostProcessingMissingFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err error
	}{
		"No error": {
			err: nil,
		},
		"Some error": {
			err: errors.New("Some error"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary directory
			tempDir := t.TempDir()

			// Write a test file to the temporary directory
			testFile := filepath.Join(tempDir, "test.json")

			// Ensure there isn't a panic!
			postProcessing(testFile, tc.err, tempDir)

			_, err := os.Stat(testFile)
			require.True(t, os.IsNotExist(err), "Test: expected test file to be deleted")
		})
	}
}
