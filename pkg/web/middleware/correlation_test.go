package middleware

import (
	"context"
	"testing"
)

func TestCorrelationFromContext_ReturnsZeroWhenAbsent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ctx  context.Context
	}{
		{"empty-context", context.Background()},
		// Pass nil to exercise the explicit nil-guard inside
		// CorrelationFromContext: callers in real middleware never
		// pass nil, but the guard is cheap insurance.
		{"nil-context", nil}, //nolint:staticcheck
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := CorrelationFromContext(tc.ctx); got.TraceID != "" || got.SpanID != "" {
				t.Fatalf("expected zero Correlation, got %+v", got)
			}
		})
	}
}

func TestWithCorrelation_RoundTripAndOverride(t *testing.T) {
	t.Parallel()

	in := Correlation{TraceID: "trace-abc", SpanID: "span-123"}
	ctx := WithCorrelation(context.Background(), in)

	if got := CorrelationFromContext(ctx); got != in {
		t.Fatalf("round trip = %+v, want %+v", got, in)
	}

	// A second WithCorrelation on the same ctx must shadow the first
	// — that's how a tracer adapter would replace stale correlation
	// after a Span has ended.
	ctx = WithCorrelation(ctx, Correlation{TraceID: "second"})
	if got := CorrelationFromContext(ctx); got.TraceID != "second" {
		t.Fatalf("TraceID = %q, want second (override)", got.TraceID)
	}
}
