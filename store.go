package main

import "sync"

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
