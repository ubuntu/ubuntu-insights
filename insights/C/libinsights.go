// main is the package for the C API.
package main

/*
#include "types.h"
*/
import "C"

import (
	"log/slog"
	"os"
	"unsafe"

	"github.com/ubuntu/ubuntu-insights/insights"
)

/* collectInsights creates a report for the specified source.
// If config is NULL, defaults are used.
// source may be NULL or "" to use platform default.
// flags may be NULL.
// If collection fails, an error string is returned.
// Otherwise, this returns NULL.
//
// If out_report is not NULL,
// the pretty printed report is returned in out_report
// as a null-terminated C string.
// Note that this return may not match with what is written
// to disk depending on provided flags and the consent state.
// If the consent state is determined to be false, an OptOut report
// will be written to disk, but the full compiled report will still be returned.
//
// The out_report must be freed by the caller.
// The error string must be freed. */
//export collectInsights
func collectInsights(config *C.CInsightsConfig, source *C.char, flags *C.CCollectFlags, out_report **C.char) *C.char { //nolint:revive // Exported for C
	return collectCustomInsights(config, source, flags, out_report, func(conf insights.Config, source string, f insights.CollectFlags) ([]byte, error) {
		return conf.Collect(source, f)
	})
}

// collector is a function that collects using the given parameters.
type collector = func(conf insights.Config, source string, flags insights.CollectFlags) ([]byte, error)

// collectCustomInsights handles C to Go translation and calls the custom collector.
func collectCustomInsights(config *C.CInsightsConfig, source *C.char, flags *C.CCollectFlags, outReport **C.char, customCollector collector) *C.char {
	conf := toGoInsightsConfig(config)

	f := insights.CollectFlags{}
	if flags != nil {
		f.Period = (uint32)(flags.period)
		f.Force = (bool)(flags.force)
		f.DryRun = (bool)(flags.dryRun)

		if flags.sourceMetricsPath != nil {
			f.SourceMetricsPath = C.GoString(flags.sourceMetricsPath)
		}
		if flags.sourceMetricsJSON != nil && flags.sourceMetricsJSONLen > 0 {
			f.SourceMetricsJSON = C.GoBytes(flags.sourceMetricsJSON, C.int(flags.sourceMetricsJSONLen))
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

/* uploadInsights uploads reports for the specified sources.
// If config is NULL, defaults are used.
// sources may be NULL or empty to handle all reports.
// sourcesLen is the number of sources in the array.
// flags may be NULL.
// If uploading fails, an error string is returned.
// Otherwise, this returns NULL.
// The error string must be freed. */
//export uploadInsights
func uploadInsights(config *C.CInsightsConfig, sources **C.char, sourcesLen C.size_t, flags *C.CUploadFlags) *C.char {
	return uploadCustomInsights(config, sources, sourcesLen, flags, func(conf insights.Config, sources []string, f insights.UploadFlags) error {
		return conf.Upload(sources, f)
	})
}

// uploader is a function that uploads using the given parameters.
type uploader = func(conf insights.Config, sources []string, flags insights.UploadFlags) error

// uploadCustomInsights handles C to Go translation and calls the custom uploader.
func uploadCustomInsights(config *C.CInsightsConfig, sources **C.char, sourcesLen C.size_t, flags *C.CUploadFlags, customUploader uploader) *C.char {
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
		f.MinAge = (uint32)(flags.minAge)
		f.Force = (bool)(flags.force)
		f.DryRun = (bool)(flags.dryRun)
	}

	err := customUploader(conf, sourceSlice, f)
	return errToCString(err)
}

/* getConsentState gets the consent state for the specified source.
// If config is NULL, defaults are used.
// source may be NULL or "" to retrieve the global source.
// If it could not be retrieved, this function returns CONSENT_UNKNOWN.
// Otherwise, it returns the consent state of the source. */
//export getConsentState
func getConsentState(config *C.CInsightsConfig, source *C.char) C.ConsentState {
	return getCustomConsentState(config, source, func(conf insights.Config, source string) C.ConsentState {
		s, err := conf.GetConsentState(source)
		if err != nil {
			return C.CONSENT_UNKNOWN
		}
		if s {
			return C.CONSENT_TRUE
		}
		return C.CONSENT_FALSE
	})
}

// consentGetter is a function that gets the consent state using the given parameters.
type consentGetter = func(conf insights.Config, source string) C.ConsentState

// getCustomConsentState handles C to Go translation and calls the custom getter.
func getCustomConsentState(config *C.CInsightsConfig, source *C.char, getter consentGetter) C.ConsentState {
	conf := toGoInsightsConfig(config)

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	return getter(conf, sourceStr)
}

/* setConsentState sets the state for the specified source to newState.
// If config is NULL, defaults are used.
// source may be NULL or "" to affect the global state.
// If the state could not be set, this function returns an error string.
// Otherwise, it returns NULL
// The error string must be freed. */
//export setConsentState
func setConsentState(config *C.CInsightsConfig, source *C.char, newState C.bool) *C.char {
	return setCustomConsentState(config, source, newState, func(conf insights.Config, source string, newState bool) error {
		return conf.SetConsentState(source, newState)
	})
}

// consentSetter is a function that sets the consent state using the given parameters.
type consentSetter = func(conf insights.Config, source string, newState bool) error

// setCustomConsentState handles C to Go translation and calls the custom setter.
func setCustomConsentState(config *C.CInsightsConfig, source *C.char, newState C.bool, setter consentSetter) *C.char {
	conf := toGoInsightsConfig(config)

	sourceStr := ""
	if source != nil {
		sourceStr = C.GoString(source)
	}

	err := setter(conf, sourceStr, (bool)(newState))
	return errToCString(err)
}

// toGoInsightsConfig converts a C Insights Config into the equivalent Go structure.
func toGoInsightsConfig(config *C.CInsightsConfig) insights.Config {
	iConf := insights.Config{Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))}
	if config != nil {
		if config.consentDir != nil {
			iConf.ConsentDir = C.GoString(config.consentDir)
		}
		if config.insightsDir != nil {
			iConf.InsightsDir = C.GoString(config.insightsDir)
		}

		if config.verbose {
			iConf.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
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
