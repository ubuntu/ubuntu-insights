package constants

import (
	"os"
	"path/filepath"
)

// systemConfigDir returns the default directory for the system-wide configuration file on Windows.
// It resolves to %PROGRAMDATA%\ubuntu-insights (typically C:\ProgramData\ubuntu-insights).
// It panics if PROGRAMDATA is unset, as it is required to reliably locate the system-wide
// configuration directory.
func systemConfigDir() string {
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		panic("PROGRAMDATA environment variable is not set")
	}
	return filepath.Join(programData, DefaultAppFolder)
}
