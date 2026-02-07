#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdnoreturn.h>
#include <string.h>

#ifdef SYSTEM_LIB
#include <insights/insights.h>
#include <insights/types.h>
#else
#include "insights.h"
#include "types.h"
#endif

static FILE* g_log_file = NULL;

void log_callback(insights_log_level level, const char* msg) {
  if (g_log_file) {
    fprintf(g_log_file, "[LIBINSIGHTS][%d] %s\n", level, msg);
    fflush(g_log_file);
  }
}

noreturn void fail(const char* fmt, ...) {
  va_list args;
  va_start(args, fmt);
  vfprintf(stderr, fmt, args);
  va_end(args);
  fprintf(stderr, "\n");
  exit(EXIT_FAILURE);
}

noreturn void usage(const char* prog_name) {
  fprintf(stderr, "Usage: %s <global-options> <command> <args>\n", prog_name);
  exit(EXIT_FAILURE);
}

bool parse_bool(const char* str) {
  return strcmp(str, "true") == 0 || strcmp(str, "1") == 0;
}

char* read_file(const char* path) {
  FILE* f = fopen(path, "rb");
  if (!f) return NULL;

  if (fseek(f, 0, SEEK_END) != 0) {
    fclose(f);
    return NULL;
  }
  long length = ftell(f);
  if (length < 0) {
    fclose(f);
    return NULL;
  }
  rewind(f);

  char* buf = malloc(length + 1);
  if (!buf) {
    fclose(f);
    return NULL;
  }

  if (fread(buf, 1, length, f) != (size_t)length) {
    free(buf);
    fclose(f);
    return NULL;
  }
  buf[length] = '\0';
  fclose(f);
  return buf;
}

void print_report(const char* report) {
  if (report) {
    printf("REPORT_START\n%s\nREPORT_END\n", report);
  }
}

// --- Command Handlers ---

int cmd_set_consent(int argc, char** argv, int idx, insights_config* cfg) {
  if (idx + 2 > argc) fail("Missing args for set-consent: <source> <state>");
  const char* source = argv[idx++];
  bool state = parse_bool(argv[idx++]);

  char* err = insights_set_consent_state(cfg, source, state);
  if (err) {
    fprintf(stderr, "Error: %s\n", err);
    free(err);
    return 1;
  }
  return 0;
}

int cmd_get_consent(int argc, char** argv, int idx, insights_config* cfg) {
  if (idx + 1 > argc) fail("Missing args for get-consent: <source>");
  const char* source = argv[idx++];

  insights_consent_state state = insights_get_consent_state(cfg, source);
  printf("%d\n", state);
  return 0;
}

int cmd_collect(int argc, char** argv, int idx, insights_config* cfg) {
  if (idx + 1 > argc) fail("Missing args for collect: <source>");
  const char* source = argv[idx++];

  insights_collect_flags flags = {.dry_run = false, .force = false};
  bool should_print_report = false;

  while (idx < argc) {
    if (strcmp(argv[idx], "--dry-run") == 0)
      flags.dry_run = true;
    else if (strcmp(argv[idx], "--force") == 0)
      flags.force = true;
    else if (strcmp(argv[idx], "--print-report") == 0)
      should_print_report = true;
    else if (strcmp(argv[idx], "--source-metrics") == 0) {
      if (++idx >= argc) fail("Missing value for --source-metrics");
      flags.source_metrics_path = argv[idx];
    }
    idx++;
  }

  char* report = NULL;
  char* err = insights_collect(cfg, source, &flags, &report);
  if (err) {
    fprintf(stderr, "Error: %s\n", err);
    free(err);
    return 1;
  }
  if (should_print_report) {
    print_report(report);
  }
  free(report);
  return 0;
}

int cmd_compile(int argc, char** argv, int idx, insights_config* cfg) {
  insights_compile_flags flags = {0};
  bool should_print_report = false;

  while (idx < argc) {
    if (strcmp(argv[idx], "--print-report") == 0)
      should_print_report = true;
    else if (strcmp(argv[idx], "--source-metrics") == 0) {
      if (++idx >= argc) fail("Missing value for --source-metrics");
      flags.source_metrics_path = argv[idx];
    }
    idx++;
  }

  char* report = NULL;
  char* err = insights_compile(cfg, &flags, &report);
  if (err) {
    fprintf(stderr, "Error: %s\n", err);
    free(err);
    return 1;
  }
  if (should_print_report) {
    print_report(report);
  }
  free(report);
  return 0;
}

