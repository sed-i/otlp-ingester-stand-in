# ADR 0006: API Endpoint Design

**Status:** Accepted

**Date:** 2026-05-13

## Context

The server needs HTTP endpoints that are easy to use from `curl` and the browser UI, while staying close enough to the OpenTelemetry Collector's OTLP HTTP paths to feel familiar. The endpoints must serve two purposes: ingesting telemetry and querying stored data.

## Decision

**Two path namespaces with consistent structure:**

| Purpose | Pattern | Examples |
|---------|---------|----------|
| Ingest (POST) | `/v1/{signal}` | `/v1/logs`, `/v1/metrics`, `/v1/traces` |
| Query (GET) | `/api/v1/{signal}` | `/api/v1/logs`, `/api/v1/metrics`, `/api/v1/traces` |

**Ingest requests require `POST` method** — `GET` returns 405 Method Not Allowed. This catches accidental tool that sends logs as a URL query.

**JSON-only** — no protobuf binary support. The demo target audience uses `curl` with `-d @file.json`. Accepting protobuf would require content negotiation (`Content-Type: application/x-protobuf`) with no clear benefit.

**Ingest response: `{"received": <count>}`** — a simple JSON object with the number of data points ingested. Enough for a quick `curl` sanity check, trivially parseable if scripted.

**Query response: `[{...}, {...}]`** — a flat JSON array of records. The UI and CLI consumers both iterate over arrays naturally. No pagination — the in-memory store is expected to hold at most hundreds of records in a demo session.

**Error responses use plain text** via `http.Error` — simple error messages like `"failed to parse OTLP JSON: ..."`. No JSON error envelope. Acceptable for a demo tool.

**Legacy alias:** `GET /api/logs` maps to the same handler as `GET /api/v1/logs`. This existed before the `/api/v1/` namespace was introduced and is preserved to avoid breaking the UI (which originally polled `/api/logs`).

**Path rationale:** The OpenTelemetry Collector serves OTLP at `POST /v1/logs`, `/v1/metrics`, `/v1/traces`. Our ingest paths match this convention. The query API (`/api/v1/{signal}`) is an orthogonal concern that does not exist in the collector.

| Option | Rejected because |
|--------|-----------------|
| Exact collector paths (`POST /v1/logs`, no query API) | No way to retrieve ingested data; need a separate query surface |
| `/ingest/{signal}` and `/query/{signal}` | Verbose; `/v1/` prefix is established convention |
| Query on ingest paths (`GET /v1/logs`) | Ambiguous — POST creates, GET reads, but same path for both is uncommon |
| Paginated query responses | Overengineering for an in-memory demo store |
| JSON error envelope (`{"error": "..."}`) | Adds parsing complexity on the client for no UX gain |

## Consequences

- 7 route registrations in `main.go`: 3 ingest, 3 query, 1 legacy alias.
- The API surface is small enough to document in a README table.
- Adding a fourth signal (e.g. profiles) follows the same pattern — add a handler and register two routes.
- The UI's JS polling loop constructs the endpoint dynamically: `"/api/v1/" + signal`.