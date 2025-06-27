package hardware

/*
#cgo LDFLAGS: -lwayland-client
#include "wayland_displays_linux.h"
#include "wayland_displays_linux_test.h"
*/
import "C"

import (
	"testing"
	"unsafe"
)

type cWaylandDisplay struct {
	Width      int32
	Height     int32
	Refresh    int32
	PhysWidth  int32
	PhysHeight int32
}

// makeCWaylandDisplay converts a Go cWaylandDisplay struct to a C struct.
func makeCWaylandDisplay(d cWaylandDisplay) (wd *C.struct_wayland_display) {
	// Cleanup is done in C.
	wd = (*C.struct_wayland_display)(C.malloc(C.size_t(C.sizeof_struct_wayland_display)))
	wd.width = C.int(d.Width)
	wd.height = C.int(d.Height)
	wd.refresh = C.int(d.Refresh)
	wd.phys_width = C.int(d.PhysWidth)
	wd.phys_height = C.int(d.PhysHeight)
	return wd
}

func makeCwaylandDisplays(d []cWaylandDisplay) (wds **C.struct_wayland_display) {
	if len(d) == 0 {
		return nil
	}

	// Cleanup is done in C.
	wds = (**C.struct_wayland_display)(C.malloc(C.size_t(len(d)) * C.size_t(C.sizeof_struct_wayland_display)))

	// Cast the C pointer 'wds' to a Go slice of *C.struct_wayland_display.
	// We cast to a pointer to a huge array (size 1<<30) so that we guarantee enough
	// room to index any element up to len(d), valid for both 32 and 64 bit systems.
	// The huge array size doesn't cause massive allocation because we're only reinterpreting
	// the memory pointed to by 'wds'. Then, we slice the array to exactly len(d) elements.
	slice := (*[1 << 30]*C.struct_wayland_display)(unsafe.Pointer(wds))[:len(d):len(d)]
	for i, display := range d {
		wd := makeCWaylandDisplay(display)
		slice[i] = wd
	}
	return wds
}

// TestingInitWayland initializes the Wayland display for testing purposes.
func TestingInitWayland(t *testing.T, cwd []cWaylandDisplay, memoryErr bool) {
	t.Helper()

	wds := makeCwaylandDisplays(cwd)
	C.set_displays(wds, C.int(len(cwd)))
	C.set_memory_error(C.bool(memoryErr))
}
