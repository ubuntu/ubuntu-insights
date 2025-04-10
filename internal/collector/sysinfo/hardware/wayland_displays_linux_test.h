#ifndef WAYLAND_DISPLAYS_TEST_H
#define WAYLAND_DISPLAYS_TEST_H
#include <stdbool.h>

// Setter function to set displays for testing purposes
void set_displays(struct wayland_display **displays, int count);

// Setter function to set memory error for testing purposes
void set_memory_error(bool error);

#endif // WAYLAND_DISPLAYS_TEST_H
