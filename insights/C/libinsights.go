// main is the package for the C API.
package main

/*
#include "types.h"
#include <stdlib.h>
*/
import "C"

import (
	"log/slog"
	"os"
	"unsafe"

	"github.com/ubuntu/ubuntu-insights/insights"
)

/**
 * insights_collect creates a report for the specified source.
 * If config is NULL, defaults are used.
 * source may be NULL or "" to use platform default.
 * flags may be NULL.
 * If collection fails, an error string is returned.
 * Otherwise, this returns NULL.
 *
 * If out_report is not NULL,
 * the pretty printed report is returned in out_report
 * as a null-terminated C string.
 * Note that this return may not match with what is written
 * to disk depending on provided flags and the consent state.
 * If the consent state is determined to be false, an OptOut report
 * will be written to disk, but the full compiled report will still be returned.
 *
 * The out_report must be freed by the caller.
 * The error string must be freed.
 **/
//export insights_collect
func insights_collect(config *C.insights_const_config, source *C.insights_const_char, flags *C.insights_const_collect_flags, out_report **C.char) *C.char { //nolint:revive // Exported for C
	return collectCustomInsights(config, source, flags, out_report, func(conf insights.Config, source string, f insights.CollectFlags) ([]byte, error) {
		return conf.Collect(source, f)
	})
}

// collector is a function that collects using the given parameters.
type collector = func(conf insights.Config, source string, flags insights.CollectFlags) ([]byte, error)

// collectCustomInsights handles C to Go translation and calls the custom collector.
func collectCustomInsights(config *C.insights_const_config, source *C.insights_const_char, flags *C.insights_const_collect_flags, outReport **C.char, customCollector collector) *C.char {
	conf := toGoInsightsConfig(config)

	f := insights.CollectFlags{}
	if flags != nil {
		f.Period = (uint32)(flags.period)
		f.Force = (bool)(flags.force)
		f.DryRun = (bool)(flags.dry_run)

		if flags.source_metrics_path != nil {
			f.SourceMetricsPath = C.GoString(flags.source_metrics_path)
		}
		if flags.source_metrics_json != nil && flags.source_metrics_json_len > 0 {
			f.SourceMetricsJSON = C.GoBytes(flags.source_metrics_json, C.int(flags.source_metrics_json_len))
		}
	}

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	report, err := customCollector(conf, sourceStr, f)
	if err != nil {
		if outReport != nil {
			*outReport = nil
		}
		return errToCString(err)
	}

	if outReport == nil {
		// avoid leaking memory or writing to nil, assume no need to return report
		return nil
	}

	if len(report) == 0 {
		*outReport = nil
		return nil
	}

	*outReport = C.CString(string(report))
	return nil
}

/**
 * insights_compile compiles the report for the specified source.
 * If config is NULL, defaults are used.
 * If "flags" is NULL, defaults are used.
 * If compilation fails, an error string is returned.
 * Otherwise, this returns NULL.
 *
 * If out_report is not NULL, the pretty printed report is
 * returned via out_report as a null-terminated C string.
 *
 * The out_report must be freed by the caller.
 * The error string must be freed.
 **/
//export insights_compile
func insights_compile(config *C.insights_const_config, flags *C.insights_const_compile_flags, out_report **C.char) *C.char { //nolint:revive // Exported for C
	return compileCustomInsights(config, flags, out_report, func(conf insights.Config, flags insights.CompileFlags) ([]byte, error) {
		return conf.Compile(flags)
	})
}

type compiler = func(conf insights.Config, flags insights.CompileFlags) ([]byte, error)

func compileCustomInsights(config *C.insights_const_config, flags *C.insights_const_compile_flags, outReport **C.char, customCompiler compiler) *C.char {
	conf := toGoInsightsConfig(config)

	f := insights.CompileFlags{}
	if flags != nil {
		if flags.source_metrics_path != nil {
			f.SourceMetricsPath = C.GoString(flags.source_metrics_path)
		}
		if flags.source_metrics_json != nil && flags.source_metrics_json_len > 0 {
			f.SourceMetricsJSON = C.GoBytes(flags.source_metrics_json, C.int(flags.source_metrics_json_len))
		}
	}

	report, err := customCompiler(conf, f)
	if err != nil {
		if outReport != nil {
			*outReport = nil
		}
		return errToCString(err)
	}

	if outReport == nil {
		// avoid leaking memory or writing to nil, assume no need to return report
		return nil
	}

	if len(report) == 0 {
		*outReport = nil
		return nil
	}

	*outReport = C.CString(string(report))
	return nil
}

