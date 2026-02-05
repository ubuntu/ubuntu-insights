//go:build !system_lib

package libinsights

// This file defines additional logic for the case when we want to run integration tests while linking against a locally generated libinsights with
// modifications meant for integration tests. This should NOT be used for autopkgtest or other system-lib based testing.

/*
#cgo CFLAGS: -I${SRCDIR}/generated
#cgo linux LDFLAGS: -L${SRCDIR}/generated -l:libinsights.so.0 -Wl,-rpath,${SRCDIR}/generated
#cgo darwin LDFLAGS: ${SRCDIR}/generated/libinsights.0.dylib -Wl,-rpath,${SRCDIR}/generated
#cgo windows LDFLAGS: -L${SRCDIR}/generated -l:libinsights.dll

#include "insights.h"
*/
import "C"
import "unsafe"

func setTestServerURL(url string) {
	cstr := C.CString(url)
	defer C.free(unsafe.Pointer(cstr))
	C.insights_set_integration_test_server_url(cstr)
}
