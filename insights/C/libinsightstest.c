#include "libinsightstest.h"
#include "types.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Test helpers
// Requires C11 or later for _Thread_local

#define TEST_CB_MAX_SIZE 4096 // 4KB

typedef struct {
  int count;
  char buf[TEST_CB_MAX_SIZE];
  size_t size;       // Current string length
  bool buf_exceeded; // Flag to indicate buffer overflow
} test_cb_state_t;

static _Thread_local test_cb_state_t test_cb_state = {0, {0}, 0, false};

static void append_log(const char *str, size_t len) {
  if (str == NULL)
    return;

  if ((test_cb_state.size + len + 1 > TEST_CB_MAX_SIZE) ||
      test_cb_state.buf_exceeded) {
    test_cb_state.buf_exceeded = true;
    return;
  }

  memcpy(test_cb_state.buf + test_cb_state.size, str, len);
  test_cb_state.size += len;
  test_cb_state.buf[test_cb_state.size] = '\0';
}

void test_log_callback_fn(insights_log_level level, const char *msg) {
  test_cb_state.count++;

  const char *lvlStr = "UNKNOWN";
  switch (level) {
  case INSIGHTS_LOG_ERROR:
    lvlStr = "ERROR";
    break;
  case INSIGHTS_LOG_WARN:
    lvlStr = "WARN";
    break;
  case INSIGHTS_LOG_INFO:
    lvlStr = "INFO";
    break;
  case INSIGHTS_LOG_DEBUG:
    lvlStr = "DEBUG";
    break;
  }

  if (msg != NULL) {
    char line[TEST_CB_MAX_SIZE];
    int ret = snprintf(line, sizeof(line), "[%s] %s\n", lvlStr, msg);
    if (ret < 0 || (size_t)ret >= sizeof(line)) {
      test_cb_state.buf_exceeded = true;
      return;
    }
    append_log(line, (size_t)ret);
  }
}

insights_logger_callback get_test_callback() { return test_log_callback_fn; }

void reset_test_callback() {
  test_cb_state.count = 0;
  test_cb_state.size = 0;
  test_cb_state.buf[0] = '\0';
  test_cb_state.buf_exceeded = false;
}

int get_test_cb_count() { return test_cb_state.count; }

char *get_test_cb_buffer() { return test_cb_state.buf; }

bool get_test_cb_buf_exceeded() { return test_cb_state.buf_exceeded; }
