# ADR 0003: Project File Structure

**Status:** Accepted

**Date:** 2026-05-13

**Supersedes:** ADR 0001

## Context

The initial implementation was a single `main.go` file (~650 lines). As the server grew to support three signals, locating specific code became harder. The file mixed type definitions, parsing helpers, HTTP handlers, the UI template, and wiring — six distinct concerns in one place.

## Decision

**Split into 5 files, all `package main`:**

```
main.go      — main(), route registration
handler.go   — HTTP handlers (handleLogs, handleMetrics, handleTraces, API GET variants, handleUI)
parser.go    — helper functions (proto value extraction, hex encoding, summaries)
store.go     — type definitions (FlatLogRecord, *Store) + store vars
ui.go        — htmlUI const
```

**No subdirectories, no `internal/` packages.** All files share `package main` — no import path changes, no new `go.mod` entries. Tests stay in `main_test.go` (one file is sufficient at ~480 lines).

This follows the standard-library convention seen in Go tools like `gofmt` and `cover`, which split single-binary tools by concern without creating packages.

**Alternative approaches considered:**

| Option | Rejected because |
|--------|-----------------|
| `internal/` sub-packages (store, handler, parser) | Overkill for ~660 lines. Internal boundary has no value since nothing imports this module. |
| Keep single file | Harder to navigate as the codebase grows. |
| One file per signal (logs.go, metrics.go, traces.go) | Splits related concerns — all parsing logic together is more cohesive than signal-per-file. |

## Consequences

- New contributors can locate code by concern: types in `store.go`, parsing in `parser.go`, HTTP in `handler.go`.
- No change to the build command — still `go run .` or `go build .`.
- The single `main_test.go` exercises all files transparently since everything is `package main`.
- If any file grows beyond ~250 lines, further splitting can follow the same pattern.