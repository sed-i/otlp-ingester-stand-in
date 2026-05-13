# ADR 0004: Tabbed UI for Multi-Signal Viewing

**Status:** Accepted

**Date:** 2026-05-13

## Context

The UI originally showed only logs in a single table. After adding metrics and traces endpoints (ADR 0002), we needed a way to view all three signals without separate pages or URL paths.

## Decision

**Single-page tabbed interface** with one tab per signal (Logs, Metrics, Spans). The HTML/CSS/JS remains a single embedded `htmlUI` const in `ui.go`.

**JavaScript polling loop reads the active tab's endpoint.** Tab switches update a `signal` variable that drives both the API endpoint and the table schema. Only the active tab is polled — no background fetches for inactive tabs avoids wasted requests.

**Schema-driven rendering** — each signal defines a column array (`schemas.logs`, `schemas.metrics`, `schemas.traces`) with key, label, CSS class, and optional render function. The table head and body are rebuilt from the schema on every render, ensuring headers always match the active signal's data shape.

**Colored badges** for signal-specific fields:

| Signal | Field | Badge colors |
|--------|-------|-------------|
| Logs | Severity | red (ERROR/FATAL), amber (WARN), green (INFO), grey (DEBUG/TRACE) |
| Metrics | Type | blue (sum), pink (gauge), purple (histogram), green (exp.histogram) |
| Spans | Kind | blue (server), pink (client), amber (producer), purple (consumer) |

**Alternative approaches considered:**

| Option | Rejected because |
|--------|-----------------|
| Separate pages (`/ui/logs`, `/ui/metrics`, `/ui/spans`) | Requires URL routing, no side-by-side comparison feel |
| Three tables on one page | Cluttered, wastes vertical space |
| React/Vue framework | Adds build toolchain for a demo tool with ~120 lines of DOM manipulation |
| SSE/WebSocket push | Overkill — 2s polling is fine for a development viewer |

## Consequences

- The `htmlUI` const is ~120 lines — larger than the original, but still a single copy-pasteable block.
- Tab switching triggers an immediate fetch (no 2s wait), so UX feels responsive.
- On fetch error, the tbody is cleared to prevent stale data from the previous tab displaying under new headers.