// Package fileutils provides utility functions for handling files.
package fileutils

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to a file atomically.
// If the file already exists, then it will be overwritten.
// Not atomic on Windows.
func AtomicWrite(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "tmp-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		if err := os.Remove(tmp.Name()); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to remove temporary file", "file", tmp.Name(), "error", err)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("could not write to temporary file: %v", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary file: %v", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("could not rename temporary file: %v", err)
	}
	return nil
}
