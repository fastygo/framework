package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/fastygo/framework/pkg/observability"
)

// stubTracer is a minimal Tracer that injects a fixed SpanContext
// and records End calls so tests can assert on lifecycle.
type stubTracer struct {
	traceID string
	spanID  string
	ended   atomic.Int32
}

type stubSpan struct {
	t *stubTracer
}

func (s stubSpan) End() { s.t.ended.Add(1) }

type tracerCtxKey struct{}

func (st *stubTracer) Start(ctx context.Context, _ string) (context.Context, observability.Span) {
	return context.WithValue(ctx, tracerCtxKey{}, true), stubSpan{t: st}
}

func (st *stubTracer) SpanContextFromContext(ctx context.Context) observability.SpanContext {
	if ctx.Value(tracerCtxKey{}) == nil {
		return observability.SpanContext{}
	}
	return observability.SpanContext{TraceID: st.traceID, SpanID: st.spanID}
}

func TestTracingMiddleware_PopulatesCorrelation(t *testing.T) {
	t.Parallel()

	tracer := &stubTracer{traceID: "trace-1", spanID: "span-1"}
	var seen Correlation

	h := TracingMiddleware(tracer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = CorrelationFromContext(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	if seen.TraceID != "trace-1" || seen.SpanID != "span-1" {
		t.Errorf("Correlation = %+v, want trace-1/span-1", seen)
	}
	if got := tracer.ended.Load(); got != 1 {
		t.Errorf("Span.End called %d times, want 1", got)
	}
}

func TestTracingMiddleware_NilIsPassthrough(t *testing.T) {
	t.Parallel()

	called := false
	h := TracingMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("inner handler never invoked under nil tracer")
	}
}

func TestTracingMiddleware_NoopTracerIsPassthrough(t *testing.T) {
	t.Parallel()

	called := false
	h := TracingMiddleware(observability.NoopTracer{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if c := CorrelationFromContext(r.Context()); c.TraceID != "" {
			t.Errorf("noop tracer leaked correlation: %+v", c)
		}
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("inner handler never invoked under noop tracer")
	}
}
