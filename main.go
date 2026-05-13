package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	collectorlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

type FlatLogRecord struct {
	Timestamp  string            `json:"timestamp"`
	Service    string            `json:"service"`
	Scope      string            `json:"scope"`
	Severity   string            `json:"severity"`
	Body       string            `json:"body"`
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Attributes map[string]string `json:"attributes"`
}

type FlatMetricRecord struct {
	Timestamp   string            `json:"timestamp"`
	Service     string            `json:"service"`
	Scope       string            `json:"scope"`
	Name        string            `json:"name"`
	Unit        string            `json:"unit"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Value       string            `json:"value"`
	Attributes  map[string]string `json:"attributes"`
}

type FlatSpanRecord struct {
	Timestamp    string            `json:"timestamp"`
	Service      string            `json:"service"`
	Scope        string            `json:"scope"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Duration     string            `json:"duration"`
	Attributes   map[string]string `json:"attributes"`
}

type LogStore struct {
	mu      sync.RWMutex
	records []FlatLogRecord
}

func (s *LogStore) Append(r FlatLogRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
}

func (s *LogStore) GetAll() []FlatLogRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FlatLogRecord, len(s.records))
	copy(out, s.records)
	return out
}

type MetricStore struct {
	mu      sync.RWMutex
	records []FlatMetricRecord
}

func (s *MetricStore) Append(r FlatMetricRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
}

func (s *MetricStore) GetAll() []FlatMetricRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FlatMetricRecord, len(s.records))
	copy(out, s.records)
	return out
}

type SpanStore struct {
	mu      sync.RWMutex
	records []FlatSpanRecord
}

func (s *SpanStore) Append(r FlatSpanRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
}

func (s *SpanStore) GetAll() []FlatSpanRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FlatSpanRecord, len(s.records))
	copy(out, s.records)
	return out
}

var logStore = &LogStore{}
var metricStore = &MetricStore{}
var spanStore = &SpanStore{}

func parseNanos(ns uint64) time.Time {
	if ns == 0 {
		return time.Now()
	}
	return time.Unix(0, int64(ns))
}

