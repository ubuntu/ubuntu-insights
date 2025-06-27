#include "wayland_displays_linux.h"
#include "wayland_displays_linux_test.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>

static bool memory_error = false;

static struct wayland_display **displays = NULL;
static struct wl_output **outputs = NULL;

static size_t count = 0;
static size_t capacity = 4;

static struct wl_display *display;
static struct wl_registry *registry;

// Function to handle geometry events from the Wayland output
void handle_geometry(void *data, struct wl_output *output, int32_t x, int32_t y,
                     int32_t physical_width, int32_t physical_height,
                     int32_t subpixel, const char *make, const char *model,
                     int32_t transform) {
  struct wayland_display *info = (struct wayland_display *)data;
  info->phys_width = physical_width;
  info->phys_height = physical_height;
}

// Function to handle mode events from the Wayland output
void handle_mode(void *data, struct wl_output *output, uint32_t flags,
                 int32_t width, int32_t height, int32_t refresh) {
  if (flags & WL_OUTPUT_MODE_CURRENT) {
    struct wayland_display *info = (struct wayland_display *)data;
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

// Function to handle global events from the Wayland registry
void global_handler(void *data, struct wl_registry *registry, uint32_t name,
                    const char *interface, uint32_t version) {
  if (strcmp(interface, "wl_output") != 0) {
    return;
  }
  // Check if we need to increase the capacity of the arrays
  if (count >= capacity) {
    capacity *= 2;
    struct wl_output **new_outputs =
        realloc(outputs, capacity * sizeof(struct wl_output *));

    struct wayland_display **new_displays =
        realloc(displays, capacity * sizeof(struct wayland_display *));
    if (!new_outputs || !new_displays) {
      free(new_outputs);
      free(new_displays);
      new_displays = NULL;
      new_outputs = NULL;
      memory_error = true;
      return;
    }
    outputs = new_outputs;
    displays = new_displays;
  }

  struct wayland_display *display = malloc(sizeof(struct wayland_display));
  if (!display) {
    memory_error = true;
    return;
  }
  memset(display, 0, sizeof(struct wayland_display));

  struct wl_output *output =
      wl_registry_bind(registry, name, &wl_output_interface, 1);

  displays[count] = display;
  outputs[count] = output;
  count++;

  wl_output_add_listener(output, &output_listener, display);
}

void global_remove(void *data, struct wl_registry *registry, uint32_t name) {}

static const struct wl_registry_listener registry_listener = {
    .global = global_handler,
    .global_remove = global_remove,
};

int init_wayland() {
  outputs = malloc(capacity * sizeof(struct wl_output *));
  displays = malloc(capacity * sizeof(struct wayland_display *));

  if (!outputs || !displays) {
    free(outputs);
    free(displays);
    outputs = NULL;
    displays = NULL;
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
  if (wl_display_roundtrip(display) < 0) {
    cleanup();
    return -1;
  }
  if (wl_display_dispatch(display) < 0) {
    cleanup();
    return -1;
  }
  return 0;
}

void cleanup() {
  if (outputs) {
    for (size_t i = 0; i < count; i++) {
      if (outputs[i]) {
        wl_output_destroy(outputs[i]);
      }
    }
  }

  free(outputs);
  outputs = NULL;

  if (registry) {
    wl_registry_destroy(registry);
    registry = NULL;
  }
  if (display) {
    wl_display_disconnect(display);
    display = NULL;
  }

  for (size_t i = 0; i < count; i++) {
    free(displays[i]);
  }
  free(displays);
  displays = NULL;
  count = 0;
  capacity = 4;
  memory_error = false;
}

struct wayland_display **get_displays() { return displays; }

int get_output_count() { return count; }
bool had_memory_error() { return memory_error; }

void set_displays(struct wayland_display **new_displays, int c) {
  cleanup();
  displays = new_displays;
  count = capacity = c;
}

void set_memory_error(bool error) { memory_error = error; }
