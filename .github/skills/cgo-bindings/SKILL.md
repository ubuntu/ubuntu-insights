---
name: cgo-bindings
description: "Handles C-bindings, building/testing of C/CGo assets, code generation, and C header verification. Use when working on C/CGo files under insights/C/."
---

# C/CGo and C-Bindings Operations

## When to Use

- Working on C header/source files or CGo wrappers in `insights/C/`.
- Generating C assets or updating dynamic library mappings.
- Validating native builds.

## Setup and Native Host Requirements

If building or testing Go modules with dynamic C dependencies _locally on your host_ (rather than inside a clean build environment or runner):

- Ensure your path is set up for C compiling dependencies.
- **Ubuntu/Debian host requirement** (not applicable/required on macOS/Windows except in cross-compilation environments):
  ```bash
  sudo apt update && sudo apt install -y libwayland-dev
  ```

## Procedures

### 1) Code Generation (C Bindings)

When any C-headers, CGo files, or bindings are modified:

- Run CGo code generation from the repository root:
  ```bash
  go generate ./insights/C/...
  ```
- Make sure C/C++ style conforms to repo guidelines before committing.

### 2) Lint and Format Checks

- Enforce formatting of C/C++ headers and sources using `clang-format`:
  - Run the CI gating script or manually format files in `insights/C/` using your preferred clang-format wrapper.

### 3) C/CGo Integration & Thread-Safety Verification

C/CGo boundaries are highly susceptible to memory access violations, thread linkage errors, and race conditions.

- To verify modifications, always execute the C test suites with Go's **Concurrency Race Detector** enabled (refer to the `go-development` skill for general test execution):
  ```bash
  (cd insights && go test -count=1 -race ./C/...)
  ```
  _(Note: A pre-existing `nakedret` warning in `insights/C/libinsights.go` is known; call it out in PR comments if unrelated to your changes)._
