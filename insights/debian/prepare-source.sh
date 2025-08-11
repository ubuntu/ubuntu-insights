#!/bin/bash
set -eu

export GOWORK=off

is_source_build=$(git status > /dev/null 2>&1 && echo "1" || true)

# Handle vendoring
if [ -n "${is_source_build}" ]; then
    # Ensure sources are clean
    rm -r generated &> /dev/null || true
    # Handle vendoring
    rm -r vendor &> /dev/null || true
    go mod vendor
fi

echo "Source Prepared"
