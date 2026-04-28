# Copilot Cloud Agent Onboarding

## What This Repository Is

- `ubuntu-insights` is a Go monorepo for Ubuntu telemetry collection and processing.
- It has two product areas:
  - Client side (`insights`): CLI (`ubuntu-insights`), Go API, C bindings (`libinsights`), Debian packaging.
  - Server side (`server`): `web-service` (HTTP intake) and `ingest-service` (validation + DB ingest).
- Shared code lives in `common`.

## Stack And Runtime

- **Language**: Go. ALWAYS check the root [go.work](../go.work) and module-level `go.mod` files for the exact Go and toolchain versions. Do not assume hardcoded Go versions.
- **C/CGo bindings**: Found under [insights/C/](../insights/C).
- **Debian packaging**: Found under [insights/debian/](../insights/debian).
- **CI Environments**: Linux, macOS, and Windows for Go checks; Debian packaging and autopkgtest run on Ubuntu.

## Repository Layout & Core Paths

- [go.work](../go.work): workspace root for the 4 modules:
  - [common/](../common/): shared helpers (`cli`, `fileutils`, test utilities including golden file helpers).
  - [insights/](../insights/): client implementation, Collectors, Consent logic, Uploader, and C bindings.
  - [server/](../server/): intake web-service and ingest pipeline.
  - [tools/](../tools/): development/utility CLI and tools.

## Developer Workflows

The repository maintains specialized on-request Agent Skills for specific domains. Refer to these skills rather than hardcoding complex paths or execution steps:

- **Go Bootstrapping, Building, Testing, & Linting**: See the `go-development` skill for commands to fetch dependencies, build components, run fast unit tests, run race-detector checks (`-race`), and run golangci-lint.
- **C/CGo and C Bindings Development**: See the `cgo-bindings` skill for compiler requirements, manual generating (`go generate ./insights/C/...`), formatting, and C integration checks.
- **Debian Package Building & Autopkgtests**: See the `debian-packaging` skill for isolated packaging via `sbuild`, local package building with debuild, and verifying dbus/systemd smoke tests.

Prefer utilizing the workspace's pre-configured VS Code tasks (such as `Go: Test Module(s)` or `Go: golangci-lint Module(s)`) via the tasks runner whenever possible.

## Key Constraints & Validation Rules

- **Strict Error and Code Style**: Automated instructions in `.github/instructions/` (e.g., `go-style.instructions.md`, `go-error-handling.instructions.md`, `go-tests.instructions.md`, `go-export-pattern.instructions.md`) are automatically applied using file-matching rules. Follow them exactly when editing code.
- **Systemd User Units**: If editing units or timers under [insights/autostart/systemd/](../insights/autostart/systemd), the systemd rules in `.github/instructions/systemd.instructions.md` apply automatically. You must run:
  ```bash
  systemd-analyze --user verify ./insights/autostart/systemd/*
  ```
- **Golden Files**: If updating golden test outputs under unit/integration tests, trigger standard tests with `TESTS_UPDATE_GOLDEN=yes` (or use the VS Code task `Go: Update Golden Files`) and commit the resulting changes together.
- **Docker Requirement**: The server integration tests utilize `testcontainers-go`; ensure a Docker-compatible runtime is active before executing them.
