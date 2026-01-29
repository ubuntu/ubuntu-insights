package main

/* This file acts as a wrapper around internal C helpers in order to get around restrictions in CGo for files using //export. */

/*
#include "types.h"
#include <stdlib.h>

static insights_logger_callback global_log_callback = NULL;

__attribute__((visibility("hidden"))) void
set_log_callback_impl(insights_logger_callback callback) {
  global_log_callback = callback;
}

void call_log_callback(insights_log_level level, char *msg) {
  if (global_log_callback) {
    global_log_callback(level, msg);
  }
}

int has_log_callback() { return global_log_callback != NULL; }
*/
import "C"

func setLogCallbackImpl(callback C.insights_logger_callback) {
	C.set_log_callback_impl(callback)
}

func callLogCallback(level C.insights_log_level, msg *C.char) {
	C.call_log_callback(level, msg)
}

func hasLogCallback() bool {
	return C.has_log_callback() != 0
}
