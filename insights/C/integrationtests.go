//go:build integrationtests && !system_lib

package main

import (
	"os"

	uploadertestutils "github.com/ubuntu/ubuntu-insights/insights/internal/uploader/testutils"
)

func init() {
	if url := os.Getenv("INSIGHTS_TEST_UPLOAD_URL"); url != "" {
		uploadertestutils.SetServerURL(url)
	}

	if os.Getenv("INSIGHTS_TEST_MAKE_PANIC") == "true" {
		panic("Intentional panic triggered by INSIGHTS_TEST_MAKE_PANIC environment variable")
	}
}
