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
	DefaultConfigPath = userConfigDir(os.UserCacheDir) + string(os.PathSeparator) + DefaultAppFolder

	// DefaultCachePath is the default path to the cache directory
	DefaultCachePath = userCacheDir(os.UserConfigDir) + string(os.PathSeparator) + DefaultAppFolder
)

func userConfigDir(osUserConfigDir func() (string, error)) string {
	dir, err := osUserConfigDir()
	if err != nil {
		return ""
	}
	return dir
}

func userCacheDir(osUserCacheDir func() (string, error)) string {
	dir, err := osUserCacheDir()
	if err != nil {
		return ""
	}
	return dir
}
