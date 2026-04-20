package middleware

import "context"

// Correlation carries distributed-tracing identifiers attached to the
// request context by an upstream tracer adapter. The framework itself
// only reads these values (in LoggerMiddleware) so log lines can be
// joined to traces without forcing a tracer SDK into the core go.mod.
//
// A real adapter (for example, a future github.com/fastygo/otel package)
// populates Correlation via WithCorrelation; the no-op default leaves
// the slot empty and the logger silently omits the fields.
type Correlation struct {
	// TraceID is the W3C-style 32-character hex trace identifier
	// populated by an upstream tracer adapter. Empty means "no trace".
	TraceID string
	// SpanID is the 16-character hex span identifier populated by
	// the same adapter. Empty means "no span".
	SpanID string
}

type correlationKey struct{}

// WithCorrelation returns a child context that carries c. Pass an empty
// Correlation (zero values) to clear any inherited correlation.
func WithCorrelation(ctx context.Context, c Correlation) context.Context {
	return context.WithValue(ctx, correlationKey{}, c)
}

// CorrelationFromContext returns the Correlation attached to ctx, or the
// zero value if none was set. Never returns nil — callers should compare
// fields against "" before logging them.
func CorrelationFromContext(ctx context.Context) Correlation {
	if ctx == nil {
		return Correlation{}
	}
	if c, ok := ctx.Value(correlationKey{}).(Correlation); ok {
		return c
	}
	return Correlation{}
}
