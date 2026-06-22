package constants

// systemConfigDir returns the default directory for the system-wide configuration file on macOS.
func systemConfigDir() string {
	return "/Library/Application Support/ubuntu-insights"
}
