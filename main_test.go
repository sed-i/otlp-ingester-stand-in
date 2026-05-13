package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
)

type expectedLogData struct {
	SeverityText  string
	AttributeKeys []string
}

func parseExpectedData(t *testing.T) expectedLogData {
	t.Helper()

	payload, err := os.ReadFile("testdata/logs.json")
	if err != nil {
		t.Fatalf("failed to read testdata/logs.json: %v", err)
	}

	var doc struct {
		ResourceLogs []struct {
			ScopeLogs []struct {
				LogRecords []struct {
					SeverityText string `json:"severityText"`
					Attributes   []struct {
						Key string `json:"key"`
					} `json:"attributes"`
				} `json:"logRecords"`
			} `json:"scopeLogs"`
		} `json:"resourceLogs"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("failed to parse testdata/logs.json: %v", err)
	}

	if len(doc.ResourceLogs) == 0 || len(doc.ResourceLogs[0].ScopeLogs) == 0 || len(doc.ResourceLogs[0].ScopeLogs[0].LogRecords) == 0 {
		t.Fatal("testdata/logs.json has no log records")
	}

	lr := doc.ResourceLogs[0].ScopeLogs[0].LogRecords[0]

	var keys []string
	for _, a := range lr.Attributes {
		keys = append(keys, a.Key)
	}
	sort.Strings(keys)

	return expectedLogData{
		SeverityText:  lr.SeverityText,
		AttributeKeys: keys,
	}
}

func TestLogIngestion(t *testing.T) {
	t.Run("Given a valid OTLP JSON payload", func(t *testing.T) {
		t.Run("When POST to /v1/logs", func(t *testing.T) {
			t.Run("Then it returns 200 and received count", func(t *testing.T) {
				store.records = nil

				payload, err := os.ReadFile("testdata/logs.json")
				if err != nil {
					t.Fatalf("failed to read testdata/logs.json: %v", err)
				}

				req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(string(payload)))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				handleLogs(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
				}

				var resp struct {
					Received int `json:"received"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Received != 1 {
					t.Errorf("expected 1 log record received, got %d", resp.Received)
				}
			})

			t.Run("Then the log record is stored correctly", func(t *testing.T) {
				store.records = nil
				want := parseExpectedData(t)

				payload, err := os.ReadFile("testdata/logs.json")
				if err != nil {
					t.Fatalf("failed to read testdata/logs.json: %v", err)
				}

				req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(string(payload)))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				handleLogs(rec, req)

				records := store.GetAll()
				if len(records) != 1 {
					t.Fatalf("expected 1 stored record, got %d", len(records))
				}

				r := records[0]

				if r.Service != "my.service" {
					t.Errorf("expected service 'my.service', got '%s'", r.Service)
				}
				if r.Scope != "my.library" {
					t.Errorf("expected scope 'my.library', got '%s'", r.Scope)
				}
				if r.Severity != want.SeverityText {
					t.Errorf("expected severity %q, got %q", want.SeverityText, r.Severity)
				}
				if r.Body != "Example log record" {
					t.Errorf("expected body 'Example log record', got '%s'", r.Body)
				}
				if r.TraceID == "" || r.TraceID == "-" {
					t.Error("expected non-empty traceID")
				}
				if r.SpanID == "" || r.SpanID == "-" {
					t.Error("expected non-empty spanID")
				}
				if r.Timestamp == "" {
					t.Error("expected non-empty timestamp")
				}

				var gotKeys []string
				for k := range r.Attributes {
					gotKeys = append(gotKeys, k)
				}
				sort.Strings(gotKeys)

				if !slices.Equal(gotKeys, want.AttributeKeys) {
					t.Errorf("attribute keys mismatch\nwant: %v\ngot:  %v", want.AttributeKeys, gotKeys)
				}
			})

			t.Run("Then GET /api/v1/logs returns the stored records", func(t *testing.T) {
				store.records = nil
				want := parseExpectedData(t)

				payload, err := os.ReadFile("testdata/logs.json")
				if err != nil {
					t.Fatalf("failed to read testdata/logs.json: %v", err)
				}

				postReq := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(string(payload)))
				postReq.Header.Set("Content-Type", "application/json")
				postRec := httptest.NewRecorder()
				handleLogs(postRec, postReq)

				getReq := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
				getRec := httptest.NewRecorder()
				handleAPILogs(getRec, getReq)

				if getRec.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", getRec.Code)
				}

				var logs []FlatLogRecord
				if err := json.Unmarshal(getRec.Body.Bytes(), &logs); err != nil {
					t.Fatalf("failed to decode logs: %v", err)
				}
				if len(logs) != 1 {
					t.Fatalf("expected 1 log, got %d", len(logs))
				}
				if logs[0].Body != "Example log record" {
					t.Errorf("expected body 'Example log record', got '%s'", logs[0].Body)
				}
				if logs[0].Severity != want.SeverityText {
					t.Errorf("expected severity %q, got %q", want.SeverityText, logs[0].Severity)
				}

				var gotKeys []string
				for k := range logs[0].Attributes {
					gotKeys = append(gotKeys, k)
				}
				sort.Strings(gotKeys)

				if !slices.Equal(gotKeys, want.AttributeKeys) {
					t.Errorf("attribute keys mismatch\nwant: %v\ngot:  %v", want.AttributeKeys, gotKeys)
				}
			})
		})
	})

	t.Run("Given an invalid request", func(t *testing.T) {
		t.Run("When POSTing with wrong method", func(t *testing.T) {
			t.Run("Then it returns 405", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/v1/logs", nil)
				rec := httptest.NewRecorder()
				handleLogs(rec, req)
				if rec.Code != http.StatusMethodNotAllowed {
					t.Errorf("expected 405, got %d", rec.Code)
				}
			})
		})

		t.Run("When POSTing invalid JSON", func(t *testing.T) {
			t.Run("Then it returns 400", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(`not json`))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				handleLogs(rec, req)
				if rec.Code != http.StatusBadRequest {
					t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
				}
			})
		})
	})

	t.Run("Given an empty store", func(t *testing.T) {
		t.Run("When GET /api/v1/logs", func(t *testing.T) {
			t.Run("Then it returns an empty array", func(t *testing.T) {
				store.records = nil

				req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
				rec := httptest.NewRecorder()
				handleAPILogs(rec, req)

				var logs []FlatLogRecord
				if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil {
					t.Fatalf("failed to decode: %v", err)
				}
				if len(logs) != 0 {
					t.Errorf("expected 0 logs, got %d", len(logs))
				}
			})
		})
	})
}