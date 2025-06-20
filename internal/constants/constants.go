// Package constants is responsible for defining the constants used in the application.
// It also provides utility functions to get the default configuration and cache paths.
package constants

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// Version is the version of the application.
	Version = "Dev"

	// manGeneration is whenever or not man pages are being generated.
	manGeneration = "false"
)

const (
	// CmdName is the name of the command line tool.
	CmdName = "ubuntu-insights"

	// WebServiceCmdName is the name of the web service command.
	WebServiceCmdName = "ubuntu-insights-web-service"

	// IngestServiceCmdName is the name of the ingest service command.
	IngestServiceCmdName = "ubuntu-insights-ingest-service"

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

	// DefaultCollectSource is the default source when none are provided.
	DefaultCollectSource = runtime.GOOS

	// DefaultPeriod is the default value for the collector's period.
	DefaultPeriod = 0

	// DefaultMinAge is the default value for the uploader's min-age.
	DefaultMinAge = 604800
)

// Service constants.
const (
	// DefaultServiceFolder is the name of the default root folder for services.
	DefaultServiceFolder = "ubuntu-insights-services"

	// DefaultServiceReportsFolder is the name of the default reports folder for services.
	DefaultServiceReportsFolder = "reports"

	// LegacyReportTag is the tag used to indicate legacy ubuntu report files.
	LegacyReportTag = "ubuntu-report"
)

var (
	// DefaultConfigPath is the default app user configuration path. It's overridden when imported.
	DefaultConfigPath = DefaultAppFolder
	// DefaultCachePath is the default app user cache path. It's overridden when imported.
	DefaultCachePath = DefaultAppFolder
	// OptOutJSON is the data sent in case of Opt-Out choice.
	OptOutJSON = struct{ OptOut bool }{true}
)

// Service variables.
var (
	// DefaultServiceDataDir is the default data directory for services.
	DefaultServiceDataDir = DefaultServiceFolder

	// DefaultServiceReportsDir is the default reports directory for services.
	DefaultServiceReportsDir = filepath.Join(DefaultServiceDataDir, DefaultServiceReportsFolder)
)

func init() {
	DefaultServiceDataDir = filepath.Join("/var/lib", DefaultServiceFolder)
	defer func() {
		DefaultServiceReportsDir = filepath.Join(DefaultServiceDataDir, DefaultServiceReportsFolder)
	}()

	// This is to ensure that the man pages which include the default values
	// are not generated with the home path at time of generation.
	if manGeneration == "true" {
		DefaultConfigPath = ""
		DefaultCachePath = ""
		return
	}

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

	if runtime.GOOS != "linux" {
		DefaultServiceDataDir = filepath.Join(userCacheDir, DefaultServiceFolder)
	}
}
