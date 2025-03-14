package main

/*
#include <stdbool.h>

typedef enum {
	CONSENT_UNKNOWN = -1,
	CONSENT_FALSE = 0,
	CONSENT_TRUE = 1,
} ConsentState;

typedef struct {
	const char* source;      //default: global
	const char* consentDir;  //default: "${os.UserConfigDir}/ubuntu-insights"
	const char* insightsDir; //default: "${os.UserCacheDir}/ubuntu-insights"
	bool verbose;            //default: false
} InsightsConfig;

// Collector
typedef struct {
	unsigned int period; // default: 1 week (604800)
	bool force, dryRun;  // default: false
} CollectFlags;

typedef struct {
	unsigned int minAge; // default: 1
	bool force, dryRun;  // default: false
} UploadFlags;

// typedefs to be able to have `const` in Go.
typedef const char Cchar;
typedef const InsightsConfig CInsightsConfig;
typedef const CollectFlags CCollectFlags;
typedef const UploadFlags CUploadFlags;

*/
import "C"

import (
	"fmt"

	"github.com/ubuntu/ubuntu-insights/pkg/insights"
)

// collectInsights creates a report for the config->source.
// metricsPath is a filepath to a JSON file containing extra metrics.
// flags may be NULL.
// If config->source is NULL or "", the source is the platform and metricsPath is ignored.
// If metricsPath is NULL or "", an error string is returned.
// If collection fails, an error string is returned.
// Otherwise, this returns NULL.
//export collectInsights
func collectInsights(config *C.CInsightsConfig, metricsPath *C.Cchar, flags *C.CCollectFlags) *C.Cchar {
	conf := toGoInsightsConfig(config)

	mpath := ""
	if metricsPath != nil {
		mpath = C.GoString(metricsPath)
	}

	f := insights.CollectFlags{}
	if flags != nil {
		f.Period = (uint)(flags.period);
		f.Force = (bool)(flags.force);
		f.DryRun = (bool)(flags.dryRun);
	}

	err := conf.Collect(mpath, f)
	return errToCString(err)
}

// uploadInsights uploads reports for the config->source.
// flags may be NULL.
// If source is NULL or "", all reports are handled.
// If uploading fails, an error string is returned.
// Otherwise, this returns NULL.
//export uploadInsights
func uploadInsights(config *C.CInsightsConfig, flags *C.CUploadFlags) *C.Cchar {
	conf := toGoInsightsConfig(config)

	f := insights.UploadFlags{}
	if flags != nil {
		f.MinAge = (uint)(flags.minAge);
		f.Force = (bool)(flags.force);
		f.DryRun = (bool)(flags.dryRun);
	}

	err := conf.Upload(f)
	return errToCString(err)
}

// getConsentState gets the consent state for the config->source.
// If source is NULL or "", the global source is retrieved.
// If it could not be retrieved, this function returns CONSENT_UNKNOWN.
// Otherwise, it returns the consent state of the source.
//export getConsentState
func getConsentState(config *C.CInsightsConfig) C.ConsentState {
	conf := toGoInsightsConfig(config)
	return (C.ConsentState)(conf.GetConsentState())
}

// setConsentState sets the state for config->source to newState.
// If source is NULL or "", the global state if effected.
// If the state could not be set, this function returns an error string.
// Otherwise, it returns NULL
//export setConsentState
func setConsentState(config *C.CInsightsConfig, newState C.bool) *C.Cchar {
	conf := toGoInsightsConfig(config)
	err := conf.SetConsentState((bool)(newState))

	return errToCString(err)
}

// toGoInsightsConfig converts a C Insights Config into the equivalent Go structure.
func toGoInsightsConfig(config *C.CInsightsConfig) insights.Config {
	iConf := insights.Config{}
	if config != nil {
		if config.source != nil {
			iConf.Source = C.GoString(config.source)
		}
		if config.consentDir != nil {
			iConf.ConsentDir = C.GoString(config.consentDir)
		}
		if config.insightsDir != nil {
			iConf.InsightsDir = C.GoString(config.insightsDir)
		}
		iConf.Verbose = (bool)(config.verbose)
	}
	return iConf
}

func errToCString(err error) *C.Cchar {
	if err != nil {
		return C.CString(fmt.Sprintf("%v", err))
	}
	return nil
}

// main to appease Go.
func main() {}
