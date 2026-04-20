package observability

import (
	"context"
	"testing"
)

type ctxMarkerKey struct{}

func TestNoopTracer_StartReturnsSameContext(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), ctxMarkerKey{}, "marker")
	got, span := NoopTracer{}.Start(ctx, "op")

	if got != ctx {
		t.Error("NoopTracer.Start must return the input ctx unchanged")
	}
	if span == nil {
		t.Fatal("NoopTracer.Start returned nil Span")
	}
	span.End()
	span.End() // double End must not panic
}

func TestNoopTracer_SpanContextIsZero(t *testing.T) {
	t.Parallel()

	sc := NoopTracer{}.SpanContextFromContext(context.Background())
	if sc.IsValid() {
		t.Errorf("expected invalid SpanContext, got %+v", sc)
	}
}

func TestSpanContext_IsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		sc   SpanContext
		want bool
	}{
		{"empty", SpanContext{}, false},
		{"only-span", SpanContext{SpanID: "s"}, false},
		{"with-trace", SpanContext{TraceID: "t", SpanID: "s"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sc.IsValid(); got != tc.want {
				t.Errorf("IsValid = %v, want %v", got, tc.want)
			}
		})
	}
}
