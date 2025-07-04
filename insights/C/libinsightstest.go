// main is the package for the C API.
package main

// most of this is copied from libinsights.go, keep them up to date.

/*
#include <stdlib.h>
#include "insights_types.h"

extern char* collectInsights(const InsightsConfig*, const char*, const CollectFlags*);
extern char* uploadInsights(const InsightsConfig*, const UploadFlags*);
extern ConsentState getConsentState(const InsightsConfig*);
extern char* setConsentState(const InsightsConfig*, bool);
*/
import "C"

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/ubuntu/ubuntu-insights/common/testutils"
	"github.com/ubuntu/ubuntu-insights/insights"
)

// TestCollectImpl tests collect since import "C" and _test aren't compatible.
func TestCollectImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config            *CInsightsConfig
		metricsPath       *string
		sourceMetricsJSON []byte
		flags             *C.CollectFlags

		err error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config:      &CInsightsConfig{},
			metricsPath: strPtr(""),
			flags:       &C.CollectFlags{},
		},

		"Config gets converted": {
			config: &CInsightsConfig{
				src:     strPtr("platform"),
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
		},

		"MetricsPath gets converted": {
			metricsPath: strPtr("path/to/metrics"),
		},

		"SourceMetricsJSON gets converted": {
			sourceMetricsJSON: []byte(`{"key": "value"}`),
		},

		"Flags get converted": {
			flags: &C.CollectFlags{
				period: C.uint(10),
				force:  C.bool(true),
				dryRun: C.bool(true),
			},
		},

		"All get converted": {
			config: &CInsightsConfig{
				src:     strPtr("wsl"),
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			metricsPath: strPtr("metrics"),
			flags: &C.CollectFlags{
				period: C.uint(2000),
				force:  C.bool(false),
				dryRun: C.bool(false),
			},
		},

		// error case
		"error returns error string": {
			err: errors.New("Error String"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// we need to convert the input here since making C strings inline is unsafe.
			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			if tc.flags == nil {
				tc.flags = &C.CollectFlags{}
			}

			if tc.metricsPath != nil {
				tc.flags.sourceMetricsPath = C.CString(*tc.metricsPath)
				defer C.free(unsafe.Pointer(tc.flags.sourceMetricsPath))
			}

			if tc.sourceMetricsJSON != nil {
				tc.flags.sourceMetricsJSON = unsafe.Pointer(&tc.sourceMetricsJSON[0])
				tc.flags.sourceMetricsJSONLen = C.size_t(len(tc.sourceMetricsJSON))
			}

			var got struct {
				Conf  insights.Config
				Flags insights.CollectFlags
			}

			ret := collectCustomInsights(inConfig, tc.flags, func(conf insights.Config, flags insights.CollectFlags) error {
				got.Conf = conf
				got.Flags = flags
				return tc.err
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.err == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.err.Error())
			}

			// ensure SourceMetricsJSON is not nil for better comparison
			if got.Flags.SourceMetricsJSON == nil {
				got.Flags.SourceMetricsJSON = []byte{}
			}
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "C structures should be correctly translated to Go")
		})
	}
}

// TestUploadImpl tests upload since import "C" and _test aren't compatible.
func TestUploadImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config *CInsightsConfig
		flags  *C.UploadFlags

		err error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &CInsightsConfig{},
			flags:  &C.UploadFlags{},
		},

		"Config gets converted": {
			config: &CInsightsConfig{
				src:     strPtr("platform"),
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
		},

		"Flags get converted": {
			flags: &C.UploadFlags{
				minAge: C.uint(10),
				force:  C.bool(true),
				dryRun: C.bool(true),
			},
		},

		"All get converted": {
			config: &CInsightsConfig{
				src:     strPtr("wsl"),
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			flags: &C.UploadFlags{
				minAge: C.uint(2000),
				force:  C.bool(false),
				dryRun: C.bool(false),
			},
		},

		// error case
		"error returns error string": {
			err: errors.New("Error String"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// we need to convert the input here since making C strings inline is unsafe.
			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			var got struct {
				Conf  insights.Config
				Flags insights.UploadFlags
			}

			ret := uploadCustomInsights(inConfig, tc.flags, func(conf insights.Config, flags insights.UploadFlags) error {
				got.Conf = conf
				got.Flags = flags
				return tc.err
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.err == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.err.Error())
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "C structures should be correctly translated to Go")
		})
	}
}

