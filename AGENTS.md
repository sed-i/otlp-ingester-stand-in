# Agent Instructions

## Workflow

### 1. Write an ADR before implementing

Before writing any code for a feature or significant change, create an Architecture Decision Record (ADR) in `docs/adr/`.

- Follow the existing naming convention: `NNNN-short-title-with-dashes.md`
- Use the next available number
- Follow the format established by existing ADRs:
  - Title: `# ADR NNNN: Title`
  - `**Status:** Proposed`
  - `**Date:** YYYY-MM-DD`
  - Sections: `## Context`, `## Decision`, `## Consequences`
  - Include rejected alternatives with brief explanations in a table
- Keep it concise — the ADR should capture the _why_, not the _how_

### 2. Implement the change

Once the ADR is written, implement the change following the plan described in the ADR.

### 3. Run tests after implementation

After making changes, run:

```bash
just test
just integrate-test
```

This runs unit and integration tests. Ensure all tests pass. If tests fail, fix the issues and re-run until passing.

## Project conventions

Conventions are established by the ADRs in [`docs/adr/`](docs/adr/). Read the relevant ADR before working in an area; open a new one to change a convention.

- [ADR 0001](docs/adr/0001-otlp-log-viewer-design.md) — HTTP routing (`net/http`), OTLP parsing, `sync.RWMutex` store, embedded UI
- [ADR 0002](docs/adr/0002-multi-signal-support.md) — multi-signal (logs/metrics/traces) store and endpoint patterns
- [ADR 0003](docs/adr/0003-project-file-structure.md) — single-package file layout (`package main`)
- [ADR 0004](docs/adr/0004-tabbed-ui.md) — tabbed UI structure
- [ADR 0005](docs/adr/0005-bdd-testing.md) — BDD-style tests, test file placement
- [ADR 0006](docs/adr/0006-api-endpoint-design.md) — API endpoint naming and design
