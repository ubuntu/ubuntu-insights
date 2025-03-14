package main

// generate shared library and header.
//go:generate sh -c "go build -o ../../../build/libinsights.so.1 -buildmode=c-shared -ldflags \"-extldflags -Wl,-soname,libinsights.so.1\" libinsights.go && mv ../../../build/libinsights.so.h ../../../build/libinsights.h"
