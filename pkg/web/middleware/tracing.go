package middleware

import (
	"net/http"

	"github.com/fastygo/framework/pkg/observability"
)

// TracingMiddleware bridges an observability.Tracer into the HTTP
// pipeline. For every request it:
//
//  1. Starts a span named "http <METHOD> <PATH>" using tracer.
//  2. Reads the resulting SpanContext via tracer.SpanContextFromContext.
//  3. Attaches the trace_id/span_id pair to the request context as a
//     Correlation, so the downstream LoggerMiddleware can decorate
//     every log line.
//  4. Defers Span.End so the span closes even on panic (RecoverMiddleware
//     should sit closer to the handler in the chain).
//
// Pass observability.NoopTracer{} (or nil) to disable: the middleware
// degrades to a passthrough with no allocations.
func TracingMiddleware(tracer observability.Tracer) Middleware {
	if tracer == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	if _, ok := tracer.(observability.NoopTracer); ok {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), spanName(r))
			defer span.End()

			sc := tracer.SpanContextFromContext(ctx)
			if sc.IsValid() {
				ctx = WithCorrelation(ctx, Correlation{
					TraceID: sc.TraceID,
					SpanID:  sc.SpanID,
				})
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// spanName builds a low-cardinality span name. We deliberately use the
// raw URL.Path: routing-aware naming (e.g. "GET /users/:id") requires
// the framework to know about route templates, which the bare http.mux
// does not expose. Adapter packages can wrap TracingMiddleware with
// their own naming once routing metadata is available.
func spanName(r *http.Request) string {
	return "http " + r.Method + " " + r.URL.Path
}
