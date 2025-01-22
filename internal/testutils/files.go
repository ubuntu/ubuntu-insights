// TiCS: disabled // Test helpers.

package testutils

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// CleanupDir removes the temporary directory including its contents.
func CleanupDir(t *testing.T, dir string) {
	t.Helper()
	assert.NoError(t, os.RemoveAll(dir), "Cleanup: failed to remove temporary directory")
}

// CopyFile copies a file from source to destination.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

// CopySymlink copies a symlink from source to destination.
func CopySymlink(src, dst string) error {
	lnk, err := os.Readlink(src)
	if err != nil {
		return err
	}

	err = os.Symlink(lnk, dst)
	if err != nil {
		return err
	}
	return nil
}

// CopyDir copies the contents of a directory to another directory.
func CopyDir(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		if info.Mode()&fs.ModeSymlink > 0 {
			return CopySymlink(path, dstPath)
		}
		return CopyFile(path, dstPath)
	})
}
