package constants

import (
	"os"

	"log/slog"
)

const (
	// DefaultAppFolder is the name of the default root folder
	DefaultAppFolder = "ubuntu-insights"

	// DefaultLogLevel is the default log level selected without any verbosity flags
	DefaultLogLevel = slog.LevelInfo
)

var (
	// DefaultConfigPath is the default path to the configuration file
	DefaultConfigPath = userConfigDir() + string(os.PathSeparator) + DefaultAppFolder

	// DefaultCachePath is the default path to the cache directory
	DefaultCachePath = userCacheDir() + string(os.PathSeparator) + DefaultAppFolder
)

func userConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return dir
}

func userCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return dir
}
