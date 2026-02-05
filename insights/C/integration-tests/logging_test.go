package libinsights_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallback(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		useCallback bool
	}{
		"WithCallback":    {useCallback: true},
		"WithoutCallback": {useCallback: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)

			var args []string
			if tc.useCallback {
				args = append(args, "--log-file", fixture.logPath)
			}
			args = append(args, "collect", fixture.source, "--dry-run", "--force")

			output, err := runDriver(t, fixture, args...)
			require.NoError(t, err)

			if tc.useCallback {
				// Verify logs are written to file.
				counts := countLogLevels(t, fixture.logPath)
				totalLogs := 0
				for _, count := range counts {
					totalLogs += count
				}
				t.Logf("Captured logs: %v", counts)
				assert.Positive(t, totalLogs, "Expected logs in file")
				assert.Positive(t, counts[2], "Expected INFO(2) logs in file")
				assert.Positive(t, counts[3], "Expected DEBUG(3) logs in file")

				// In certain execution environments, libwayland itself emits logs that we can't control.
				assert.NotContains(t, strings.ToLower(output), "insights", "Expected no insights logs in stdout/stderr")
			} else {
				assert.Contains(t, strings.ToLower(output), "insights", "Expected insights logs in stdout/stderr")
			}
		})
	}
}
