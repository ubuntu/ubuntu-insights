---
description: "Use when creating or editing Go export_test wrappers. Covers the repository export pattern for exposing internal symbols to tests without changing production APIs."
applyTo: "**/export_test.go"
---

# Go Export Pattern For Tests

Use `export_test.go` to expose internals only for tests in the same package.

- Keep these files test-only bridges: no production logic, no runtime behavior changes.
- Prefer simple exported wrappers around private members:
  - Type aliases for private types where useful (`type AppConfig = appConfig`).
  - Getter methods to expose private fields needed by tests.
  - Setter/helper methods to configure app/command state for tests.
  - Option constructors that inject test doubles when needed.
- Keep helper setup deterministic and test-friendly:
  - Use `t.Helper()` in setup helpers.
  - Use `t.TempDir()` for temporary files.
  - Use `require.NoError` for setup failures with clear `Setup:`-prefixed messages.
- Preserve encapsulation where possible:
  - If returning internal maps/slices/pointers, guard with existing locks and avoid unnecessary mutation in tests.
- Keep naming explicit (`NewForTests`, `GenerateTestConfig`, `SetArgs`, `AllowSet`) and place wrappers near the internals they expose.

## Scope Notes

- This pattern is for test support only and must not be referenced by non-test production code.
- Prefer this pattern over widening production visibility solely for tests.

## Companion Test Code Outside export_test.go

In regular test files (`*_test.go`) that pair with this pattern:

- Consume helpers exposed by `export_test.go` instead of changing production visibility.
- Build tests around explicit setup helpers (`NewForTests`, `GenerateTestConfig`, `GenerateTestAllowlist`) when available.
- Keep setup and assertion logic in `*_test.go`; keep `export_test.go` focused on bridging internals.
- Prefer package-internal tests (`package samepkg`) when you need access via these exported wrappers.
- Avoid duplicating wrapper logic in `*_test.go`; add missing wrappers in `export_test.go` instead.

## Do And Don’t

```go
// DO: keep export_test.go as a narrow bridge for tests.
func (a *App) SetArgs(args ...string) {
  a.cmd.SetArgs(args)
}

// DON'T: move test assertions or business logic into export_test.go.
func (a *App) VerifyBehaviorInTests(t *testing.T) {
  // avoid assertion logic here
}
```

```go
// DO in *_test.go: use exported test helpers from export_test.go.
func TestRun(t *testing.T) {
  app := NewForTests(t, nil, allowlistPath)
  app.SetArgs("--json-logs")
  // assertions belong here
}
```
