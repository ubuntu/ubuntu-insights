package cli

import (
	"log/slog"
	"os"

	"github.com/ubuntu/ubuntu-insights/common/internal/constants"
)

// SetVerbosity sets the logging level for the default logger based on the verbose flag count.
// If level is negative, it infers quiet operation and sets the logger to errors only.
//
// This function has the same behaviors as slog.SetLogLoggerLevel.
func SetVerbosity(level int) {
	slog.SetLogLoggerLevel(getLevel(level))
}

// SetSlog sets the logging level and format for the default logger.
func SetSlog(level int, jsonLogs bool) {
	slogLevel := getLevel(level)
	if jsonLogs {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel})))
		return
	}

	SetVerbosity(level)
}

func getLevel(level int) slog.Level {
	if level < 0 {
		return slog.LevelError
	}

	switch level {
	case 0:
		return constants.DefaultLogLevel
	case 1:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}