/**
 * insights_write writes the report to disk based on the consent state.
 * If config is NULL, defaults are used.
 * If "source" is NULL or "" the platform default is used.
 * If "report" is not a valid Insights report, an error string is returned.
 * If "flags" is NULL, defaults are used.
 * If writing fails, an error string is returned.
 * Any error string returned must be freed.
 **/
//export insights_write
func insights_write(config *C.insights_const_config, source *C.insights_const_char, report *C.insights_const_char, flags *C.insights_const_write_flags) *C.char {
	return writeCustomInsights(config, source, report, flags, func(conf insights.Config, source string, report []byte, flags insights.WriteFlags) error {
		return conf.Write(source, report, flags)
	})
}

type writer = func(conf insights.Config, source string, report []byte, flags insights.WriteFlags) error

func writeCustomInsights(config *C.insights_const_config, source *C.insights_const_char, report *C.insights_const_char, flags *C.insights_const_write_flags,
	customWriter writer) *C.char {
	conf := toGoInsightsConfig(config)

	f := insights.WriteFlags{}
	if flags != nil {
		f.Period = (uint32)(flags.period)
		f.Force = (bool)(flags.force)
		f.DryRun = (bool)(flags.dry_run)
	}

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	bReport := []byte{}
	if report != nil {
		bReport = []byte(C.GoString(report))
	}

	err := customWriter(conf, sourceStr, bReport, f)
	if err != nil {
		return errToCString(err)
	}

	return nil
}

/**
 * insights_upload uploads reports for the specified sources.
 * If config is NULL, defaults are used.
 * sources may be NULL or empty to handle all reports.
 * sourcesLen is the number of sources in the array.
 * flags may be NULL.
 * If uploading fails, an error string is returned.
 * Otherwise, this returns NULL.
 * The error string must be freed.
 **/
//export insights_upload
func insights_upload(config *C.insights_const_config, sources **C.insights_const_char, sources_len C.size_t, flags *C.insights_const_upload_flags) *C.char { //nolint:revive // Exported for C
	return uploadCustomInsights(config, sources, sources_len, flags, func(conf insights.Config, sources []string, f insights.UploadFlags) error {
		return conf.Upload(sources, f)
	})
}

// uploader is a function that uploads using the given parameters.
type uploader = func(conf insights.Config, sources []string, flags insights.UploadFlags) error

// uploadCustomInsights handles C to Go translation and calls the custom uploader.
func uploadCustomInsights(config *C.insights_const_config, sources **C.insights_const_char, sourcesLen C.size_t, flags *C.insights_const_upload_flags, customUploader uploader) *C.char {
	conf := toGoInsightsConfig(config)
	// Convert C string array to Go slice
	var sourceSlice []string
	if sources != nil && sourcesLen > 0 {
		sourceSlice = make([]string, sourcesLen)
		// Convert the C array of C.char pointers to a Go slice of strings.
		//   1. unsafe.Pointer(sources) - Convert C double pointer to Go's generic unsafe pointer
		//   2. (*[1 << 28]*C.char)(...) - Cast to pointer to a huge Go array of *C.char pointers
		//      - This doesn't allocate memory, just reinterprets the existing C memory layout
		//   3. [:sourcesLen:sourcesLen] - Slice the array to exactly sourcesLen elements
		//      - First sourcesLen is the length (number of accessible elements)
		//      - Second sourcesLen is the capacity (prevents slice from growing beyond this)
		for i, s := range (*[1 << 28]*C.char)(unsafe.Pointer(sources))[:sourcesLen:sourcesLen] {
			if s != nil {
				sourceSlice[i] = C.GoString(s)
			}
		}
	}

	f := insights.UploadFlags{}
	if flags != nil {
		f.MinAge = (uint32)(flags.min_age)
		f.Force = (bool)(flags.force)
		f.DryRun = (bool)(flags.dry_run)
	}

	err := customUploader(conf, sourceSlice, f)
	return errToCString(err)
}

