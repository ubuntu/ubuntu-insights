//go:build integrationtests

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	uploadertestutils "github.com/ubuntu/ubuntu-insights/insights/internal/uploader/testutils"
)

//export insights_set_integration_test_server_url
func insights_set_integration_test_server_url(url *C.char) {
	goURL := C.GoString(url)
	uploadertestutils.SetServerURL(goURL)
}
