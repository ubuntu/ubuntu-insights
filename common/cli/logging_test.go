package cli_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/common/cli"
	"github.com/ubuntu/ubuntu-insights/common/internal/constants"
)

// hacky way to allow us to reset the default logger.
var defaultLogger = *slog.Default()

func TestSetVerbosity(t *testing.T) {
	testCases := []struct {
		name    string
		pattern []int
	}{
		{
			name:    "info",
			pattern: []int{1},
		},
		{
			name:    "none",
			pattern: []int{0},
		},
		{
			name:    "info none",
			pattern: []int{1, 0},
		},
		{
			name:    "info debug",
			pattern: []int{1, 2},
		},
		{
			name:    "info debug none",
			pattern: []int{1, 2, 0},
		},
		{
			name:    "debug",
			pattern: []int{2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slog.SetDefault(&defaultLogger)

			for _, p := range tc.pattern {
				cli.SetVerbosity(p)

				switch p {
				case 0:
					assert.True(t, slog.Default().Enabled(context.Background(), constants.DefaultLogLevel))
					assert.False(t, slog.Default().Enabled(context.Background(), constants.DefaultLogLevel-1))
				case 1:
					assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
					assert.False(t, slog.Default().Enabled(context.Background(), slog.LevelInfo-1))
				default:
					assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
					assert.False(t, slog.Default().Enabled(context.Background(), slog.LevelDebug-1))
				}
			}
		})
	}
}

func TestSetSlog(t *testing.T) {
	testCases := []struct {
		name    string
		level   int
		jsonLog bool
	}{
		{
			name:    "info",
			level:   1,
			jsonLog: false,
		},
		{
			name:    "none",
			level:   0,
			jsonLog: false,
		},
		{
			name:    "info json",
			level:   1,
			jsonLog: true,
		},
		{
			name:    "debug json",
			level:   2,
			jsonLog: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slog.SetDefault(&defaultLogger)
			cli.SetSlog(tc.level, tc.jsonLog)

			_, isJSON := slog.Default().Handler().(*slog.JSONHandler)
			assert.Equal(t, tc.jsonLog, isJSON, "unexpected log handler type")
		})
	}
}
