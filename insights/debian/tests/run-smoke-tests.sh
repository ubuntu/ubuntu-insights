#!/usr/bin/env bash

set -exuo pipefail

# Verify man page is installed
MANPAGER="cat" man ubuntu-insights

# Verify systemd can find the units (will succeed even if inactive)
systemctl --user --no-pager list-unit-files ubuntu-insights-collect.service
systemctl --user --no-pager list-unit-files ubuntu-insights-collect.timer
systemctl --user --no-pager list-unit-files ubuntu-insights-upload.service
systemctl --user --no-pager list-unit-files ubuntu-insights-upload.timer

# Go through a typical workflow, ensuring that no errors are encountered. This should eventually be replaced with true end-to-end tests.
ubuntu-insights --version
ubuntu-insights --help
ubuntu-insights collect -df
ubuntu-insights collect
ubuntu-insights consent -s=true
ubuntu-insights collect
ubuntu-insights upload -df
