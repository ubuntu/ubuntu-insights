// Package constants is responsible for defining the constants used in the application.
// It also provides utility functions to get the default configuration and cache paths.
package constants

import (
	"encoding/json"
	"fmt"
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

	// DefaultAppFolder is the name of the default root folder.
	DefaultAppFolder = "ubuntu-insights"

	// LocalFolder is the default name of the local collected reports folder.
	LocalFolder = "local"

	// UploadedFolder is the default name of the uploaded reports folder.
	UploadedFolder = "uploaded"

	// DefaultConsentFileName is the default base name of the consent state files.
	DefaultConsentFileName = "consent.toml"

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

	// MaxConcurrentUploadsPerSource is the maximum number of concurrent uploads per source.
	MaxConcurrentUploadsPerSource = MaxReports

	// MaxConcurrentSources is the maximum number of sources that can be processed concurrently.
	MaxConcurrentSources = 10
)

var (
	// DefaultConfigPath is the default app user configuration path. It's overridden when imported.
	DefaultConfigPath = DefaultAppFolder
	// DefaultCachePath is the default app user cache path. It's overridden when imported.
	DefaultCachePath = DefaultAppFolder
	// OptOutJSON is the data sent in case of Opt-Out choice.
	OptOutJSON = struct{ OptOut bool }{true}
	// OptOutPayload is the marshalled version of OptOutJSON.
	OptOutPayload []byte
)

func init() {
	initalizePaths()
	initializeOptOutPayload()
}

// initalizePaths initializes the default configuration and cache paths based on the user's home directory.
// If the manGeneration variable is set to "true", it will clear the path variables to avoid including potentially
// misleading paths in the generated man pages.
func initalizePaths() {
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
}

// initializeOptOutPayload initializes the OptOutPayload variable with the marshalled OptOutJSON.
func initializeOptOutPayload() {
	var err error
	OptOutPayload, err = json.Marshal(OptOutJSON)
	if err != nil {
		panic(fmt.Sprintf("Could not marshal OptOutJSON: %v", err))
	}
}
