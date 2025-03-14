package main

// generate shared library and header.
//go:generate go build -o ../../../build/libinsights.so -buildmode=c-shared libinsights.go
