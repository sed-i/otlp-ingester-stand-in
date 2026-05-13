package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

const maxFilterLimit = 1000

func parseTimeRange(q url.Values) (*time.Time, *time.Time, int, int, error) {
	var from, to *time.Time
	if s := q.Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("invalid 'from' timestamp: %w", err)
		}
		from = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("invalid 'to' timestamp: %w", err)
		}
		to = &t
	}
	limit := 0
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, nil, 0, 0, fmt.Errorf("invalid 'limit': %s", s)
		}
		if n > maxFilterLimit {
			n = maxFilterLimit
		}
		limit = n
	}
	offset := 0
	if s := q.Get("offset"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, nil, 0, 0, fmt.Errorf("invalid 'offset': %s", s)
		}
		offset = n
	}
	return from, to, limit, offset, nil
}

func parseAttrFilters(q url.Values) (exact, wildcard map[string]string) {
	exact = make(map[string]string)
	wildcard = make(map[string]string)
	for k, vs := range q {
		if !strings.HasPrefix(k, "attr[") || !strings.HasSuffix(k, "]") {
			continue
		}
		attrKey := strings.ToLower(k[5 : len(k)-1])
		attrVal := vs[0]
		if strings.HasSuffix(attrVal, "*") {
			wildcard[attrKey] = attrVal[:len(attrVal)-1]
		} else {
			exact[attrKey] = attrVal
		}
	}
	return
}

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
				logBody := stringValue(lr.GetBody())
				severity := lr.GetSeverityText()
				if severity == "" {
					severity = fmt.Sprintf("SEVERITY_%d", lr.GetSeverityNumber())
				}

				rec := FlatLogRecord{
					Timestamp:  parseNanos(lr.GetTimeUnixNano()).UTC().Format(time.RFC3339Nano),
					Service:    service,
					Scope:      scope,
					Severity:   severity,
					Body:       logBody,
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

	q := r.URL.Query()
	if len(q) == 0 {
		json.NewEncoder(w).Encode(logStore.GetAll())
		return
	}

	from, to, limit, offset, err := parseTimeRange(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	exact, wildcard := parseAttrFilters(q)

	f := LogFilter{
		Body:         q.Get("body"),
		BodyContains: q.Get("body_contains"),
		Severity:     q.Get("severity"),
		Service:      q.Get("service"),
		Attrs:        exact,
		AttrWildcard: wildcard,
		From:         from,
		To:           to,
		Limit:        limit,
		Offset:       offset,
	}

	if s := q.Get("body_regex"); s != "" {
		re, err := regexp.Compile(s)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid 'body_regex': %v", err), http.StatusBadRequest)
			return
		}
		f.BodyRegex = re
	}

	json.NewEncoder(w).Encode(logStore.Filter(f))
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

	q := r.URL.Query()
	if len(q) == 0 {
		json.NewEncoder(w).Encode(metricStore.GetAll())
		return
	}

	from, to, limit, offset, err := parseTimeRange(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	exact, wildcard := parseAttrFilters(q)

	f := MetricFilter{
		Name:         q.Get("name"),
		NameContains: q.Get("name_contains"),
		Type:         q.Get("type"),
		Service:      q.Get("service"),
		Attrs:        exact,
		AttrWildcard: wildcard,
		From:         from,
		To:           to,
		Limit:        limit,
		Offset:       offset,
	}

	json.NewEncoder(w).Encode(metricStore.Filter(f))
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

	q := r.URL.Query()
	if len(q) == 0 {
		json.NewEncoder(w).Encode(spanStore.GetAll())
		return
	}

	from, to, limit, offset, err := parseTimeRange(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	exact, wildcard := parseAttrFilters(q)

	f := SpanFilter{
		Name:         q.Get("name"),
		Kind:         q.Get("kind"),
		Service:      q.Get("service"),
		TraceID:      q.Get("trace_id"),
		SpanID:       q.Get("span_id"),
		Attrs:        exact,
		AttrWildcard: wildcard,
		From:         from,
		To:           to,
		Limit:        limit,
		Offset:       offset,
	}

	json.NewEncoder(w).Encode(spanStore.Filter(f))
}

func handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlUI))
}