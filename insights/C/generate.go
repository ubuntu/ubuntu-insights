package main

// generate shared library and header.
//go:generate go run -tags=tools ./generate/...
//go:generate go run -tags=tools,integrationtests ./generate/... integration-tests/generated
