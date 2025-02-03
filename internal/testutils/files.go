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
)

// CopyFile copies a file from source to destination.
func CopyFile(t *testing.T, src, dst string) error {
	t.Helper()

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
func CopySymlink(t *testing.T, src, dst string) error {
	t.Helper()

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
func CopyDir(t *testing.T, srcDir, dstDir string) error {
	t.Helper()
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
			return os.MkdirAll(dstPath, 0700)
		}
		if info.Mode()&fs.ModeSymlink > 0 {
			return CopySymlink(t, path, dstPath)
		}
		return CopyFile(t, path, dstPath)
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
