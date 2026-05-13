# OTLP Telemetry Viewer

Minimal Go webserver that ingests OTLP JSON payloads (logs, metrics, traces) and displays them in a browser UI.

## Quick Start

```bash
go run .
```

Server starts on `http://localhost:8080`.

## Usage

### Send telemetry

```bash
curl -X POST -H "Content-Type: application/json" -d @testdata/logs.json -i localhost:8080/v1/logs
curl -X POST -H "Content-Type: application/json" -d @testdata/metrics.json -i localhost:8080/v1/metrics
curl -X POST -H "Content-Type: application/json" -d @testdata/trace.json -i localhost:8080/v1/traces
```

or simply

```bash
just push
```

### View telemetry

Open `http://localhost:8080/ui` in your browser. The page auto-refreshes every 2 seconds.

### Query telemetry

All query endpoints accept filtering via query parameters and return JSON arrays.

**GET `/api/v1/logs`** — supported parameters:

| Parameter       | Description                                                                              |
| --------------- | ---------------------------------------------------------------------------------------- |
| `body`          | Exact body match                                                                         |
| `body_contains` | Body substring match                                                                     |
| `body_regex`    | Body regex match (e.g. `^Example.*`)                                                     |
| `severity`      | Exact severity match (e.g. `Information`, `ERROR`)                                       |
| `service`       | Exact service name match                                                                 |
| `attr[name]`    | Attribute exact match (e.g. `attr[int.attribute]=10`)                                    |
| `attr[name]`    | Attribute wildcard prefix when value ends with `*` (e.g. `attr[string.attribute]=some*`) |
| `from`          | Start of time range (RFC3339, e.g. `2018-12-01T00:00:00Z`)                               |
| `to`            | End of time range (RFC3339)                                                              |
| `limit`         | Max results (default unlimited, capped at 1000)                                          |
| `offset`        | Skip first N results                                                                     |

**GET `/api/v1/metrics`** — supported parameters:

| Parameter                       | Description                                                        |
| ------------------------------- | ------------------------------------------------------------------ |
| `name`                          | Exact metric name match                                            |
| `name_contains`                 | Metric name substring match                                        |
| `type`                          | Metric type (`sum`, `gauge`, `histogram`, `exponential_histogram`) |
| `service`                       | Exact service name match                                           |
| `attr[name]`                    | Attribute exact or wildcard match                                  |
| `from`, `to`, `limit`, `offset` | Same as logs                                                       |

**GET `/api/v1/traces`** — supported parameters:

| Parameter                       | Description                                                        |
| ------------------------------- | ------------------------------------------------------------------ |
| `name`                          | Exact span name match                                              |
| `kind`                          | Span kind (`server`, `client`, `producer`, `consumer`, `internal`) |
| `service`                       | Exact service name match                                           |
| `trace_id`                      | Trace ID (case-insensitive hex)                                    |
| `span_id`                       | Span ID (case-insensitive hex)                                     |
| `attr[name]`                    | Attribute exact or wildcard match                                  |
| `from`, `to`, `limit`, `offset` | Same as logs                                                       |

#### Examples

```bash
curl 'localhost:8080/api/v1/logs?body=Example+log+record'
curl 'localhost:8080/api/v1/logs?body_contains=Example'
curl 'localhost:8080/api/v1/logs?body_regex=^Example.*'
curl 'localhost:8080/api/v1/logs?severity=Information&service=my.service'
curl 'localhost:8080/api/v1/logs?attr[int.attribute]=10'
curl 'localhost:8080/api/v1/logs?attr[string.attribute]=some*'
curl 'localhost:8080/api/v1/logs?from=2018-12-01T00:00:00Z&to=2018-12-31T23:59:59Z'
curl 'localhost:8080/api/v1/metrics?name=request.count&type=sum'
curl 'localhost:8080/api/v1/traces?service=my.service&kind=server'
```

## Examples

All [testdata](testdata/) was taken from [opentelemetry-proto/examples](https://github.com/open-telemetry/opentelemetry-proto/tree/main/examples)

## Endpoints

| Method | Path              | Description                                   |
| ------ | ----------------- | --------------------------------------------- |
| POST   | `/v1/logs`        | Ingest OTLP JSON log payload                  |
| POST   | `/v1/metrics`     | Ingest OTLP JSON metrics payload              |
| POST   | `/v1/traces`      | Ingest OTLP JSON trace payload                |
| GET    | `/api/v1/logs`    | Query stored logs with filters                |
| GET    | `/api/v1/metrics` | Query stored metrics with filters             |
| GET    | `/api/v1/traces`  | Query stored spans with filters               |
| GET    | `/api/logs`       | Legacy alias for `/api/v1/logs`               |
| GET    | `/ui`             | Web UI displaying telemetry (auto-refreshing) |

## Dependencies

Uses the same OTLP proto types as the OpenTelemetry Collector:

- `go.opentelemetry.io/proto/otlp` — protobuf-generated OTLP types
- `google.golang.org/protobuf` — protojson unmarshaling
