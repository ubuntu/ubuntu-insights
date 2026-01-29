#ifndef LIBINSIGHTSTEST_IMPL_H
#define LIBINSIGHTSTEST_IMPL_H

#include "types.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// External functions from libinsights
extern char *insights_collect(const insights_config *, const char *,
                              const insights_collect_flags *, char **);
extern char *insights_compile(const insights_config *,
                              const insights_compile_flags *, char **);
extern char *insights_write(const insights_config *, const char *, const char *,
                            const insights_write_flags *);
extern char *insights_upload(const insights_config *, const char **, size_t,
                             const insights_upload_flags *);
extern insights_consent_state
insights_get_consent_state(const insights_config *, const char *);
extern char *insights_set_consent_state(const insights_config *, const char *,
                                        bool);
extern void insights_set_log_callback(insights_logger_callback);

// Test helpers
insights_logger_callback get_test_callback();
void reset_test_callback();
int get_test_cb_count();
char *get_test_cb_buffer();
bool get_test_cb_buf_exceeded();

#endif
