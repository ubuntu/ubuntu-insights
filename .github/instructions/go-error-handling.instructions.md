---
description: "Use when creating or editing Go error handling. Covers repository guidance for sentinel errors, `%w` vs `%v`, `errors.Is`/`errors.As`, and when to use `errors.Join`."
applyTo: "**/*.go"
---

# Go Error Handling Conventions

- Prefer `errors.New` for static sentinel errors and declare them as `var ErrSomething = errors.New("...")`.
- Use `%w` only when callers are expected to match the underlying error later with `errors.Is` or `errors.As`.
- If the underlying error is only being included for human-readable context, use `%v` instead of `%w`.
- Do not mechanically wrap every returned error. Wrapping is part of the API surface because it preserves the cause chain.
- Prefer one meaningful layer of context at the abstraction boundary that changes what the operation means to the caller.
- When a caller needs to match a domain-specific condition and still retain extra detail, prefer `errors.Join` with a sentinel error.
- Use lowercase error messages without trailing punctuation.
- Avoid exporting foreign implementation details by default. Expose them only when callers have a concrete need to branch on them.

## Preferred Patterns

```go
var ErrConsentFileNotFound = errors.New("consent file not found")

func loadConsent(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return errors.Join(ErrConsentFileNotFound, err)
        }
        return fmt.Errorf("failed to read consent file: %v", err)
    }

    _ = data
    return nil
}
```

```go
func decodeConfig(path string) error {
    if err := readConfig(path); err != nil {
        return fmt.Errorf("invalid configuration file: %w", err)
    }
    return nil
}
```

Use `%w` in the second example only because callers legitimately need to inspect the underlying parse error later and act upon it differently from other errors.
