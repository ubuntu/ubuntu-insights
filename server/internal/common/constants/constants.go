// Package constants is responsible for defining the constants used in the application.
// It also provides utility functions to get the default configuration and cache paths.
package constants

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	// Version is the version of the application.
	Version = "Dev"
)

const (
	// WebServiceCmdName is the name of the web service command.
	WebServiceCmdName = "ubuntu-insights-web-service"

	// IngestServiceCmdName is the name of the ingest service command.
	IngestServiceCmdName = "ubuntu-insights-ingest-service"
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

// Service variables.
var (
	// DefaultServiceDataDir is the default data directory for services.
	DefaultServiceDataDir = DefaultServiceFolder

	// DefaultServiceReportsDir is the default reports directory for services.
	DefaultServiceReportsDir = filepath.Join(DefaultServiceDataDir, DefaultServiceReportsFolder)
)

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Sprintf("Could not fetch cache directory: %v", err))
	}

	DefaultServiceDataDir = filepath.Join(userCacheDir, DefaultServiceFolder)
	DefaultServiceReportsDir = filepath.Join(DefaultServiceDataDir, DefaultServiceReportsFolder)
}
