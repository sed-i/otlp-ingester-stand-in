package main

import (
	"regexp"
	"strings"
	"sync"
	"time"
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

type LogFilter struct {
	Body         string
	BodyContains string
	BodyRegex    *regexp.Regexp
	Severity     string
	Service      string
	Attrs        map[string]string
	AttrWildcard map[string]string
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

type MetricFilter struct {
	Name         string
	NameContains string
	Type         string
	Service      string
	Attrs        map[string]string
	AttrWildcard map[string]string
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

type SpanFilter struct {
	Name         string
	Kind         string
	Service      string
	TraceID      string
	SpanID       string
	Attrs        map[string]string
	AttrWildcard map[string]string
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

func filterTime(ts string, from, to *time.Time) bool {
	if from == nil && to == nil {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return false
	}
	if from != nil && t.Before(*from) {
		return false
	}
	if to != nil && t.After(*to) {
		return false
	}
	return true
}

func matchAttrs(attrs map[string]string, exact, wildcard map[string]string) bool {
	for k, v := range exact {
		rv, ok := attrs[k]
		if !ok || rv != v {
			return false
		}
	}
	for k, prefix := range wildcard {
		rv, ok := attrs[k]
		if !ok || !strings.HasPrefix(rv, prefix) {
			return false
		}
	}
	return true
}

func (s *LogStore) Filter(f LogFilter) []FlatLogRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []FlatLogRecord
	skipped := 0
	for _, r := range s.records {
		if f.Service != "" && r.Service != f.Service {
			continue
		}
		if f.Severity != "" && r.Severity != f.Severity {
			continue
		}
		if f.Body != "" && r.Body != f.Body {
			continue
		}
		if f.BodyContains != "" && !strings.Contains(r.Body, f.BodyContains) {
			continue
		}
		if f.BodyRegex != nil && !f.BodyRegex.MatchString(r.Body) {
			continue
		}
		if !filterTime(r.Timestamp, f.From, f.To) {
			continue
		}
		if !matchAttrs(r.Attributes, f.Attrs, f.AttrWildcard) {
			continue
		}
		if f.Offset > 0 {
			if skipped < f.Offset {
				skipped++
				continue
			}
		}
		out = append(out, r)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	if out == nil {
		out = []FlatLogRecord{}
	}
	return out
}

func (s *MetricStore) Filter(f MetricFilter) []FlatMetricRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []FlatMetricRecord
	skipped := 0
	for _, r := range s.records {
		if f.Service != "" && r.Service != f.Service {
			continue
		}
		if f.Name != "" && r.Name != f.Name {
			continue
		}
		if f.NameContains != "" && !strings.Contains(r.Name, f.NameContains) {
			continue
		}
		if f.Type != "" && r.Type != f.Type {
			continue
		}
		if !filterTime(r.Timestamp, f.From, f.To) {
			continue
		}
		if !matchAttrs(r.Attributes, f.Attrs, f.AttrWildcard) {
			continue
		}
		if f.Offset > 0 {
			if skipped < f.Offset {
				skipped++
				continue
			}
		}
		out = append(out, r)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	if out == nil {
		out = []FlatMetricRecord{}
	}
	return out
}

func (s *SpanStore) Filter(f SpanFilter) []FlatSpanRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []FlatSpanRecord
	skipped := 0
	for _, r := range s.records {
		if f.Service != "" && r.Service != f.Service {
			continue
		}
		if f.Name != "" && r.Name != f.Name {
			continue
		}
		if f.Kind != "" && r.Kind != f.Kind {
			continue
		}
		if f.TraceID != "" && !strings.EqualFold(r.TraceID, f.TraceID) {
			continue
		}
		if f.SpanID != "" && !strings.EqualFold(r.SpanID, f.SpanID) {
			continue
		}
		if !filterTime(r.Timestamp, f.From, f.To) {
			continue
		}
		if !matchAttrs(r.Attributes, f.Attrs, f.AttrWildcard) {
			continue
		}
		if f.Offset > 0 {
			if skipped < f.Offset {
				skipped++
				continue
			}
		}
		out = append(out, r)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	if out == nil {
		out = []FlatSpanRecord{}
	}
	return out
}
