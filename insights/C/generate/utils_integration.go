//go:build tools && integrationtests

// This file is used for injecting additional logic for the integration test case.
package main

import "os"

func init() {
	integrationtests = true

	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}
}
