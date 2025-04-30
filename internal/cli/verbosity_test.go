package cli_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/internal/cli"
	"github.com/ubuntu/ubuntu-insights/internal/constants"
)

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
