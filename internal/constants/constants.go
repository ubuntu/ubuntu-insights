// Package constants is responsible for defining the constants used in the application.
// It also provides utility functions to get the default configuration and cache paths.
package constants

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	// CmdName is the name of the command line tool.
	CmdName = "ubuntu-insights"

	// DefaultAppFolder is the name of the default root folder.
	DefaultAppFolder = "ubuntu-insights"

	// DefaultLogLevel is the default log level selected without any verbosity flags.
	DefaultLogLevel = slog.LevelInfo

	// GlobalFileName is the default base name of the consent state files.
	GlobalFileName = "consent.toml"

	// ConsentSourceBaseSeparator is the default separator between the source and the base name of the consent state files
	ConsentSourceBaseSeparator = "-"

	// Report extension is the default extension for the report files.
	ReportExtension = ".json"
)

type options struct {
	baseDir func() (string, error)
}

type option func(*options)

// GetDefaultConfigPath is the default path to the configuration file.
func GetDefaultConfigPath(opts ...option) string {
	o := options{baseDir: os.UserConfigDir}
	for _, opt := range opts {
		opt(&o)
	}

	return filepath.Join(getBaseDir(o.baseDir), DefaultAppFolder)
}

// GetDefaultCachePath is the default path to the cache directory.
func GetDefaultCachePath(opts ...option) string {
	o := options{baseDir: os.UserCacheDir}
	for _, opt := range opts {
		opt(&o)
	}

	return filepath.Join(getBaseDir(o.baseDir), DefaultAppFolder)
}

// getBaseDir is a helper function to handle the case where the baseDir function returns an error, and instead return an empty string.
func getBaseDir(baseDirFunc func() (string, error)) string {
	dir, err := baseDirFunc()
	if err != nil {
		return ""
	}
	return dir
}
