---
description: "Use when creating or editing Go production code. Covers repository Go style for naming, control flow, dependency setup, and keeping logic explicit and easy to maintain."
applyTo: "**/*.go"
---

# Go Style Conventions

- Prefer small, explicit functions over clever abstractions.
- Validate required inputs and dependencies early, then return immediately on invalid state.
- Strive for making invalid states non-representable to avoid spreading validation everywhere.
- Keep control flow flat: prefer guard clauses and early returns over nested `else` blocks.
- Keep exported APIs narrow. Do not widen visibility just to satisfy tests; use the existing `export_test.go` pattern instead.
- Prefer descriptive names over abbreviations, except for well-known initialisms (`URL`, `JSON`, `HTTP`) and conventional short receiver names.
- Keep constructors and options pragmatic:
  - Use a constructor when required dependencies or invariants must be enforced.
  - Do not introduce option plumbing for a type that can be configured more clearly with direct fields or simple arguments.
- Prefer package-level sentinel errors only for conditions callers need to branch on.
- Keep comments for non-obvious invariants, edge cases, or intent; avoid comments that restate the code.
- Avoid unnecessary helper extraction. A short local block is usually better than a helper that obscures the main path.
- At a given layer, either log an error or return it with context. Avoid duplicating the same failure message in both places unless each layer adds distinct operational value.

## Quick Pattern

```go
func NewProcessor(reportsDir string, uploader Uploader) (*Processor, error) {
    if reportsDir == "" {
        return nil, errors.New("reportsDir must be set")
    }
    if uploader == nil {
        return nil, errors.New("uploader must be set")
    }

    return &Processor{
        reportsDir: reportsDir,
        uploader:   uploader,
    }, nil
}
```
