package cqrs

import (
	"context"
	"errors"
	"testing"
)

// recorder captures the call order of pipeline behaviors and the
// final handler so we can assert composition semantics.
type recorder struct{ events []string }

func (r *recorder) record(name string) { r.events = append(r.events, name) }

type tagBehavior struct {
	rec *recorder
	tag string
}

func (b tagBehavior) Handle(ctx context.Context, req any, next HandlerFunc) (any, error) {
	b.rec.record(b.tag + ":before")
	result, err := next(ctx, req)
	b.rec.record(b.tag + ":after")
	return result, err
}

type pingRequest struct{}

func TestDispatcher_Pipeline_OrderOutermostFirst(t *testing.T) {
	rec := &recorder{}
	d := NewDispatcher(
		tagBehavior{rec: rec, tag: "outer"},
		tagBehavior{rec: rec, tag: "middle"},
		tagBehavior{rec: rec, tag: "inner"},
	)
	d.RegisterCommandHandler(requestKey(pingRequest{}), func(context.Context, any) (any, error) {
		rec.record("handler")
		return "ok", nil
	})

	if _, err := d.Dispatch(context.Background(), pingRequest{}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	want := []string{
		"outer:before",
		"middle:before",
		"inner:before",
		"handler",
		"inner:after",
		"middle:after",
		"outer:after",
	}
	if len(rec.events) != len(want) {
		t.Fatalf("event count: got %d (%v), want %d (%v)",
			len(rec.events), rec.events, len(want), want)
	}
	for i := range want {
		if rec.events[i] != want[i] {
			t.Fatalf("event[%d]: got %q, want %q (full: %v)",
				i, rec.events[i], want[i], rec.events)
		}
	}
}

// shortCircuit returns a fixed result without calling next, so the
// handler must never run.
type shortCircuit struct{ rec *recorder }

func (s shortCircuit) Handle(_ context.Context, _ any, _ HandlerFunc) (any, error) {
	s.rec.record("short-circuit")
	return "from-behavior", nil
}

func TestDispatcher_Pipeline_ShortCircuit(t *testing.T) {
	rec := &recorder{}
	d := NewDispatcher(shortCircuit{rec: rec})
	d.RegisterCommandHandler(requestKey(pingRequest{}), func(context.Context, any) (any, error) {
		rec.record("handler")
		return "from-handler", nil
	})

	got, err := d.Dispatch(context.Background(), pingRequest{})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != "from-behavior" {
		t.Fatalf("result: got %v, want %q", got, "from-behavior")
	}
	if len(rec.events) != 1 || rec.events[0] != "short-circuit" {
		t.Fatalf("handler must not run when behavior short-circuits, events=%v", rec.events)
	}
}

// errorBehavior wraps the handler error with extra context so we can
// verify behaviors observe the inner error.
type errorBehavior struct{ wrapMsg string }

func (b errorBehavior) Handle(ctx context.Context, req any, next HandlerFunc) (any, error) {
	res, err := next(ctx, req)
	if err != nil {
		return res, errors.New(b.wrapMsg + ": " + err.Error())
	}
	return res, nil
}

func TestDispatcher_Pipeline_BehaviorObservesHandlerError(t *testing.T) {
	d := NewDispatcher(errorBehavior{wrapMsg: "outer-saw"})
	handlerErr := errors.New("inner-failed")
	d.RegisterCommandHandler(requestKey(pingRequest{}), func(context.Context, any) (any, error) {
		return nil, handlerErr
	})

	_, err := d.Dispatch(context.Background(), pingRequest{})
	if err == nil {
		t.Fatalf("expected wrapped error, got nil")
	}
	want := "outer-saw: inner-failed"
	if err.Error() != want {
		t.Fatalf("wrapped error: got %q, want %q", err.Error(), want)
	}
}

func TestDispatcher_NoBehaviors_HandlerRunsDirectly(t *testing.T) {
	d := NewDispatcher() // zero behaviors
	d.RegisterCommandHandler(requestKey(pingRequest{}), func(context.Context, any) (any, error) {
		return 42, nil
	})

	got, err := d.Dispatch(context.Background(), pingRequest{})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != 42 {
		t.Fatalf("result: got %v, want 42", got)
	}
}
