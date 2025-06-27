// TiCS: disabled // Test helpers.

package testutils

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ubuntu/ubuntu-insights/common/fileutils"
)

// CopyFile copies a file from source to destination.
func CopyFile(t *testing.T, src, dst string) error {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return fileutils.AtomicWrite(dst, data)
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

// GetDirHashedContents is like GetDirContents but hashes the contents of the files.
//
// This is for situations where it is very unlikely that the contents of the files will change,
// and we don't care about not being able to see the actual diff, and we can identify the contents in other ways,
// such as by the filename. For those situations, hashing the contents can make the golden files much more readable.
func GetDirHashedContents(t *testing.T, dir string, maxDepth uint) (map[string]string, error) {
	t.Helper()

	dirContents, err := GetDirContents(t, dir, maxDepth)
	if err != nil {
		return nil, err
	}

	for k, v := range dirContents {
		dirContents[k] = fmt.Sprint(HashString(v))
	}

	return dirContents, nil
}

// HashString returns the crc32 checksum of a string, removing any Windows line endings before hashing.
func HashString(s string) uint32 {
	s = strings.ReplaceAll(s, "\r\n", "\n") // Normalize line endings
	return crc32.ChecksumIEEE([]byte(s))
}
