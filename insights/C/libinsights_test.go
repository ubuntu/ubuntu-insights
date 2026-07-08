//go:build cgo

// main_test is the package for testing the C API.
package main_test

import (
	"testing"

	main "github.com/ubuntu/ubuntu-insights/insights/C"
)

// TestCollect tests C.CollectInsights.
func TestCollect(t *testing.T) {
	main.TestCollectImpl(t)
}

// TestCompile tests C.CompileInsights.
func TestCompile(t *testing.T) {
	main.TestCompileImpl(t)
}

// TestWrite tests C.WriteInsights.
func TestWrite(t *testing.T) {
	main.TestWriteImpl(t)
}

// TestUpload tests C.UploadInsights.
func TestUpload(t *testing.T) {
	main.TestUploadImpl(t)
}

// TestGetConsent tests C.GetConsentState.
func TestGetConsent(t *testing.T) {
	main.TestGetConsentImpl(t)
}

// TestSetConsent tests C.SetConsentState.
func TestSetConsent(t *testing.T) {
	main.TestSetConsentImpl(t)
}

// TestFakeMain "tests" main. This is just for coverage since main does nothing.
func TestFakeMain(t *testing.T) {
	main.TestMainImpl(t)
}

// TestLogCallback tests the C logging callback integration.
func TestLogCallback(t *testing.T) {
	main.TestLogCallbackImpl(t)
}

// TestGetSystemOptOut tests the C insights_get_system_opt_out_state binding.
func TestGetSystemOptOut(t *testing.T) {
	main.TestGetSystemOptOutImpl(t)
}

// TestSetSystemOptOut tests the C insights_set_system_opt_out_state binding.
func TestSetSystemOptOut(t *testing.T) {
	main.TestSetSystemOptOutImpl(t)
}
