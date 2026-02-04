#ifndef WAYLAND_DISPLAYS_H
#define WAYLAND_DISPLAYS_H

#include <stdbool.h>
#include <stdint.h>

// Structure representing the wayland display information
struct wayland_display {
  int32_t width;
  int32_t height;
  int32_t refresh;
  int32_t phys_width;
  int32_t phys_height;
};

// Initialize Wayland display information.
int init_wayland();

// Cleanup Wayland display information.
void cleanup();

// Get the Wayland display information.
struct wayland_display **get_displays();

// Get the number of Wayland displays.
int get_output_count();

// Checks if there was a memory error.
bool had_memory_error();

#endif  // WAYLAND_DISPLAYS_H
