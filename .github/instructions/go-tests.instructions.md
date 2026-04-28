---
description: "Use when creating or editing Go tests, including integration test wrappers and CGo test harness files. Covers table-driven map/dict test cases, subtest structure, golden file usage, and intentional golden updates."
applyTo: "**/*_test.go,**/*integrationtests.go,**/libinsightstest.go"
---

# Go Test Conventions

- Prefer table-driven tests keyed by name in a map/dict (`map[string]struct{...}`), iterated as `for name, tc := range tests { ... }`.
- Use subtests for each case: `t.Run(name, func(t *testing.T) { ... })`.
- Keep test cases deterministic and self-contained; avoid hidden shared mutable state between cases.
- Use golden files when expected output is large, structured, or hard to maintain inline.
- Use shared helpers from `common/testutils/golden.go`:
  - `LoadWithUpdateFromGolden` for plaintext expectations.
  - `LoadWithUpdateFromGoldenYAML` for structured/YAML expectations.
- Update golden files intentionally with `TESTS_UPDATE_GOLDEN=yes`, then commit the updated golden artifacts in the same PR.
- Prefer explicit, stable assertions over ad hoc string contains checks.

## Quick Pattern

```go
tests := map[string]struct {
    input string
    want  string
}{
    "simple case": {input: "x", want: "y"},
}

for name, tc := range tests {
    t.Run(name, func(t *testing.T) {
        got := run(tc.input)
        want := testutils.LoadWithUpdateFromGolden(t, got)
        require.Equal(t, want, got)
    })
}
```
