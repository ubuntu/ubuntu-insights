package platform_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo/platform"
)

func TestNewWindows(t *testing.T) {
	t.Parallel()

	s := platform.New(slog.Default())
	require.NotEmpty(t, s, "platform sysinfo Collector has custom fields")
}

func TestCollectWindows(t *testing.T) {
	t.Parallel()

	s := platform.New(slog.Default())
	info, err := s.Collect()
	require.NoError(t, err)
	assert.Empty(t, info, "Windows platform info should be empty")
}
