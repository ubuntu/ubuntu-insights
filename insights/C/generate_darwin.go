// main is the package for the C API.
package main

// generate shared library and header.
//go:generate go build -o ../generated/libinsights.so -buildmode=c-shared "libinsights.go" "log_handler.go" "internal.go"

// Copy types.h to the generated folder
//go:generate cp types.h ../generated/types.h
