package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
