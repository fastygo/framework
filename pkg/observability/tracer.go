// Package observability defines the framework's tracing contract.
//
// Goal: let an application enrich every log line with trace_id/span_id
// (so log queries can pivot to a Tempo/Jaeger trace) without forcing
// the framework's go.mod to pull in any tracing SDK.
//
// The framework only consumes the Tracer interface. A real adapter
// (today: planned as github.com/fastygo/otel) implements Tracer using
// the OpenTelemetry SDK and is wired into the application by the
// composition root via AppBuilder.WithTracer. Applications that do
// not need tracing pass NoopTracer{} (or simply skip WithTracer — the
// builder defaults to no-op).
//
// This split keeps the framework dependency footprint at exactly one
// indirect dependency (go.uber.org/goleak, used in tests only) while
// still offering first-class tracing integration when needed.
package observability

import "context"

// SpanContext carries the immutable identifiers attached to the
// currently-active span. Empty strings indicate "no active span" — the
// no-op tracer always returns the zero value.
//
// Field naming matches the W3C trace-context spec (TraceID, SpanID)
// so adapters mapping from OpenTelemetry, OpenCensus, or Datadog APM
// can fill them without further translation.
type SpanContext struct {
	// TraceID is the 32-character hex trace identifier (W3C
	// trace-context). Empty means "no trace yet".
	TraceID string
	// SpanID is the 16-character hex span identifier (W3C
	// trace-context). Empty means "no span yet".
	SpanID string
}

// IsValid reports whether sc identifies a real span. Useful in
// middleware to decide whether to emit correlation log fields.
func (sc SpanContext) IsValid() bool { return sc.TraceID != "" }

// Span represents an active unit of work returned by Tracer.Start.
// The framework only ever calls End on it; the underlying SDK owns
// any richer span manipulation (attributes, events, status).
//
// Implementations must be safe to call End multiple times: middleware
// uses defer span.End(), but error paths may also call End explicitly.
type Span interface {
	End()
}

// Tracer is the minimum surface the framework needs to participate in
// distributed tracing. Implementations live outside the framework;
// the only built-in is NoopTracer for opt-out applications.
//
// Start opens a new span as a child of any span already attached to
// ctx and returns the new context plus the span handle. Implementations
// may return the same ctx (and a no-op span) when tracing is disabled
// for performance reasons.
//
// SpanContextFromContext returns the SpanContext of the span attached
// to ctx, or the zero value when no span is active.
type Tracer interface {
	Start(ctx context.Context, spanName string) (context.Context, Span)
	SpanContextFromContext(ctx context.Context) SpanContext
}

// NoopTracer is the default implementation: every Start returns the
// input context and a span whose End is a no-op; SpanContextFromContext
// always returns the zero SpanContext. Using NoopTracer adds zero
// allocations on the hot path.
type NoopTracer struct{}

// Start implements Tracer.
func (NoopTracer) Start(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}

// SpanContextFromContext implements Tracer. Always returns the zero
// SpanContext so middleware writes no trace_id/span_id fields.
func (NoopTracer) SpanContextFromContext(_ context.Context) SpanContext {
	return SpanContext{}
}

type noopSpan struct{}

func (noopSpan) End() {}

// Compile-time guard that NoopTracer satisfies Tracer.
var _ Tracer = NoopTracer{}
