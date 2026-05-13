# ADR 0001: Minimal OTLP Telemetry Viewer

**Status:** Superseded by ADR 0003

**Date:** 2026-05-13

## Context

We need a minimal Golang webserver that ingests OTLP JSON log payloads and displays them in a browser. The server must accept `POST /v1/logs` with the same JSON format the OpenTelemetry Collector expects, and serve a web UI at `/ui` showing ingested logs.

## Decision

**Single-file, single-binary Go server** with no runtime framework — just `net/http` from the standard library for HTTP routing and serving.

**OTLP parsing via `go.opentelemetry.io/proto/otlp`** — the same protobuf-generated Go types used by the OpenTelemetry Collector. JSON payloads are unmarshaled with `protojson.Unmarshal` into `ExportLogsServiceRequest`. This avoids hand-writing and maintaining a fragile JSON parser for the OTLP schema.

**In-memory log store** protected by a `sync.RWMutex`. No persistence — logs are lost on restart. Acceptable for a demo/development tool.

**Embedded HTML UI** as a Go `const` string. No external frontend assets, build steps, or JS toolchain. The page polls `GET /api/logs` every 2 seconds with `fetch()`, renders a table, and applies severity-based color badges. Dark theme for console/observability aesthetic.

**Alternative approaches considered:**

| Option | Rejected because |
|--------|-----------------|
| Hand-parse OTLP JSON with `encoding/json` | Brittle, diverges from collector behavior, more code |
| Use a web framework (gin, chi, echo) | Adds dependency with no benefit for 3 routes |
| SSE / WebSocket for real-time UI updates | More complexity than 2s polling for a demo |
| Persist logs to disk | Out of scope for a minimal viewer |

## Consequences

- Adding a binary dependency on `go.opentelemetry.io/proto/otlp` and `google.golang.org/protobuf` (~3 MB indirect deps). This is acceptable since the collector itself depends on these.
- No log retention across restarts. Must re-POST logs after restart.
- UI has up to 2-second poll latency. Acceptable for a development viewer.
- The single-file structure is easy to copy, review, and deploy. No build pipeline needed — `go run .` is sufficient.