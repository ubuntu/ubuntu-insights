// main is the package for the C API.
package main

/*
#include "libinsightstest.h"
*/
import "C"

import (
	"errors"
	"log/slog"
	"runtime"
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
			mockErr: errors.New("error string"),
		},
		"Report is not returned in error case": {
			outReport: new(*C.char),
			mockErr:   errors.New("error string"),
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

// TestCompileImpl tests compile.
func TestCompileImpl(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config            *insightsConfig
		metricsPath       *string
		sourceMetricsJSON []byte

		outReport **C.char

		mockOut []byte
		mockErr error
	}{
		// conversion cases
		"Null values are empty": {},

		"Empty values are empty": {
			metricsPath: strPtr(""),
		},

		"Arguments get converted": {
			config: &insightsConfig{
				verbose: true,
			},
			metricsPath:       strPtr("metrics"),
			sourceMetricsJSON: []byte(`{"key": "value"}`),
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
		"Error returns error string": {
			mockErr: errors.New("error string"),
		},
		"Report is not returned in error case": {
			outReport: new(*C.char),
			mockErr:   errors.New("error string"),
			mockOut:   []byte(`{"output": "no report in error"}`),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inConfig, cleanup := makeConfig(tc.config)
			defer cleanup()

			flags := &C.insights_compile_flags{}
			if tc.metricsPath != nil {
				flags.source_metrics_path = C.CString(*tc.metricsPath)
				defer C.free(unsafe.Pointer(flags.source_metrics_path))
			}

			if tc.sourceMetricsJSON != nil {
				flags.source_metrics_json = unsafe.Pointer(&tc.sourceMetricsJSON[0])
				flags.source_metrics_json_len = C.size_t(len(tc.sourceMetricsJSON))
			}

			var got struct {
				Conf  insights.Config
				Flags insights.CompileFlags

				OutReport string
			}

			ret := compileCustomInsights(inConfig, flags, tc.outReport, func(conf insights.Config, flags insights.CompileFlags) ([]byte, error) {
				got.Conf = conf
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

			if tc.outReport != nil {
				got.OutReport = C.GoString(*tc.outReport)
			}

			// ensure SourceMetricsJSON is not nil for better comparison
			if got.Flags.SourceMetricsJSON == nil {
				got.Flags.SourceMetricsJSON = []byte{}
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
				force:   C.bool(true),
				dry_run: C.bool(true),
			},
		},

		// Error case
		"Error is returned": {
			mockErr: errors.New("error string"),
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
			err: errors.New("error string"),
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
			err: errors.New("error string"),
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

// TestLogCallbackImpl tests that the log callback is correctly invoked.
func TestLogCallbackImpl(t *testing.T) {
	t.Parallel()

	C.insights_set_log_callback(C.get_test_callback())

	tests := map[string]struct {
		logFunc func(l *slog.Logger)
	}{
		"Single call": {
			logFunc: func(l *slog.Logger) {
				l.Info("info message", "key", "val")
			},
		},
		"Multiple calls mixed levels": {
			logFunc: func(l *slog.Logger) {
				l.Error("first error")
				l.Info("then info")
				l.Warn("finally warn", "code", 123)
			},
		},
		"Debug logs are captured": {
			logFunc: func(l *slog.Logger) {
				l.Debug("debug details", "id", 123)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Lock this parallel test to an OS thread so that C._Thread_local storage is consistent
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			C.reset_test_callback()
			defer C.reset_test_callback() // Ensure memory is freed when test finishes

			// Mock collector that just logs
			mockCollector := func(conf insights.Config, source string, flags insights.CollectFlags) ([]byte, error) {
				tc.logFunc(conf.Logger)
				return []byte("report"), nil
			}

			var outReport *C.char
			collectCustomInsights(nil, nil, nil, &outReport, mockCollector)

			logs := C.GoString(C.get_test_cb_buffer())
			assert.False(t, bool(C.get_test_cb_buf_exceeded()), "Log buffer should not have exceeded max size")
			want := testutils.LoadWithUpdateFromGolden(t, logs)
			assert.Equal(t, want, logs)
		})
	}
}
