---
name: debian-packaging
description: "Handles Debian package building, dependency management, sbuild, and autopkgtest runs. Use when working under insights/debian/ or compiling .deb archives."
---

# Debian Packaging and Autopkgtests

This skill covers the tools and procedures required to build and verify Debian packages in clean, isolated environments.

## When to Use

- Editing files under `insights/debian/` (such as `changelog`, `control`, `rules`).
- Building the `.deb` client binary or source packages.
- Running Debian package builds and autopkgtests.

## Procedures

### 1) Isolated Builds (SBuild) — SAFEST & RECOMMENDED

To keep the developer's host environment clean and avoid dependency conflicts, package compilation should be run in isolated chroots.

- Use the pre-configured VS Code tasks:
  - **SBuild** task (`sbuild --source insights`): Builds the full debian binary package inside a secure sbuild chroot, automatically fetching the required build-deps.
  - **SBuild Source Only** task.
- To use sbuild from the CLI (if sbuild is configured on your system):
  ```bash
  sbuild --source insights
  ```

### 2) Running Autopkgtests & Smoke Tests

Autopkgtests are defined in `insights/debian/tests/control` and verify package functionality:

- **Go Tests (`run-tests.sh`)**: Runs standard module tests inside the package build environment.
- **Smoke Tests (`run-smoke-tests.sh`)**: Relies on user systemd and dbus-user-session behavior.
- _Troubleshooting Smoke Tests_: If smoke tests fail in container or VM test runners, ensure that systemd-logind is active and a user-systemd/dbus bus can be reached (via `loginctl` lingering user sessions and active user bus sessions).
