#ifndef INSIGHTS_TYPES_H
#define INSIGHTS_TYPES_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

typedef enum {
  INSIGHTS_CONSENT_UNKNOWN = -1,
  INSIGHTS_CONSENT_FALSE = 0,
  INSIGHTS_CONSENT_TRUE = 1,
} insights_consent_state;

typedef enum {
  INSIGHTS_LOG_ERROR = 0,
  INSIGHTS_LOG_WARN = 1,
  INSIGHTS_LOG_INFO = 2,
  INSIGHTS_LOG_DEBUG = 3,
} insights_log_level;

typedef void (*insights_logger_callback)(insights_log_level level,
                                         const char *msg);

typedef struct {
  const char *consent_dir;  // default: "${os.UserConfigDir}/ubuntu-insights"
  const char *insights_dir; // default: "${os.UserCacheDir}/ubuntu-insights"
  bool verbose;             // Debug if true, info otherwise (default: false)
} insights_config;

/**
 * @brief Parameters for insights collection.
 *
 * @note source_metrics_path and source_metrics_json are mutually exclusive.
 */
typedef struct {
  const char *source_metrics_path; // Path to JSON file (default: empty)
  const void *source_metrics_json; // Raw JSON data as bytes (default: NULL)
  size_t source_metrics_json_len;  // Length of source_metrics_json in bytes
  uint32_t period;                 // Collection period in seconds (default: 0)
  bool force;   // Force collection, ignoring duplicates (default: false)
  bool dry_run; // Simulate operation without writing files (default: false)
} insights_collect_flags;

typedef struct {
  const char *source_metrics_path; // Path to JSON file (default: empty)
  const void *source_metrics_json; // Raw JSON data as bytes (default: NULL)
  size_t source_metrics_json_len;  // Length of source_metrics_json in bytes
} insights_compile_flags;

typedef struct {
  uint32_t period; // Collection period in seconds (default: 0)
  bool force;      // Force write, ignoring duplicates (default: false)
  bool dry_run;    // Simulate operation without writing files (default: false)
} insights_write_flags;

typedef struct {
  uint32_t min_age; // default: 1
  bool force;
  bool dry_run; // default: false
} insights_upload_flags;

// Typedefs to be able to have `const` in Go (GNU style lowercase with
// underscores).
typedef const char insights_const_char;
typedef const insights_config insights_const_config;
typedef const insights_collect_flags insights_const_collect_flags;
typedef const insights_compile_flags insights_const_compile_flags;
typedef const insights_write_flags insights_const_write_flags;
typedef const insights_upload_flags insights_const_upload_flags;

#endif // INSIGHTS_TYPES_H