func attrValue(v *commonv1.AnyValue) string {
	if v == nil {
		return ""
	}
	switch {
	case v.GetStringValue() != "":
		return v.GetStringValue()
	case v.GetBoolValue():
		return "true"
	case v.GetIntValue() != 0:
		return strconv.FormatInt(v.GetIntValue(), 10)
	case v.GetDoubleValue() != 0:
		return strconv.FormatFloat(v.GetDoubleValue(), 'f', -1, 64)
	case v.GetArrayValue() != nil:
		var parts []string
		for _, av := range v.GetArrayValue().GetValues() {
			parts = append(parts, attrValue(av))
		}
		return strings.Join(parts, ", ")
	case v.GetKvlistValue() != nil:
		var parts []string
		for _, kv := range v.GetKvlistValue().GetValues() {
			parts = append(parts, kv.GetKey()+"="+attrValue(kv.GetValue()))
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func extractServiceName(attrs []*commonv1.KeyValue) string {
	for _, a := range attrs {
		if a.GetKey() == "service.name" {
			return attrValue(a.GetValue())
		}
	}
	return "-"
}

func attrsToMap(attrs []*commonv1.KeyValue) map[string]string {
	m := make(map[string]string)
	for _, a := range attrs {
		m[a.GetKey()] = attrValue(a.GetValue())
	}
	return m
}

func stringValue(v *commonv1.AnyValue) string {
	if v == nil {
		return ""
	}
	switch {
	case v.GetStringValue() != "":
		return v.GetStringValue()
	case v.GetIntValue() != 0:
		return strconv.FormatInt(v.GetIntValue(), 10)
	case v.GetDoubleValue() != 0:
		return strconv.FormatFloat(v.GetDoubleValue(), 'f', -1, 64)
	case v.GetBoolValue():
		return "true"
	default:
		return attrValue(v)
	}
}

func bytesToHex(b []byte) string {
	if len(b) == 0 {
		return "-"
	}
	return hex.EncodeToString(b)
}

func spanKindString(k tracev1.Span_SpanKind) string {
	switch k {
	case tracev1.Span_SPAN_KIND_SERVER:
		return "server"
	case tracev1.Span_SPAN_KIND_CLIENT:
		return "client"
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return "producer"
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return "consumer"
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return "internal"
	default:
		return "unspecified"
	}
}

func metricValueString(dp *metricsv1.NumberDataPoint) string {
	switch {
	case dp.GetAsDouble() != 0:
		return strconv.FormatFloat(dp.GetAsDouble(), 'f', -1, 64)
	default:
		return strconv.FormatInt(dp.GetAsInt(), 10)
	}
}

func histogramSummary(dp *metricsv1.HistogramDataPoint) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("count=%d", dp.GetCount()))
	parts = append(parts, fmt.Sprintf("sum=%.3f", dp.GetSum()))
	parts = append(parts, fmt.Sprintf("min=%.3f", dp.GetMin()))
	parts = append(parts, fmt.Sprintf("max=%.3f", dp.GetMax()))
	buckets := make([]string, len(dp.GetBucketCounts()))
	for i, c := range dp.GetBucketCounts() {
		buckets[i] = strconv.FormatUint(c, 10)
	}
	parts = append(parts, fmt.Sprintf("buckets=[%s]", strings.Join(buckets, ",")))
	return strings.Join(parts, " ")
}

func expHistogramSummary(dp *metricsv1.ExponentialHistogramDataPoint) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("count=%d", dp.GetCount()))
	parts = append(parts, fmt.Sprintf("sum=%.3f", dp.GetSum()))
	parts = append(parts, fmt.Sprintf("scale=%d", dp.GetScale()))
	parts = append(parts, fmt.Sprintf("zeroCount=%d", dp.GetZeroCount()))
	parts = append(parts, fmt.Sprintf("min=%.3f", dp.GetMin()))
	parts = append(parts, fmt.Sprintf("max=%.3f", dp.GetMax()))
	if pos := dp.GetPositive(); pos != nil {
		buckets := make([]string, len(pos.GetBucketCounts()))
		for i, c := range pos.GetBucketCounts() {
			buckets[i] = strconv.FormatUint(c, 10)
		}
		parts = append(parts, fmt.Sprintf("positive=[offset=%d,buckets=%s]", pos.GetOffset(), strings.Join(buckets, ",")))
	}
	return strings.Join(parts, " ")
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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", handleLogs)
	mux.HandleFunc("/v1/metrics", handleMetrics)
	mux.HandleFunc("/v1/traces", handleTraces)
	mux.HandleFunc("/api/v1/logs", handleAPILogs)
	mux.HandleFunc("/api/v1/metrics", handleAPIMetrics)
	mux.HandleFunc("/api/v1/traces", handleAPITraces)
	mux.HandleFunc("/api/logs", handleAPILogs)
	mux.HandleFunc("/ui", handleUI)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/ui", http.StatusFound)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

