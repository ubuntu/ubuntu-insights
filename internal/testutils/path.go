package testutils

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// CurrentDir returns the current file directory.
func CurrentDir() string {
	// p is the path to the caller file
	_, p, _, _ := runtime.Caller(1)
	return filepath.Dir(p)
}

// ProjectRoot returns the absolute path to the project root.
func ProjectRoot() string {
	// p is the path to the current file, in this case -> {PROJECT_ROOT}/internal/testutils/path.go
	_, p, _, _ := runtime.Caller(0)

	for range 3 {
		// Ignores the last 3 elements -> /internal/testutils/path.go
		p = filepath.Dir(p)
	}

	return p
}

// MakeReadOnly makes dest read only and restore permission on cleanup.
func MakeReadOnly(t *testing.T, dest string) {
	t.Helper()

	// Get current dest permissions
	fi, err := os.Stat(dest)
	require.NoError(t, err, "Cannot stat %s", dest)
	mode := fi.Mode()

	var perms fs.FileMode = 0444
	if fi.IsDir() {
		perms = 0555
	}
	err = os.Chmod(dest, perms)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := os.Stat(dest)
		if errors.Is(err, os.ErrNotExist) {
			return
		}

		err = os.Chmod(dest, mode)
		require.NoError(t, err)
	})
}
