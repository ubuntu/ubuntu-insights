package platform_test

import (
	"flag"
	"os"
	"testing"

	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

func TestMain(m *testing.M) {
	flag.Parse()
	dir, ok := testutils.SetupHelperCoverdir()

	r := m.Run()
	if ok {
		os.Remove(dir)
	}
	os.Exit(r)
}
