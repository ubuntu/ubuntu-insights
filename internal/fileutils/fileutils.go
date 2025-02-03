// Package fileutils provides utility functions for handling files.
package fileutils

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileLogError returns the data in the file path, trimming whitespace, or "" on error.
func ReadFileLogError(path string, log *slog.Logger) string {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Warn("failed to read file", "file", path, "error", err)
		return ""
	}

	return strings.TrimSpace(string(f))
}

// ConvertUnitToBytes takes a string bytes unit and converts value to bytes.
// If the unit is not recognized an error is returned, value is returned as is.
func ConvertUnitToBytes(unit string, value int) (int, error) {
	switch strings.ToLower(unit) {
	case "":
		fallthrough
	case "b":
		return value, nil
	case "k":
		fallthrough
	case "kb":
		fallthrough
	case "kib":
		return value * 1024, nil
	case "m":
		fallthrough
	case "mb":
		fallthrough
	case "mib":
		return value * 1024 * 1024, nil
	case "g":
		fallthrough
	case "gb":
		fallthrough
	case "gib":
		return value * 1024 * 1024 * 1024, nil
	case "t":
		fallthrough
	case "tb":
		fallthrough
	case "tib":
		return value * 1024 * 1024 * 1024 * 1024, nil
	default:
		return value, fmt.Errorf("unrecognized bytes unit: %s", unit)
	}
}

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
