#!/usr/bin/env bash

set -exuo pipefail

export GOPROXY=off
export GOTOOLCHAIN=local

PATH=$PATH:$("$(dirname "$0")"/../get-depends-go-bin-path.sh)
export PATH

go test -tags=system_lib -v ./...