/**
 * insights_get_consent_state gets the consent state for the specified source.
 * If config is NULL, defaults are used.
 * source may be NULL or "" to retrieve the default consent state.
 * If it could not be retrieved, this function returns CONSENT_UNKNOWN.
 * Otherwise, it returns the consent state of the source.
 **/
//export insights_get_consent_state
func insights_get_consent_state(config *C.insights_const_config, source *C.insights_const_char) C.insights_consent_state {
	return getCustomConsentState(config, source, func(conf insights.Config, source string) C.insights_consent_state {
		s, err := conf.GetConsentState(source)
		if err != nil {
			return C.INSIGHTS_CONSENT_UNKNOWN
		}
		if s {
			return C.INSIGHTS_CONSENT_TRUE
		}
		return C.INSIGHTS_CONSENT_FALSE
	})
}

// consentGetter is a function that gets the consent state using the given parameters.
type consentGetter = func(conf insights.Config, source string) C.insights_consent_state

// getCustomConsentState handles C to Go translation and calls the custom getter.
func getCustomConsentState(config *C.insights_const_config, source *C.insights_const_char, getter consentGetter) C.insights_consent_state {
	conf := toGoInsightsConfig(config)

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	return getter(conf, sourceStr)
}

/**
 * insights_set_consent_state sets the state for the specified source to newState.
 * If config is NULL, defaults are used.
 * source may be NULL or "" to affect the default consent state.
 * If the state could not be set, this function returns an error string.
 * Otherwise, it returns NULL
 * The error string must be freed.
 **/
//export insights_set_consent_state
func insights_set_consent_state(config *C.insights_const_config, source *C.insights_const_char, new_state C.bool) *C.char { //nolint:revive // Exported for C
	return setCustomConsentState(config, source, new_state, func(conf insights.Config, source string, newState bool) error {
		return conf.SetConsentState(source, newState)
	})
}

/**
 * insights_set_log_callback sets the callback function for logging.
 * The callback receives the log level and the null terminated message.
 * Setting the callback overrides the default logging behavior.
 * Pass NULL to reset to default behavior.
 *
 * The callback may be called concurrently from multiple threads.
 * The implementation must be thread-safe.
 */
//export insights_set_log_callback
func insights_set_log_callback(callback C.insights_logger_callback) {
	setLogCallbackImpl(callback)
}

// consentSetter is a function that sets the consent state using the given parameters.
type consentSetter = func(conf insights.Config, source string, newState bool) error

// setCustomConsentState handles C to Go translation and calls the custom setter.
func setCustomConsentState(config *C.insights_const_config, source *C.insights_const_char, newState C.bool, setter consentSetter) *C.char {
	conf := toGoInsightsConfig(config)

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	err := setter(conf, sourceStr, (bool)(newState))
	return errToCString(err)
}

// toGoInsightsConfig converts a C Insights Config into the equivalent Go structure.
func toGoInsightsConfig(config *C.insights_const_config) insights.Config {
	iConf := insights.Config{Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))}
	if hasLogCallback() {
		// If a callback is registered, use it with Debug level (capture all)
		loggerOut := func(l slog.Level, msg string) {
			level := C.INSIGHTS_LOG_ERROR
			switch l {
			case slog.LevelDebug:
				level = C.INSIGHTS_LOG_DEBUG
			case slog.LevelInfo:
				level = C.INSIGHTS_LOG_INFO
			case slog.LevelWarn:
				level = C.INSIGHTS_LOG_WARN
			case slog.LevelError:
				level = C.INSIGHTS_LOG_ERROR
			}

			cMsg := C.CString(msg)
			defer C.free(unsafe.Pointer(cMsg))
			callLogCallback((C.insights_log_level)(level), cMsg)
		}
		handler := NewCLogHandler(slog.HandlerOptions{Level: slog.LevelDebug}, loggerOut)
		iConf.Logger = slog.New(handler)
	} else if config != nil {
		if config.verbose {
			iConf.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	}

	if config != nil {
		if config.consent_dir != nil {
			iConf.ConsentDir = C.GoString(config.consent_dir)
		}
		if config.insights_dir != nil {
			iConf.InsightsDir = C.GoString(config.insights_dir)
		}
	}
	return iConf
}

func errToCString(err error) *C.char {
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

// main to appease Go.
func main() {}