// TestGetConsentImpl tests getConsentState since import "C" and _test aren't compatible.
func TestGetConsentImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config *CInsightsConfig

		state insights.ConsentState
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &CInsightsConfig{},
		},

		"Config gets converted": {
			config: &CInsightsConfig{
				src:     strPtr("platform"),
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
		},

		// return cases
		"unknown state is correctly converted": {
			state: insights.ConsentUnknown,
		},

		"false state is correctly converted": {
			state: insights.ConsentFalse,
		},

		"true state is correctly converted": {
			state: insights.ConsentTrue,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// we need to convert the input here since making C strings inline is unsafe.
			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			var got insights.Config

			ret := getCustomConsentState(inConfig, func(conf insights.Config) insights.ConsentState {
				got = conf
				return tc.state
			})

			switch tc.state {
			case insights.ConsentUnknown:
				// we have to convert C.ConsentState to C.ConsentState because...
				assert.Equal(t, C.ConsentState(C.CONSENT_UNKNOWN), ret)
			case insights.ConsentFalse:
				assert.Equal(t, C.ConsentState(C.CONSENT_FALSE), ret)
			case insights.ConsentTrue:
				assert.Equal(t, C.ConsentState(C.CONSENT_TRUE), ret)
			default:
				panic("Test case wants invalid enum!")
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "C structures should be correctly translated to Go")
		})
	}
}

// TestSetConsentImpl tests setConsentState since import "C" and _test aren't compatible.
func TestSetConsentImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config *CInsightsConfig
		state  C.bool

		err error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &CInsightsConfig{},
		},

		"Config gets converted": {
			config: &CInsightsConfig{
				src:     strPtr("platform"),
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
		},

		"false state is correctly converted": {
			state: C.bool(false),
		},

		"true state is correctly converted": {
			state: C.bool(true),
		},

		// error case
		"error returns error string": {
			err: errors.New("Error String"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// we need to convert the input here since making C strings inline is unsafe.
			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			var got struct {
				Conf  insights.Config
				State bool
			}

			ret := setCustomConsentState(inConfig, tc.state, func(conf insights.Config, state bool) error {
				got.Conf = conf
				got.State = state
				return tc.err
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.err == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.err.Error())
			}

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "C structures should be correctly translated to Go")
		})
	}
}

// strPtr allows us to take the address of a string literal.
func strPtr(in string) *string {
	return &in
}

// CInsightsConfig lets us setup a C.InsightsConfig easier.
type CInsightsConfig struct {
	src, consent, cache *string
	verbose             bool
}

// makeConfig is a helper to create a C InsightsConfig.
func makeConfig(conf *CInsightsConfig) (cnf *C.InsightsConfig, clean func()) {
	defer func() {
		clean = func() {
			if cnf != nil {
				C.free(unsafe.Pointer(cnf.source))
				C.free(unsafe.Pointer(cnf.consentDir))
				C.free(unsafe.Pointer(cnf.insightsDir))
			}
		}
	}()

	if conf != nil {
		cnf = &C.InsightsConfig{}
		if conf.src != nil {
			cnf.source = C.CString(*conf.src)
		}
		if conf.consent != nil {
			cnf.consentDir = C.CString(*conf.consent)
		}
		if conf.cache != nil {
			cnf.insightsDir = C.CString(*conf.cache)
		}
		cnf.verbose = C.bool(conf.verbose)
	}

	return cnf, clean
}

// TestMainImpl calls main which does nothing.
func TestMainImpl(t *testing.T) {
	t.Parallel()
	main()
}