const htmlUI = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>OTLP Telemetry Viewer</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0d1117;color:#c9d1d9;padding:20px}
h1{font-size:1.5em;margin-bottom:4px;color:#58a6ff}
.subtitle{color:#8b949e;font-size:0.85em;margin-bottom:16px}
.tabs{display:flex;gap:0;margin-bottom:16px;border-bottom:2px solid #21262d}
.tab{padding:8px 20px;cursor:pointer;color:#8b949e;font-size:14px;font-weight:600;border:none;background:none;border-bottom:2px solid transparent;margin-bottom:-2px;transition:color .15s,border-color .15s}
.tab:hover{color:#c9d1d9}
.tab.active{color:#58a6ff;border-bottom-color:#58a6ff}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{padding:8px 12px;text-align:left;border-bottom:1px solid #21262d}
th{background:#161b22;color:#8b949e;font-weight:600;position:sticky;top:0}
tr:hover{background:#1c2128}
td{max-width:400px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.empty{text-align:center;padding:40px;color:#8b949e}
.status{color:#8b949e;font-size:0.85em;margin-bottom:12px}
.mono{font-family:'SF Mono','Fira Code',monospace;font-size:12px}
.severity{text-transform:uppercase;font-weight:600;font-size:11px;padding:2px 6px;border-radius:4px}
.sev-ERROR,.sev-FATAL{background:#da3633;color:#fff}
.sev-WARN{background:#d29922;color:#000}
.sev-INFO,.sev-INFORMATION{background:#238636;color:#fff}
.sev-DEBUG,.sev-TRACE{background:#484f58;color:#fff}
.badge{font-size:11px;padding:2px 6px;border-radius:4px;font-weight:600}
.badge-sum{background:#1f6feb;color:#fff}
.badge-gauge{background:#bf4b8a;color:#fff}
.badge-histogram{background:#8957e5;color:#fff}
.badge-exponential_histogram{background:#1a7f37;color:#fff}
.badge-server{background:#1f6feb;color:#fff}
.badge-client{background:#bf4b8a;color:#fff}
.badge-producer{background:#d29922;color:#000}
.badge-consumer{background:#8957e5;color:#fff}
.badge-internal{background:#8b949e;color:#fff}
</style>
</head>
<body>
<h1>OTLP Telemetry Viewer</h1>
<div class="subtitle">Ingest via POST /v1/logs /v1/metrics /v1/traces</div>
<div class="tabs">
  <button class="tab active" data-signal="logs">Logs</button>
  <button class="tab" data-signal="metrics">Metrics</button>
  <button class="tab" data-signal="traces">Spans</button>
</div>
<div class="status" id="status">Loading...</div>
<div style="overflow-x:auto">
<table>
<thead id="thead"></thead>
<tbody id="tbody"></tbody>
</table>
</div>
<script>
(function(){
  var signal="logs";
  var endpoint="/api/v1/logs";
  var labels={logs:"log record",metrics:"data point",traces:"span"};

  var schemas={
    logs:[
      {key:"timestamp",label:"Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"severity",label:"Severity",render:function(v){return '<span class="severity sev-'+v.replace(/\d.*/,"")+'">'+v+'</span>'}},
      {key:"body",label:"Body"},
      {key:"trace_id",label:"Trace ID",cls:"mono"},
      {key:"span_id",label:"Span ID",cls:"mono"}
    ],
    metrics:[
      {key:"timestamp",label:"Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"name",label:"Name"},
      {key:"type",label:"Type",render:function(v){return '<span class="badge badge-'+v+'">'+v+'</span>'}},
      {key:"value",label:"Value"}
    ],
    traces:[
      {key:"timestamp",label:"Start Time",cls:"mono"},
      {key:"service",label:"Service"},
      {key:"scope",label:"Scope"},
      {key:"name",label:"Name"},
      {key:"kind",label:"Kind",render:function(v){return '<span class="badge badge-'+v+'">'+v+'</span>'}},
      {key:"duration",label:"Duration",cls:"mono"},
      {key:"trace_id",label:"Trace ID",cls:"mono"},
      {key:"span_id",label:"Span ID",cls:"mono"}
    ]
  };

  function renderHead(){var s=schemas[signal];document.getElementById("thead").innerHTML='<tr>'+s.map(function(c){return'<th>'+c.label+'</th>'}).join("")+'</tr>'}

  function renderBody(rows){
    var s=schemas[signal],tbody=document.getElementById("tbody");
    if(rows.length===0){
      var ep=endpoint.replace("/api","");tbody.innerHTML='<tr><td colspan="'+s.length+'" class="empty">Waiting for '+signal+' &mdash; POST to '+ep+'</td></tr>';
      return
    }
    tbody.innerHTML=rows.slice().reverse().map(function(r){return'<tr>'+s.map(function(c){
      var raw=c.render?c.render(r[c.key]):r[c.key];
      return'<td'+(c.cls?' class="'+c.cls+'"':'')+'>'+raw+'</td>'
    }).join("")+'</tr>'}).join("")
  }

  async function poll(){var status=document.getElementById("status");try{var res=await fetch(endpoint);var rows=await res.json();status.textContent=rows.length+" "+labels[signal]+(rows.length!==1?"s":"")+"  ·  auto-refreshing every 2s";renderBody(rows)}catch(e){status.textContent="Error: "+e.message;document.getElementById("tbody").innerHTML=""};setTimeout(poll,2000)}

  document.querySelectorAll(".tab").forEach(function(t){t.addEventListener("click",function(){
    document.querySelectorAll(".tab").forEach(function(x){x.classList.remove("active")});this.classList.add("active");
    signal=this.dataset.signal;endpoint="/api/v1/"+signal;document.getElementById("status").textContent="Loading...";
    renderHead();poll()
  })});

  renderHead();poll()
})();
</script>
</body>
</html>`
