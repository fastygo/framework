package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureSlog routes slog.Default() output into a buffer for the
// duration of fn and returns whatever was written. It restores the
// previous default handler before returning.
func captureSlog(t *testing.T, fn func()) string {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	fn()
	return buf.String()
}

func TestLogger_OmitsCorrelationWhenAbsent(t *testing.T) {
	out := captureSlog(t, func() {
		h := LoggerMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set(RequestIDHeader, "rid-1")
		h.ServeHTTP(httptest.NewRecorder(), req)
	})

	if strings.Contains(out, "trace_id") || strings.Contains(out, "span_id") {
		t.Fatalf("log unexpectedly contains correlation fields:\n%s", out)
	}
	if !strings.Contains(out, "rid-1") {
		t.Fatalf("log missing request_id rid-1:\n%s", out)
	}
}

func TestLogger_IncludesCorrelationWhenPresent(t *testing.T) {
	out := captureSlog(t, func() {
		h := LoggerMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set(RequestIDHeader, "rid-2")
		ctx := WithCorrelation(req.Context(), Correlation{
			TraceID: "trace-xyz",
			SpanID:  "span-789",
		})
		h.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))
	})

	// Parse the first JSON line and assert both fields are present.
	first := strings.SplitN(out, "\n", 2)[0]
	var entry map[string]any
	if err := json.Unmarshal([]byte(first), &entry); err != nil {
		t.Fatalf("parse json: %v\nraw=%s", err, first)
	}
	if entry["trace_id"] != "trace-xyz" {
		t.Errorf("trace_id = %v, want trace-xyz", entry["trace_id"])
	}
	if entry["span_id"] != "span-789" {
		t.Errorf("span_id = %v, want span-789", entry["span_id"])
	}
	if entry["request_id"] != "rid-2" {
		t.Errorf("request_id = %v, want rid-2", entry["request_id"])
	}
}

func TestLogger_PartialCorrelation(t *testing.T) {
	out := captureSlog(t, func() {
		h := LoggerMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		ctx := WithCorrelation(req.Context(), Correlation{TraceID: "only-trace"})
		h.ServeHTTP(httptest.NewRecorder(), req.WithContext(ctx))
	})

	if !strings.Contains(out, "only-trace") {
		t.Errorf("expected trace_id present:\n%s", out)
	}
	if strings.Contains(out, "span_id") {
		t.Errorf("span_id must be omitted when empty:\n%s", out)
	}
}
