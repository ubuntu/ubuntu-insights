package libinsights_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
)

func TestUpload(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		consentState bool
		dryRun       bool
		expectError  bool
	}{
		"DryRun Upload": {
			dryRun:      true,
			expectError: false,
		},
		"Opt-in Upload": {
			consentState: true,
			dryRun:       false,
			expectError:  false,
		},
		"Opt-out upload": {
			dryRun:      false,
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !tc.dryRun && systemLib {
				t.Skip("Skipping regular upload test with system lib")
			}

			testServer, testServerState := setupTestServer(t)
			defer testServer.Close()

			fixture := setupTestFixture(t)

			// We want to ensure that upload is filtering based on consent, so we always consent at the collect stage, then set the desired consent state before upload.
			_, err := runDriver(t, fixture, "set-consent", fixture.source, "true")
			require.NoError(t, err, "Setup: failed to set consent state")

			_, err = runDriver(t, fixture, "collect", fixture.source)
			require.NoError(t, err, "Setup: failed to collect report")

			_, err = runDriver(t, fixture, "set-consent", fixture.source, fmt.Sprintf("%v", tc.consentState))
			require.NoError(t, err, "Setup: failed to set consent state")

			args := []string{"upload", fixture.source}
			if tc.dryRun {
				args = append(args, "--dry-run")
			}
			if !systemLib {
				fixture.uploadURL = testServer.URL
			}

			out, err := runDriver(t, fixture, args...)

			if tc.expectError {
				require.Error(t, err, "Expected upload to fail")
				return
			}
			require.NoError(t, err, "Unexpected upload failed: %s", out)

			if !tc.dryRun {
				testServerState.mu.Lock()
				got := testServerState.Received
				testServerState.mu.Unlock()
				want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
				assert.Equal(t, want, got, "Received reports do not match expected")
			}
		})
	}
}
