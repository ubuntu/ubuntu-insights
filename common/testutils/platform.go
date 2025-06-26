package testutils

import (
	"os"
	"runtime"
)

// IsUnixNonRoot returns true if the current operating system is Unix-like and not running as root.
func IsUnixNonRoot() bool {
	if !IsUnix() {
		return false
	}
	return os.Getuid() != 0
}

// IsUnix returns true if the current operating system is Unix-like.
func IsUnix() bool {
	if o := runtime.GOOS; o == "linux" || o == "darwin" {
		return true
	}
	return false
}
