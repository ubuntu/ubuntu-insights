package platform_test

import (
	"testing"

	"github.com/ubuntu/ubuntu-insights/internal/collector/sysinfo/platform"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	_ = platform.WSL{}
}