int cmd_write(int argc, char** argv, int idx, insights_config* cfg) {
  if (idx + 2 > argc) fail("Missing args for write: <source> <report_path>");
  const char* source = argv[idx++];
  const char* report_path = argv[idx++];

  char* report_content = read_file(report_path);
  if (!report_content) fail("Failed to read report file: %s", report_path);

  insights_write_flags flags = {.dry_run = false, .force = false};
  while (idx < argc) {
    if (strcmp(argv[idx], "--dry-run") == 0)
      flags.dry_run = true;
    else if (strcmp(argv[idx], "--force") == 0)
      flags.force = true;
    idx++;
  }

  char* err = insights_write(cfg, source, report_content, &flags);
  free(report_content);
  if (err) {
    fprintf(stderr, "Error: %s\n", err);
    free(err);
    return 1;
  }
  return 0;
}

int cmd_upload(int argc, char** argv, int idx, insights_config* cfg) {
  if (idx + 1 > argc) fail("Missing args for upload");

  const char* sources[50];
  size_t sources_len = 0;

  // Parse non-flag arguments as sources
  while (idx < argc && argv[idx][0] != '-') {
    if (sources_len >= 50) fail("Too many sources specified (max 50)");
    sources[sources_len++] = argv[idx++];
  }

  insights_upload_flags flags = {
      .dry_run = false, .force = false, .min_age = 0};

  while (idx < argc) {
    if (strcmp(argv[idx], "--dry-run") == 0)
      flags.dry_run = true;
    else if (strcmp(argv[idx], "--force") == 0)
      flags.force = true;
    else if (strcmp(argv[idx], "--min-age") == 0) {
      if (++idx >= argc) fail("Missing value for --min-age");
      char* endptr;
      long val = strtol(argv[idx], &endptr, 10);
      if (*endptr != '\0' || endptr == argv[idx])
        fail("Invalid integer for --min-age: %s", argv[idx]);
      flags.min_age = (int)val;
    }
    idx++;
  }

  char* err = insights_upload(cfg, sources, sources_len, &flags);
  if (err) {
    fprintf(stderr, "Error: %s\n", err);
    free(err);
    return 1;
  }
  return 0;
}

// --- Main Dispatch ---

typedef struct {
  const char* name;
  int (*fn)(int argc, char** argv, int idx, insights_config* cfg);
} cmd_entry;

int main(int argc, char** argv) {
  if (argc < 2) usage(argv[0]);

  insights_config config = {
      .consent_dir = NULL, .insights_dir = NULL, .verbose = true};
  const char* log_file_path = NULL;
  int idx = 1;  // Start after prog name

  // Parse global options
  while (idx < argc && argv[idx][0] == '-') {
    if (strcmp(argv[idx], "--consent-dir") == 0) {
      if (++idx >= argc) fail("Missing value for --consent-dir");
      config.consent_dir = argv[idx];
    } else if (strcmp(argv[idx], "--insights-dir") == 0) {
      if (++idx >= argc) fail("Missing value for --insights-dir");
      config.insights_dir = argv[idx];
    } else if (strcmp(argv[idx], "--log-file") == 0) {
      if (++idx >= argc) fail("Missing value for --log-file");
      log_file_path = argv[idx];
    } else {
      // Found a non-global flag or command
      break;
    }
    idx++;
  }

  // Set up logging if requested
  if (log_file_path) {
    g_log_file = fopen(log_file_path, "a");
    if (!g_log_file) {
      perror("Failed to open log file");
      return 1;
    }
    insights_set_log_callback(log_callback);
  }

  if (idx >= argc) {
    fprintf(stderr, "No command specified\n");
    if (g_log_file) fclose(g_log_file);
    return 1;
  }

  const char* cmd_name = argv[idx++];

  // Command dispatch table
  cmd_entry commands[] = {{"set-consent", cmd_set_consent},
                          {"get-consent", cmd_get_consent},
                          {"collect", cmd_collect},
                          {"compile", cmd_compile},
                          {"write", cmd_write},
                          {"upload", cmd_upload},
                          {NULL, NULL}};

  int result = 1;
  bool found = false;
  for (int i = 0; commands[i].name != NULL; i++) {
    if (strcmp(cmd_name, commands[i].name) == 0) {
      result = commands[i].fn(argc, argv, idx, &config);
      found = true;
      break;
    }
  }

  if (!found) {
    fprintf(stderr, "Unknown command: %s\n", cmd_name);
  }

  if (g_log_file) fclose(g_log_file);
  return result;
}
