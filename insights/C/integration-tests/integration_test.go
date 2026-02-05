package libinsights_test

import (
	"os"
	"testing"

	libinsights "github.com/ubuntu/ubuntu-insights/insights/C/integration-tests"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCollect(t *testing.T) {
	libinsights.TestCollect(t)
}

func TestCompileAndWrite(t *testing.T) {
	libinsights.TestCompileAndWrite(t)
}

func TestUpload(t *testing.T) {
	libinsights.TestUpload(t)
}

func TestSetConsent(t *testing.T) {
	libinsights.TestSetConsent(t)
}

func TestGetConsent(t *testing.T) {
	libinsights.TestGetConsent(t)
}

func TestCallback(t *testing.T) {
	libinsights.TestCallback(t)
}

// Wrapper to facilitate log capture in set log callback tests.
func TestHelperCallbackWorkWrapper(t *testing.T) {
	libinsights.TestHelperCallbackWork(t)
}
