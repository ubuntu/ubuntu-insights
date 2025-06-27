package testutils

import (
	"path/filepath"
	"runtime"
)

// ModuleRoot returns the path to the module's root directory.
func ModuleRoot() string {
	// p is the path to the caller file, in this case {MODULE_ROOT}/internal/common/testutils/path.go
	_, p, _, _ := runtime.Caller(0)
	// Ignores the last 4 elements -> /internal/common/testutils/path.go
	for range 4 {
		p = filepath.Dir(p)
	}
	return p
}
