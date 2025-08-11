#ifndef INSIGHTS_TYPES_H
#define INSIGHTS_TYPES_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

typedef enum {
  CONSENT_UNKNOWN = -1,
  CONSENT_FALSE = 0,
  CONSENT_TRUE = 1,
} ConsentState;

typedef struct {
  const char *consentDir;  // default: "${os.UserConfigDir}/ubuntu-insights"
  const char *insightsDir; // default: "${os.UserCacheDir}/ubuntu-insights"
  bool verbose;            // Debug if true, info otherwise (default: false)
} InsightsConfig;

/**
 * @brief Parameters for insights collection.
 *
 * @note sourceMetricsPath and sourceMetricsJSON are mutually exclusive.
 */
typedef struct {
  const char *sourceMetricsPath; // Path to JSON file (default: empty)
  const void *sourceMetricsJSON; // Raw JSON data as bytes (default: NULL)
  size_t sourceMetricsJSONLen;   // Length of sourceMetricsJSON in bytes
  uint32_t period;               // Collection period in seconds (default: 0)
  bool force;  // Force collection, ignoring duplicates (default: false)
  bool dryRun; // Simulate operation without writing files (default: false)
} CollectFlags;

typedef struct {
  uint32_t minAge;    // default: 1
  bool force, dryRun; // default: false
} UploadFlags;

// typedefs to be able to have `const` in Go.
typedef const char Cchar;
typedef const InsightsConfig CInsightsConfig;
typedef const CollectFlags CCollectFlags;
typedef const UploadFlags CUploadFlags;

#endif // INSIGHTS_TYPES_H
