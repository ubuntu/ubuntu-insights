// Package fileutils provides utility functions for handling files.
package fileutils

import (
	"fmt"
	"log/slog"
	"os"
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
