#!/bin/bash
set -eu

export GOWORK=off

is_source_build=$(git status > /dev/null 2>&1 && echo "1" || true)

# Handle vendoring
if [ -n "${is_source_build}" ]; then
    # Ensure sources are clean
    rm -r generated &> /dev/null || true
		rm -r C/integration-tests/generated &> /dev/null || true
    # Handle vendoring
    rm -r vendor &> /dev/null || true
    GOTOOLCHAIN=auto go mod vendor
fi

# Check that the vendor directory exists and is not empty to confirm vendoring succeeded.
if [ ! -d "vendor" ] || [ -z "$(ls -A vendor)" ]; then
		echo "Vendor directory not found or empty!"
		exit 1
fi

echo "Source Prepared"
