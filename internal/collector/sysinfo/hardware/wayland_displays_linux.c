#include "wayland_displays_linux.h"

static struct wayland_display **displays = NULL;
static size_t capacity = 4;
static size_t output_count = 0;
static bool memory_error = false;

static struct wl_display *display;
static struct wl_registry *registry;

void handle_geometry(void *data, struct wl_output *output, int32_t x, int32_t y,
                     int32_t physical_width, int32_t physical_height,
                     int32_t subpixel, const char *make, const char *model,
                     int32_t transform) {
  if (output_count >= capacity) {
    capacity *= 2;
    displays = realloc(displays, capacity * sizeof(struct wayland_display *));
    if (!displays) {
      memory_error = true;
      return;
    }
  }
  struct wayland_display *info = malloc(sizeof(struct wayland_display));
  if (!info) {
    memory_error = true;
    return;
  }
  info->phys_width = physical_width;
  info->phys_height = physical_height;

  displays[output_count++] = info;
}

void handle_mode(void *data, struct wl_output *output, uint32_t flags,
                 int32_t width, int32_t height, int32_t refresh) {
  if (flags & WL_OUTPUT_MODE_CURRENT) {
    struct wayland_display *info = displays[output_count - 1];
    info->width = width;
    info->height = height;
    info->refresh = refresh;
  }
}

static const struct wl_output_listener output_listener = {
    .geometry = handle_geometry,
    .mode = handle_mode,
    .done = NULL,
    .scale = NULL,
};

void global_handler(void *data, struct wl_registry *registry, uint32_t name,
                    const char *interface, uint32_t version) {
  if (strcmp(interface, "wl_output") == 0) {
    struct wl_output *output =
        wl_registry_bind(registry, name, &wl_output_interface, 1);
    wl_output_add_listener(output, &output_listener, NULL);
  }
}

void global_remove(void *data, struct wl_registry *registry, uint32_t name) {}

static const struct wl_registry_listener registry_listener = {
    .global = global_handler,
    .global_remove = global_remove,
};

int init_wayland() {
  displays = malloc(capacity * sizeof(struct wayland_display *));
  if (!displays) {
    memory_error = true;
    return -1;
  }

  display = wl_display_connect(NULL);
  if (!display) {
    cleanup();
    return -1;
  }

  registry = wl_display_get_registry(display);
  wl_registry_add_listener(registry, &registry_listener, NULL);
  // Call twice to ensure we get all the events
  wl_display_roundtrip(display);
  wl_display_roundtrip(display);
  return 0;
}

void cleanup() {
  if (display) {
    wl_display_disconnect(display);
    display = NULL;
  }
  for (size_t i = 0; i < output_count; i++) {
    free(displays[i]);
  }
  free(displays);
  displays = NULL;
  output_count = 0;
  capacity = 4;
  memory_error = false;
}

struct wayland_display **get_displays() { return displays; }

int get_output_count() { return output_count; }
bool had_memory_error() { return memory_error; }