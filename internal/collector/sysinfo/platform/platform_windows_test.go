package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
)

func TestNewWindows(t *testing.T) {
	t.Parallel()

	s := platform.New()
	require.NotEmpty(t, s, "platform sysinfo Collector has custom fields")
}

func TestCollectWindows(t *testing.T) {
	t.Parallel()

	s := platform.New()
	info, err := s.Collect()
	require.NoError(t, err)
	assert.Empty(t, info, "Windows platform info should be empty")
}
