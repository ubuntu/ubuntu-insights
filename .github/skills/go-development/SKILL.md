---
name: go-development
description: "Handles Go dependency bootstrapping, testing (including -race detector), and linting. Use when checking, validating, or running test suites for Go modules."
---

# Go Development and Testing

This skill provides on-demand workflows for bootstrapping dependencies, running standard or race-detecting tests, and running linters.

## When to Use

- Initializing/bootstrapping the workspace dependencies.
- Running package tests or looking for concurrent issues using the race detector.
- Running golangci-lint to verify code style and standards compliance locally.

## Procedures

### 1) Bootstrapping Dependencies

Download Go module dependencies per workspace module. It is recommended to do this per module rather than a single workspace download to avoid network/tool timeouts:

```bash
(cd common && go mod download)
(cd insights && go mod download)
(cd server && go mod download)
(cd tools && go mod download)
```

### 2) Running Tests (Standard & Race Detector)

Always run tests before pushing changes. This verifies both logical correctness and concurrent thread safety.

#### Standard Unit & Integration Tests (Faster)

Run regular non-race test checks:

```bash
(cd common && go test -count=1 ./...)
(cd insights && go test -count=1 ./...)
(cd server && go test -count=1 ./...)
```

#### Race Detection Tests (Highly Recommended)

Telemetry collection runs asynchronously. Identifying concurrency bugs and data races is a hard requirement. It is highly recommended to run tests with Go's race detector active:

```bash
(cd common && go test -count=1 -race ./...)
(cd insights && go test -count=1 -race ./...)
(cd server && go test -count=1 -race ./...)
```

_(Note: Docker or a Docker-compatible engine is required for running server-side integration tests as they rely on `testcontainers-go`)_.

### 3) Code Audits and Linting

Run `golangci-lint` utilizing the repository's shared configuration to match CI guidelines:

```bash
(cd common && go tool golangci-lint run --config ../.golangci.yaml)
(cd insights && go tool golangci-lint run --config ../.golangci.yaml)
(cd server && go tool golangci-lint run --config ../.golangci.yaml)
```

### 4) Building Modules (Optional / Debugging only)

_Warning: Running compilation builds is unnecessary for daily development, lints, or test runs. Avoid compiling binary outputs unless specifically needed for deployment, packaging, or debugging. Building binaries risks leaving unwanted, untracked binary files (_.deb, binary executables) in the workspace.\*

If you must compile a binary output for diagnostic testing:

- **Client CommandLine Interface**:
  ```bash
  go build -o ubuntu-insights ./insights/cmd/insights
  ```
- **Server Services**:
  ```bash
  go build ./server/cmd/web-service ./server/cmd/ingest-service
  ```
