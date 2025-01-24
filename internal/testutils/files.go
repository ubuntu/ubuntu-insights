// TiCS: disabled // Test helpers.

package testutils

import (
	"bytes"
	"fmt"
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
		return CopyFile(path, dstPath)
	})
}

// GetDirContents returns the contents of a directory as a map of file paths to file contents.
// The contents are read as strings.
// The maxDepth parameter limits the depth of the directory tree to read.
func GetDirContents(t *testing.T, dir string, maxDepth uint) (map[string]string, error) {
	t.Helper()

	files := make(map[string]string)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == dir {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		depth := uint(len(filepath.SplitList(relPath)))
		if depth > maxDepth {
			return fmt.Errorf("max depth %d exceeded at %s", maxDepth, relPath)
		}

		if !d.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			// Normalize content between Windows and Linux
			content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
			files[filepath.ToSlash(relPath)] = string(content)
		}

		return nil
	})

	return files, err
}
