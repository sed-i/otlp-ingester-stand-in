# ADR 0005: BDD-Style Unit Testing

**Status:** Accepted

**Date:** 2026-05-13

## Context

The server needs unit tests to verify OTLP JSON parsing, storage, and API response correctness. Tests should read clearly for someone unfamiliar with the codebase.

## Decision

**BDD-style `Given/When/Then` subtests** using only Go's standard `testing` package — no external BDD framework.

Tests are structured as nested `t.Run` calls following the pattern:

```
TestLogIngestion
  Given a valid OTLP JSON payload
    When POST to /v1/logs
      Then it returns 200 and received count
      Then the log record is stored correctly
      Then GET /api/v1/logs returns the stored records
    Given an empty store
      When GET /api/v1/logs
        Then it returns an empty array
  Given an invalid request
    When POSTing with wrong method
      Then it returns 405
```

**Data-driven expected values** — tests parse `testdata/*.json` directly (using `encoding/json`, not protojson) to extract expected fields (severity text, attribute keys) rather than hardcoding them. This means the test data can be updated from the OTLP proto examples repo and tests will still pass as long as parsing remains correct.

**Test data in `testdata/`** — follows Go convention. Go tooling ignores `testdata/` for builds, and `go test` resolves paths relative to the package directory.

**HTTP handler tests via `httptest`** — `httptest.NewRequest` + `httptest.NewRecorder` exercise handlers directly without starting a real server. Fast, no port conflicts, no network.

**Alternative approaches considered:**

| Option | Rejected because |
|--------|-----------------|
| Ginkgo/Gomega | Adds dependency and learning curve for a small test suite |
| Integration tests (real HTTP server) | Slower, port conflicts, harder to debug |
| Table-driven tests without Given/When/Then | Less readable for non-Go developers |
| Hardcoded expected values | Brittle when test data changes |

## Consequences

- 27 test cases across 3 test functions (9 logs, 7 metrics, 7 traces + 4 error/empty cases).
- Test readability benefits newcomers — the output reads as natural language assertions.
- `just test` runs the full suite with `go test -v -count=1 ./...`.