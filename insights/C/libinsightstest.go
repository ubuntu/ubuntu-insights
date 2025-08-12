// main is the package for the C API.
package main

// most of this is copied from libinsights.go, keep them up to date.

/*
#include <stdlib.h>
#include "types.h"

extern char* insights_collect(const insights_config*, const char*, const insights_collect_flags*, char**);
extern char* insights_write(const insights_config*, const char*, const char*, const insights_write_flags*);
extern char* insights_upload(const insights_config*, const char**, size_t, const insights_upload_flags*);
extern insights_consent_state insights_get_consent_state(const insights_config*, const char*);
extern char* insights_set_consent_state(const insights_config*, const char*, bool);
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
		config            *insightsConfig
		source            string
		metricsPath       *string
		sourceMetricsJSON []byte
		flags             *C.insights_collect_flags

		outReport **C.char

		mockOut []byte
		mockErr error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config:      &insightsConfig{},
			metricsPath: strPtr(""),
			flags:       &C.insights_collect_flags{},
		},

		"Config gets converted": {
			config: &insightsConfig{
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
			flags: &C.insights_collect_flags{
				period:  C.uint32_t(10),
				force:   C.bool(true),
				dry_run: C.bool(true),
			},
		},

		"All get converted": {
			config: &insightsConfig{
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			source:      "wsl",
			metricsPath: strPtr("metrics"),
			flags: &C.insights_collect_flags{
				period:  C.uint32_t(2000),
				force:   C.bool(false),
				dry_run: C.bool(false),
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
				tc.flags = &C.insights_collect_flags{}
			}

			if tc.metricsPath != nil {
				tc.flags.source_metrics_path = C.CString(*tc.metricsPath)
				defer C.free(unsafe.Pointer(tc.flags.source_metrics_path))
			}

			if tc.sourceMetricsJSON != nil {
				tc.flags.source_metrics_json = unsafe.Pointer(&tc.sourceMetricsJSON[0])
				tc.flags.source_metrics_json_len = C.size_t(len(tc.sourceMetricsJSON))
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

// TestWriteImpl tests the write functionality.
func TestWriteImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config *insightsConfig
		source string
		report string
		flags  *C.insights_write_flags

		mockErr error
	}{
		"Empty case": {},
		"Parameters are passed": {
			config: &insightsConfig{
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			source: "platform",
			report: "report data",
			flags: &C.insights_write_flags{
				period:  C.uint32_t(10),
				dry_run: C.bool(true),
			},
		},

		// Error case
		"Error is returned": {
			mockErr: errors.New("Error String"),
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
				Report string
				Flags  insights.WriteFlags
			}

			sourceStr := C.CString(tc.source)
			defer C.free(unsafe.Pointer(sourceStr))

			reportStr := C.CString(tc.report)
			defer C.free(unsafe.Pointer(reportStr))

			ret := writeCustomInsights(inConfig, sourceStr, reportStr, tc.flags, func(conf insights.Config, source string, report []byte, flags insights.WriteFlags) error {
				got.Conf = conf
				got.Source = source
				got.Report = string(report)
				got.Flags = flags
				return tc.mockErr
			})
			defer C.free(unsafe.Pointer(ret))

			if tc.mockErr == nil {
				assert.Nil(t, ret)
			} else {
				assert.Equal(t, C.GoString(ret), tc.mockErr.Error())
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
		config  *insightsConfig
		sources []string
		flags   *C.insights_upload_flags

		err error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &insightsConfig{},
			flags:  &C.insights_upload_flags{},
		},

		"Config gets converted": {
			config: &insightsConfig{
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			sources: []string{"platform"},
		},

		"Flags get converted": {
			flags: &C.insights_upload_flags{
				min_age: C.uint32_t(10),
				force:   C.bool(true),
				dry_run: C.bool(true),
			},
		},

		"All get converted": {
			config: &insightsConfig{
				consent: strPtr("home/etc/wsl/dir"),
				cache:   strPtr("insights/wsl/dir"),
				verbose: false,
			},
			sources: []string{"wsl", "app2"},
			flags: &C.insights_upload_flags{
				min_age: C.uint32_t(2000),
				force:   C.bool(false),
				dry_run: C.bool(false),
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
		config *insightsConfig
		source string

		state C.insights_consent_state
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &insightsConfig{},
		},

		"Config gets converted": {
			config: &insightsConfig{
				consent: strPtr("home/etc/dir"),
				cache:   strPtr("insights/dir"),
				verbose: true,
			},
			source: "platform",
		},

		// return cases
		"unknown state is correctly converted": {
			state: C.INSIGHTS_CONSENT_UNKNOWN,
		},

		"false state is correctly converted": {
			state: C.INSIGHTS_CONSENT_FALSE,
		},

		"true state is correctly converted": {
			state: C.INSIGHTS_CONSENT_TRUE,
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

			ret := getCustomConsentState(inConfig, sourceStr, func(conf insights.Config, source string) C.insights_consent_state {
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
		config *insightsConfig
		source string
		state  C.bool

		err error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			config: &insightsConfig{},
		},

		"Config gets converted": {
			config: &insightsConfig{
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

// insightsConfig lets us setup a C.insights_config easier.
type insightsConfig struct {
	consent, cache *string // Removed src since source is now passed as parameter
	verbose        bool
}

// makeConfig is a helper to create a C insights_config.
func makeConfig(conf *insightsConfig) (cnf *C.insights_config, clean func()) {
	defer func() {
		clean = func() {
			if cnf != nil {
				C.free(unsafe.Pointer(cnf.consent_dir))
				C.free(unsafe.Pointer(cnf.insights_dir))
			}
		}
	}()

	if conf != nil {
		cnf = &C.insights_config{}
		if conf.consent != nil {
			cnf.consent_dir = C.CString(*conf.consent)
		}
		if conf.cache != nil {
			cnf.insights_dir = C.CString(*conf.cache)
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
