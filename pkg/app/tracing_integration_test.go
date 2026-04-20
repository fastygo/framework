package app

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/observability"
)

// fixedTracer is a deterministic Tracer used by the integration test:
// every Start returns the same SpanContext and the same Span pointer
// so we can assert that LoggerMiddleware picked up trace_id / span_id.
type fixedTracer struct{}

type fixedSpan struct{}

func (fixedSpan) End() {}

type fixedTracerKey struct{}

func (fixedTracer) Start(ctx context.Context, _ string) (context.Context, observability.Span) {
	return context.WithValue(ctx, fixedTracerKey{}, true), fixedSpan{}
}

func (fixedTracer) SpanContextFromContext(ctx context.Context) observability.SpanContext {
	if ctx.Value(fixedTracerKey{}) == nil {
		return observability.SpanContext{}
	}
	return observability.SpanContext{TraceID: "deadbeef", SpanID: "cafef00d"}
}

func TestApp_Tracer_PropagatesIntoLogger(t *testing.T) {
	// Sequential: this test reroutes slog.SetDefault.
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithTracer(fixedTracer{}).
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/no-route", nil))

	// Find the http.response line and assert correlation fields.
	var found bool
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["msg"] != "http.response" {
			continue
		}
		if entry["trace_id"] != "deadbeef" || entry["span_id"] != "cafef00d" {
			t.Fatalf("log line missing correlation: %v\n---\n%s", entry, buf.String())
		}
		found = true
	}
	if !found {
		t.Fatalf("no http.response log line emitted\n---\n%s", buf.String())
	}
}

func TestApp_NoTracer_NoCorrelationFields(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/no-route", nil))

	if strings.Contains(buf.String(), "trace_id") || strings.Contains(buf.String(), "span_id") {
		t.Fatalf("correlation fields leaked without tracer:\n%s", buf.String())
	}
}
