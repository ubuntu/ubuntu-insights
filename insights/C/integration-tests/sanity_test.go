package libinsights_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	t.Parallel()

	fixture := setupTestFixture(t)
	fixture.makePanic = true

	output, err := runDriver(t, fixture, "set-consent", "static-test-source", "false")
	if !systemLib {
		require.Error(t, err, "Expected error due to intentional panic")
		require.Contains(t, output, "panic", "Expected panic message in output")
		return
	}
	require.NoError(t, err, "Expected no error when running with system lib. Check if we are correctly linking to the system library")
	require.NotContains(t, output, "panic", "Did not expect panic message in output when running with system lib")
}
