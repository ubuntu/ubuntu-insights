// main is the package for the C API.
package main

// Make sure cgo is enabled `$env:CGO_ENABLED="1"`.
// generate shared library and header, this requires setting up a gcc compiler on windows.
//go:generate go build -o ../../build/libinsights.dll -buildmode=c-shared libinsights.go
