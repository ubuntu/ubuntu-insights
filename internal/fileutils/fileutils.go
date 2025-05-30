// Package fileutils provides utility functions for handling files.
package fileutils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite writes data to a file atomically.
// If the file already exists, then it will be overwritten.
// Not atomic on Windows.
func AtomicWrite(path string, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "tmp-*.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %v", err)
	}
	defer func() {
		_ = tmp.Close()
		if e := os.Remove(tmp.Name()); e != nil && !os.IsNotExist(e) {
			err = fmt.Errorf("failed to remove temporary file %s: %v", tmp.Name(), e)
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

// ReadFileLogError returns the data in the file path, trimming whitespace, or "" on error.
// If an error occurs, it logs the error at the Warn level.
func ReadFileLogError(path string, log *slog.Logger) string {
	return ReadFileLog(path, log, slog.LevelWarn)
}

// ReadFileLog returns the data in the file path, trimming whitespace, or "" on error.
// If an error occurs, it logs the error at the specified level.
func ReadFileLog(path string, log *slog.Logger, level slog.Level) string {
	log.Debug("reading file", "file", path)
	f, err := os.ReadFile(path)
	if err != nil {
		log.Log(context.Background(), level, "failed to read file", "file", path, "error", err)
		return ""
	}
	return strings.TrimSpace(string(f))
}

// ConvertUnitToBytes takes a string bytes unit and converts value to bytes.
// If the unit is not recognized an error is returned, value is returned as is.
func ConvertUnitToBytes[T ~int | ~int64 | ~uint | ~uint64 | ~float64](unit string, value T) (T, error) {
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

// ConvertUnitToStandard takes a string bytes unit and converts value to the standard unit (Mebibytes).
// If the unit is not recognized an error is returned, value is returned as is.
func ConvertUnitToStandard[T ~int | ~int64 | ~uint | ~uint64 | ~float64](unit string, value T) (T, error) {
	v, err := ConvertUnitToBytes(unit, value)
	if err != nil {
		return v, err
	}
	return v / 1024 / 1024, nil
}
