// main is the package for the C API.
package main

// most of this is copied from libinsights.go, keep them up to date.

/*
#include <stdlib.h>
#include "types.h"

extern char* collectInsights(const InsightsConfig*, const char*, const CollectFlags*, char**);
extern char* uploadInsights(const InsightsConfig*, const char**, size_t, const UploadFlags*);
extern ConsentState getConsentState(const InsightsConfig*, const char*);
extern char* setConsentState(const InsightsConfig*, const char*, bool);
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
		source            string
		metricsPath       *string
		sourceMetricsJSON []byte
		flags             *C.CollectFlags

		outReport **C.char

		mockOut []byte
		mockErr error
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
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			source: "platform",
		},

		"MetricsPath gets converted": {
			metricsPath: strPtr("path/to/metrics"),
		},

		"SourceMetricsJSON gets converted": {
			sourceMetricsJSON: []byte(`{"key": "value"}`),
		},

		"Flags get converted": {
			flags: &C.CollectFlags{
				period: C.uint32_t(10),
				force:  C.bool(true),
				dryRun: C.bool(true),
			},
		},

		"All get converted": {
			config: &CInsightsConfig{
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			source:      "wsl",
			metricsPath: strPtr("metrics"),
			flags: &C.CollectFlags{
				period: C.uint32_t(2000),
				force:  C.bool(false),
				dryRun: C.bool(false),
			},
		},

		// Report output
		"Report is returned when outReport and outReportLen are provided": {
			outReport: new(*C.char),
			mockOut:   []byte(`{"output": "report data"}`),
		},
		"Report is not returned when outReport is nil": {
			outReport: nil,
			mockOut:   []byte(`{"output": "no report"}`),
		},
		"Report is returned safely when empty": {
			outReport: new(*C.char),
			mockOut:   []byte(""),
		},
		"Report return is safe when output has null terminator in middle": {
			outReport: new(*C.char),
			mockOut:   []byte(`{"output": "report data with null \x00 in middle"}`),
		},

		// error case
		"error returns error string": {
			mockErr: errors.New("Error String"),
		},
		"Report is not returned in error case": {
			outReport: new(*C.char),
			mockErr:   errors.New("Error String"),
			mockOut:   []byte(`{"output": "no report in error"}`),
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
				Conf   insights.Config
				Source string
				Flags  insights.CollectFlags

				OutReport string
			}

			sourceStr := C.CString(tc.source)
			defer C.free(unsafe.Pointer(sourceStr))

			ret := collectCustomInsights(inConfig, sourceStr, tc.flags, tc.outReport, func(conf insights.Config, source string, flags insights.CollectFlags) ([]byte, error) {
				got.Conf = conf
				got.Source = source
				got.Flags = flags
				return tc.mockOut, tc.mockErr
			})
			defer C.free(unsafe.Pointer(ret))
			defer func() {
				if tc.outReport != nil {
					C.free(unsafe.Pointer(*tc.outReport))
				}
			}()

			if tc.mockErr == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.mockErr.Error())
			}

			// ensure SourceMetricsJSON is not nil for better comparison
			if got.Flags.SourceMetricsJSON == nil {
				got.Flags.SourceMetricsJSON = []byte{}
			}

			if tc.outReport != nil {
				got.OutReport = C.GoString(*tc.outReport)
			}

			assert.NotNil(t, got.Conf.Logger, "Logger should not be nil in the callback")
			got.Conf.Logger = nil // Logger is not part of the golden file, so we set it to nil for comparison.
			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			assert.Equal(t, want, got, "C structures should be correctly translated to Go")
		})
	}
}

// TestUploadImpl tests upload since import "C" and _test aren't compatible.
func TestUploadImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config  *CInsightsConfig
		sources []string
		flags   *C.UploadFlags

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
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			sources: []string{"platform"},
		},

		"Flags get converted": {
			flags: &C.UploadFlags{
				minAge: C.uint32_t(10),
				force:  C.bool(true),
				dryRun: C.bool(true),
			},
		},

		"All get converted": {
			config: &CInsightsConfig{
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			sources: []string{"wsl", "app2"},
			flags: &C.UploadFlags{
				minAge: C.uint32_t(2000),
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
			inConfig, configCleanup := makeConfig(tc.config)
			defer configCleanup()

			var got struct {
				Conf    insights.Config
				Sources []string
				Flags   insights.UploadFlags
			}

			// Convert sources to C string array
			var cSources **C.char
			var cSourcesLen C.size_t
			sourcesCleanup := func() {}
			if len(tc.sources) > 0 {
				cSourcesLen = C.size_t(len(tc.sources))
				sourcesPtr := make([]*C.char, len(tc.sources))
				for i, source := range tc.sources {
					sourcesPtr[i] = C.CString(source)
				}
				cSources = (**C.char)(unsafe.Pointer(&sourcesPtr[0]))
				sourcesCleanup = func() {
					for _, ptr := range sourcesPtr {
						C.free(unsafe.Pointer(ptr))
					}
				}
			}
			defer sourcesCleanup()

			ret := uploadCustomInsights(inConfig, cSources, cSourcesLen, tc.flags, func(conf insights.Config, sources []string, flags insights.UploadFlags) error {
				got.Conf = conf
				got.Sources = sources
				got.Flags = flags

				if got.Sources == nil {
					got.Sources = []string{}
				}

				return tc.err
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.err == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.err.Error())
			}

			assert.NotNil(t, got.Conf.Logger, "Logger should not be nil in the callback")
			got.Conf.Logger = nil // Logger is not part of the golden file, so we set it to nil for comparison.
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
		source string

		state C.ConsentState
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &CInsightsConfig{},
		},

		"Config gets converted": {
			config: &CInsightsConfig{
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			source: "platform",
		},

		// return cases
		"unknown state is correctly converted": {
			state: C.CONSENT_UNKNOWN,
		},

		"false state is correctly converted": {
			state: C.CONSENT_FALSE,
		},

		"true state is correctly converted": {
			state: C.CONSENT_TRUE,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// we need to convert the input here since making C strings inline is unsafe.
			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			var got struct {
				Conf   insights.Config
				Source string
			}

			sourceStr := C.CString(tc.source)
			defer C.free(unsafe.Pointer(sourceStr))

			ret := getCustomConsentState(inConfig, sourceStr, func(conf insights.Config, source string) C.ConsentState {
				got.Conf = conf
				got.Source = source
				return tc.state
			})

			assert.Equal(t, tc.state, ret, "Did not get expected consent state")

			assert.NotNil(t, got.Conf.Logger, "Logger should not be nil in the callback")
			got.Conf.Logger = nil // Logger is not part of the golden file, so we set it to nil for comparison.
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
		source string
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
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			source: "platform",
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
				Conf   insights.Config
				Source string
				State  bool
			}

			sourceStr := C.CString(tc.source)
			defer C.free(unsafe.Pointer(sourceStr))

			ret := setCustomConsentState(inConfig, sourceStr, tc.state, func(conf insights.Config, source string, state bool) error {
				got.Conf = conf
				got.Source = source
				got.State = state
				return tc.err
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.err == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.err.Error())
			}

			assert.NotNil(t, got.Conf.Logger, "Logger should not be nil in the callback")
			got.Conf.Logger = nil // Logger is not part of the golden file, so we set it to nil for comparison.
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
	consent, cache *string // Removed src since source is now passed as parameter
	verbose        bool
}

// makeConfig is a helper to create a C InsightsConfig.
func makeConfig(conf *CInsightsConfig) (cnf *C.InsightsConfig, clean func()) {
	defer func() {
		clean = func() {
			if cnf != nil {
				C.free(unsafe.Pointer(cnf.consentDir))
				C.free(unsafe.Pointer(cnf.insightsDir))
			}
		}
	}()

	if conf != nil {
		cnf = &C.InsightsConfig{}
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
