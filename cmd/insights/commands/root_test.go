package commands

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				setVerbosity(p)

				switch p {
				case 0:
					assert.True(t, slog.Default().Enabled(context.Background(), constants.DefaultLogLevel))
				case 1:
					assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelInfo))
				default:
					assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
				}
			}
		})
	}
}

func TestUsageError(t *testing.T) {
	app, err := New()
	require.NoError(t, err)

	// Test when SilenceUsage is true
	app.cmd.SilenceUsage = true
	assert.False(t, app.UsageError())

	// Test when SilenceUsage is false
	app.cmd.SilenceUsage = false
	assert.True(t, app.UsageError())
}
