# ADR 0002: Multi-Signal Support (Logs, Metrics, Traces)

**Status:** Accepted

**Date:** 2026-05-13

## Context

The initial server only handled logs (`POST /v1/logs`). The tool needs to review all three OTLP signals — logs, metrics, and traces — from a single interface.

## Decision

**Extend to all three OTLP signals**, each with its own ingest and query endpoints:

| Signal | Ingest (POST) | Query (GET) |
|--------|--------------|-------------|
| Logs | `/v1/logs` | `/api/v1/logs` |
| Metrics | `/v1/metrics` | `/api/v1/metrics` |
| Traces | `/v1/traces` | `/api/v1/traces` |

**Separate flat record types per signal** rather than a union type:

- `FlatLogRecord` — timestamp, severity, body, trace/span IDs, attributes
- `FlatMetricRecord` — name, type (sum/gauge/histogram/exponential_histogram), value summary, attributes
- `FlatSpanRecord` — name, kind, duration, trace/span/parent IDs, attributes

**Separate in-memory stores** (`LogStore`, `MetricStore`, `SpanStore`) with identical `sync.RWMutex`-protected `Append`/`GetAll` patterns. Combined stores would force an untyped union; separate stores keep each signal's data shape clear.

**Parity with the OpenTelemetry Collector** — each signal uses its corresponding proto type (`ExportLogsServiceRequest`, `ExportMetricsServiceRequest`, `ExportTraceServiceRequest`) parsed via `protojson.Unmarshal`, same as the collector's OTLP HTTP receiver.

**Alternative approaches considered:**

| Option | Rejected because |
|--------|-----------------|
| Single `FlatRecord` with optional fields | Unclear which fields apply per signal, confusing consumers |
| One store with `any`/interface slices | Loses type safety, requires type assertions at query time |
| Separate servers per signal | Unnecessary operational overhead for a demo |

## Consequences

- Three separate handler functions with similar but signal-specific parsing logic. Some boilerplate (`protojson.Unmarshal`, error wrapping, response formatting) is repeated but intentionally so — extracting shared parsing would require generics or interfaces with minimal payoff at this scale.
- Each signal gets its own tab in the UI (see ADR 0004).
- Test data files live in `testdata/`: `logs.json`, `metrics.json`, `trace.json` — all from the OTLP proto examples repo.