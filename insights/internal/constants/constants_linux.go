package constants

// systemConfigDir returns the default directory for the system-wide configuration file on Linux.
func systemConfigDir() string {
	return "/etc/" + DefaultAppFolder
}
