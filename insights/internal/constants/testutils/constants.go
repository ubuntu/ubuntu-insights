// Package testutils provides test utilities for the constants package.
package testutils

import (
	"github.com/ubuntu/ubuntu-insights/common/testsdetection"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
)

func init() {
	// No import outside of testing environment.
	testsdetection.MustBeTesting()
}

// Normalize normalizes programmatic constants to ensure they have static values for testing.
func Normalize() {
	constants.PlatformSource = "PLATFORM-SOURCE"
	constants.PlatformConsentFile = constants.PlatformSource + constants.ConsentSourceBaseSeparator + constants.DefaultConsentFilenameBase
}
