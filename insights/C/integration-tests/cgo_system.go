//go:build system_lib

package libinsights

// This file defines additional logic for the case when we want to run integration tests while linking against the system-installed libinsights.
// For example, when running tests via autopkgtest.

/*
#cgo CFLAGS: -DSYSTEM_LIB
#cgo LDFLAGS: -linsights
*/
import "C"

func init() {
	systemLib = true
}

func setTestServerURL(url string) {
	panic("System lib build does not support setting test server URL")
}
