# OTLP Log Viewer

Minimal Go webserver that ingests OTLP JSON log payloads and displays them in a browser UI.

## Quick Start

```bash
go run .
```

Server starts on `http://localhost:8080`.

## Usage

### Send logs

```bash
curl -X POST -H "Content-Type: application/json" -d @logs.json -i localhost:8080/v1/logs
```

### View logs

Open `http://localhost:8080/ui` in your browser. The page auto-refreshes every 2 seconds.

## Example `logs.json`

```json
{
  "resourceLogs": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": { "stringValue": "my.service" }
          }
        ]
      },
      "scopeLogs": [
        {
          "scope": {
            "name": "my.library",
            "version": "1.0.0"
          },
          "logRecords": [
            {
              "timeUnixNano": "1544712660300000000",
              "observedTimeUnixNano": "1544712660300000000",
              "severityNumber": 10,
              "severityText": "Information",
              "traceId": "5B8EFFF798038103D269B633813FC60C",
              "spanId": "EEE19B7EC3C1B174",
              "body": { "stringValue": "Example log record" },
              "attributes": [
                { "key": "string.attribute", "value": { "stringValue": "some string" } },
                { "key": "boolean.attribute", "value": { "boolValue": true } },
                { "key": "int.attribute", "value": { "intValue": "10" } }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

More examples: [opentelemetry-proto/examples](https://github.com/open-telemetry/opentelemetry-proto/tree/main/examples)

## Endpoints

| Method | Path        | Description                              |
| ------ | ----------- | ---------------------------------------- |
| POST   | `/v1/logs`  | Ingest OTLP JSON log payload             |
| GET    | `/api/logs` | Return stored logs as JSON               |
| GET    | `/ui`       | Web UI displaying logs (auto-refreshing) |

## Dependencies

Uses the same OTLP proto types as the OpenTelemetry Collector:

- `go.opentelemetry.io/proto/otlp` — protobuf-generated OTLP types
- `google.golang.org/protobuf` — protojson unmarshaling