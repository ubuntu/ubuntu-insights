package constants

import (
	"os"
	"path/filepath"
)

// systemConfigDir returns the default directory for the system-wide configuration file on Windows.
// It resolves to %PROGRAMDATA%\ubuntu-insights (typically C:\ProgramData\ubuntu-insights).
func systemConfigDir() string {
	return filepath.Join(os.Getenv("PROGRAMDATA"), "ubuntu-insights")
}
