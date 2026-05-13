package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

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
