package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	req := &collectorlogs.ExportLogsServiceRequest{}
	if err := protojson.Unmarshal(body, req); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse OTLP JSON: %v", err), http.StatusBadRequest)
		return
	}

	count := 0
	for _, rl := range req.GetResourceLogs() {
		service := ""
		if rl.GetResource() != nil {
			service = extractServiceName(rl.GetResource().GetAttributes())
		}
		if service == "" {
			service = "-"
		}

		for _, sl := range rl.GetScopeLogs() {
			scope := "-"
			if sl.GetScope() != nil {
				scope = sl.GetScope().GetName()
			}

			for _, lr := range sl.GetLogRecords() {
				body := stringValue(lr.GetBody())
				severity := lr.GetSeverityText()
				if severity == "" {
					severity = fmt.Sprintf("SEVERITY_%d", lr.GetSeverityNumber())
				}

				rec := FlatLogRecord{
					Timestamp:  parseNanos(lr.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
					Service:    service,
					Scope:      scope,
					Severity:   severity,
					Body:       body,
					TraceID:    bytesToHex(lr.GetTraceId()),
					SpanID:     bytesToHex(lr.GetSpanId()),
					Attributes: attrsToMap(lr.GetAttributes()),
				}
				logStore.Append(rec)
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"received":%d}`, count)
}

func handleAPILogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logStore.GetAll())
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	req := &collectormetrics.ExportMetricsServiceRequest{}
	if err := protojson.Unmarshal(body, req); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse OTLP JSON: %v", err), http.StatusBadRequest)
		return
	}

	count := 0
	for _, rm := range req.GetResourceMetrics() {
		service := ""
		if rm.GetResource() != nil {
			service = extractServiceName(rm.GetResource().GetAttributes())
		}
		if service == "" {
			service = "-"
		}

		for _, sm := range rm.GetScopeMetrics() {
			scope := "-"
			if sm.GetScope() != nil {
				scope = sm.GetScope().GetName()
			}

			for _, m := range sm.GetMetrics() {
				for _, dp := range m.GetSum().GetDataPoints() {
					rec := FlatMetricRecord{
						Timestamp:   parseNanos(dp.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
						Service:     service,
						Scope:       scope,
						Name:        m.GetName(),
						Unit:        m.GetUnit(),
						Description: m.GetDescription(),
						Type:        "sum",
						Value:       metricValueString(dp),
						Attributes:  attrsToMap(dp.GetAttributes()),
					}
					metricStore.Append(rec)
					count++
				}
				for _, dp := range m.GetGauge().GetDataPoints() {
					rec := FlatMetricRecord{
						Timestamp:   parseNanos(dp.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
						Service:     service,
						Scope:       scope,
						Name:        m.GetName(),
						Unit:        m.GetUnit(),
						Description: m.GetDescription(),
						Type:        "gauge",
						Value:       metricValueString(dp),
						Attributes:  attrsToMap(dp.GetAttributes()),
					}
					metricStore.Append(rec)
					count++
				}
				for _, dp := range m.GetHistogram().GetDataPoints() {
					rec := FlatMetricRecord{
						Timestamp:   parseNanos(dp.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
						Service:     service,
						Scope:       scope,
						Name:        m.GetName(),
						Unit:        m.GetUnit(),
						Description: m.GetDescription(),
						Type:        "histogram",
						Value:       histogramSummary(dp),
						Attributes:  attrsToMap(dp.GetAttributes()),
					}
					metricStore.Append(rec)
					count++
				}
				for _, dp := range m.GetExponentialHistogram().GetDataPoints() {
					rec := FlatMetricRecord{
						Timestamp:   parseNanos(dp.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
						Service:     service,
						Scope:       scope,
						Name:        m.GetName(),
						Unit:        m.GetUnit(),
						Description: m.GetDescription(),
						Type:        "exponential_histogram",
						Value:       expHistogramSummary(dp),
						Attributes:  attrsToMap(dp.GetAttributes()),
					}
					metricStore.Append(rec)
					count++
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"received":%d}`, count)
}

func handleAPIMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metricStore.GetAll())
}

func handleTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	req := &collectortrace.ExportTraceServiceRequest{}
	if err := protojson.Unmarshal(body, req); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse OTLP JSON: %v", err), http.StatusBadRequest)
		return
	}

	count := 0
	for _, rs := range req.GetResourceSpans() {
		service := ""
		if rs.GetResource() != nil {
			service = extractServiceName(rs.GetResource().GetAttributes())
		}
		if service == "" {
			service = "-"
		}

		for _, ss := range rs.GetScopeSpans() {
			scope := "-"
			if ss.GetScope() != nil {
				scope = ss.GetScope().GetName()
			}

			for _, sp := range ss.GetSpans() {
				startNanos := sp.GetStartTimeUnixNano()
				endNanos := sp.GetEndTimeUnixNano()
				duration := time.Duration(endNanos-startNanos) * time.Nanosecond

				rec := FlatSpanRecord{
					Timestamp:    parseNanos(startNanos).UTC().Format(time.RFC3339Nano),
					Service:      service,
					Scope:        scope,
					Name:         sp.GetName(),
					Kind:         spanKindString(sp.GetKind()),
					TraceID:      bytesToHex(sp.GetTraceId()),
					SpanID:       bytesToHex(sp.GetSpanId()),
					ParentSpanID: bytesToHex(sp.GetParentSpanId()),
					Duration:     duration.String(),
					Attributes:   attrsToMap(sp.GetAttributes()),
				}
				spanStore.Append(rec)
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"received":%d}`, count)
}

func handleAPITraces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spanStore.GetAll())
}

func handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlUI))
}
