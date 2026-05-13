run:
	@go run .

test:
	@go test -v -count=1 ./...

push-logs:
    curl -X POST -H "Content-Type: application/json" -d @testdata/logs.json -i localhost:8080/v1/logs
get-logs:
    curl -s localhost:8080/api/v1/logs | jq

push-metrics:
    curl -X POST -H "Content-Type: application/json" -d @testdata/metrics.json -i localhost:8080/v1/metrics
get-metrics:
    curl -s localhost:8080/api/v1/metrics | jq

push-traces:
    curl -X POST -H "Content-Type: application/json" -d @testdata/trace.json -i localhost:8080/v1/traces
get-traces:
    curl -s localhost:8080/api/v1/traces | jq

push-events:
    curl -X POST -H "Content-Type: application/json" -d @testdata/events.json -i localhost:8080/v1/logs

push: push-logs push-metrics push-traces
get: get-logs get-metrics get-traces
match:
    curl 'localhost:8080/api/v1/logs?body=Example+log+record'
    curl -s --data-urlencode 'attr[int.attribute]=10' localhost:8080/api/v1/logs

integrate-test:
	@go test -v -count=1 ./... -ginkgo.v
