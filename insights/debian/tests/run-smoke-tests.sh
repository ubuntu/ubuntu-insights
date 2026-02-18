#!/usr/bin/env bash

set -exuo pipefail

# Verify man page is installed
MANPAGER="cat" man ubuntu-insights

# Systemd user setup
# This triggers logind to create /run/user/UID and start 'systemd --user'
sudo loginctl enable-linger "$(id -u)"

# Point to the directory logind just created
XDG_RUNTIME_DIR="/run/user/$(id -u)"
export XDG_RUNTIME_DIR

# Wait for the user bus socket to actually appear
timeout 10s bash -c "until [ -S \"\$XDG_RUNTIME_DIR/bus\" ]; do sleep 0.5; done"

# Define the bus address so systemctl knows where to look
export DBUS_SESSION_BUS_ADDRESS="unix:path=$XDG_RUNTIME_DIR/bus"

systemctl --user --no-pager list-unit-files ubuntu-insights-collect.service
systemctl --user --no-pager list-unit-files ubuntu-insights-collect.timer
systemctl --user --no-pager list-unit-files ubuntu-insights-upload.service
systemctl --user --no-pager list-unit-files ubuntu-insights-upload.timer

# Go through a typical workflow, ensuring that no errors are encountered.
ubuntu-insights --version
ubuntu-insights --help
ubuntu-insights collect -df
ubuntu-insights collect
ubuntu-insights consent -s=true
ubuntu-insights collect
ubuntu-insights upload -df
