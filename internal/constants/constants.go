// Package constants is responsible for defining the constants used in the application.
// It also provides utility functions to get the default configuration and cache paths.
package constants

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	// Version is the version of the application.
	Version = "Dev"
)

const (
	// CmdName is the name of the command line tool.
	CmdName = "ubuntu-insights"

	// DefaultAppFolder is the name of the default root folder.
	DefaultAppFolder = "ubuntu-insights"

	// DefaultLogLevel is the default log level selected without any verbosity flags.
	DefaultLogLevel = slog.LevelWarn

	// LocalFolder is the default name of the local collected reports folder.
	LocalFolder = "local"

	// UploadedFolder is the default name of the uploaded reports folder.
	UploadedFolder = "uploaded"

	// GlobalFileName is the default base name of the consent state files.
	GlobalFileName = "consent.toml"

	// ConsentSourceBaseSeparator is the default separator between the source and the base name of the consent state files.
	ConsentSourceBaseSeparator = "-"

	// ReportExt is the default extension for the report files.
	ReportExt = ".json"

	// MaxReports is the maximum number of report files that can exist in a folder.
	MaxReports = 150
)

var (
	// DefaultConfigPath is the default app user configuration path. It's overridden when imported.
	DefaultConfigPath = DefaultAppFolder
	// DefaultCachePath is the default app user cache path. It's overridden when imported.
	DefaultCachePath = DefaultAppFolder
	// OptOutJSON is the data sent in case of Opt-Out choice.
	OptOutJSON = struct{ OptOut bool }{true}
)

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Sprintf("Could not fetch config directory: %v", err))
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("Could not fetch cache directory: %v", err))
	}

	DefaultConfigPath = filepath.Join(userConfigDir, DefaultConfigPath)
	DefaultCachePath = filepath.Join(userCacheDir, DefaultCachePath)
}
