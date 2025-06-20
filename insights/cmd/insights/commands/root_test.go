package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
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

func TestRootCmd(t *testing.T) {
	app, err := New()
	require.NoError(t, err)

	cmd := app.RootCmd()

	assert.NotNil(t, cmd, "Returned root cmd should not be nil")
	assert.Equal(t, constants.CmdName, cmd.Name())
}
