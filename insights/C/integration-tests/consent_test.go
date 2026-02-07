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
		state           bool
		useSystemSource bool
	}{
		"Set opt-in":        {state: true},
		"Set opt-out":       {state: false},
		"Set system opt-in": {state: true, useSystemSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)

			targetSource := fixture.source
			if tc.useSystemSource {
				targetSource = constants.DefaultCollectSource
			}

			_, err := runDriver(t, fixture, "set-consent", "static-test-source", "false")
			require.NoError(t, err)

			_, err = runDriver(t, fixture, "set-consent", targetSource, fmt.Sprintf("%v", tc.state))
			require.NoError(t, err)

			states := validateConsent(t, fixture.consentDir)

			checkSource := targetSource
			if tc.useSystemSource {
				checkSource = "SYSTEM"
			}

			assert.Equal(t, tc.state, states[checkSource], "Consent state mismatch for %s", checkSource)
			assert.False(t, states["static-test-source"], "Consent state mismatch for static-test-source")
		})
	}
}

func TestGetConsent(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		isNotSet        bool
		initialState    bool
		expectState     string
		useSystemSource bool
	}{
		"Reads opt-in state":               {initialState: true, expectState: "1"},
		"Reads opt-out state":              {initialState: false, expectState: "0"},
		"Handles unset state":              {isNotSet: true, expectState: "-1"},
		"Reads system source opt-in state": {initialState: true, expectState: "1", useSystemSource: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fixture := setupTestFixture(t)

			targetSource := fixture.source
			if tc.useSystemSource {
				targetSource = constants.DefaultCollectSource
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
