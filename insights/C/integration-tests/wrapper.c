#ifdef SYSTEM_LIB
#include <insights/insights.h>
#include <insights/types.h>
#else
#include "insights.h"
#include "types.h"
#endif

// Defined in integration.go via //export
extern void goLogCallback(insights_log_level level, char* msg);

void log_callback_c_wrapper(insights_log_level level, const char* msg) {
  goLogCallback(level, (char*)msg);
}
