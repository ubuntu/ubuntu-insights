package libinsights_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

func TestSetConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		state             bool
		usePlatformSource bool
	}{
		"Set opt-in":        {state: true},
		"Set opt-out":       {state: false},
		"Set system opt-in": {state: true, usePlatformSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)

			targetSource := fixture.source
			if tc.usePlatformSource {
				targetSource = ""
			}

			_, err := runDriver(t, fixture, "set-consent", "static-test-source", "false")
			require.NoError(t, err)

			_, err = runDriver(t, fixture, "set-consent", targetSource, fmt.Sprintf("%v", tc.state))
			require.NoError(t, err)

			states := validateConsent(t, fixture.consentDir)

			checkSource := targetSource
			if tc.usePlatformSource {
				checkSource = constants.PlatformSource
			}

			assert.Equal(t, tc.state, states[checkSource], "Consent state mismatch for %s", checkSource)
			assert.False(t, states["static-test-source"], "Consent state mismatch for static-test-source")
		})
	}
}

func TestGetConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		isNotSet          bool
		initialState      bool
		expectState       string
		usePlatformSource bool
	}{
		"Reads opt-in state":               {initialState: true, expectState: "1"},
		"Reads opt-out state":              {initialState: false, expectState: "0"},
		"Handles unset state":              {isNotSet: true, expectState: "-1"},
		"Reads system source opt-in state": {initialState: true, expectState: "1", usePlatformSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)

			targetSource := fixture.source
			if tc.usePlatformSource {
				targetSource = ""
			}

			if !tc.isNotSet {
				_, err := runDriver(t, fixture, "set-consent", targetSource, fmt.Sprintf("%v", tc.initialState))
				require.NoError(t, err)
			}

			// The result will be printed to stdout, so we need to redirect everything else to a log file to avoid mixing it with logs.
			out, err := runDriver(t, fixture, "--log-file", fixture.logPath, "get-consent", targetSource)
			require.NoError(t, err)
			assert.Equal(t, tc.expectState, strings.TrimSpace(out), "Consent state mismatch")
		})
	}
}
