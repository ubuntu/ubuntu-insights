//go:build !cgo

package main_test

import "testing"

func TestCgoRequired(t *testing.T) {
	t.Fatal("CGO is required for insights/C tests, but it is disabled.")
}
