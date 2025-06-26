package cli

import (
	"log/slog"

	"github.com/ubuntu/ubuntu-insights/common/internal/constants"
)

// SetVerbosity sets the global logging level based on the verbose flag count.
func SetVerbosity(level int) {
	switch level {
	case 0:
		slog.SetLogLoggerLevel(constants.DefaultLogLevel)
	case 1:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	default:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
}
