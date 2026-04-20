package cqrs

import (
	"context"
	"errors"
	"testing"

	"github.com/fastygo/framework/pkg/core"
)

type createUser struct{ Name string }
type listUsers struct{ Limit int }

func TestDispatcher_RegisterAndDispatchCommand(t *testing.T) {
	d := NewDispatcher()
	d.RegisterCommandHandler(requestKey(createUser{}), func(_ context.Context, req any) (any, error) {
		c, _ := req.(createUser)
		return "id-" + c.Name, nil
	})

	got, err := d.Dispatch(context.Background(), createUser{Name: "ada"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != "id-ada" {
		t.Fatalf("result: got %v, want %q", got, "id-ada")
	}
}

func TestDispatcher_RegisterAndDispatchQuery(t *testing.T) {
	d := NewDispatcher()
	d.RegisterQueryHandler(requestKey(listUsers{}), func(_ context.Context, req any) (any, error) {
		q, _ := req.(listUsers)
		return q.Limit * 2, nil
	})

	got, err := d.Dispatch(context.Background(), listUsers{Limit: 5})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != 10 {
		t.Fatalf("result: got %v, want 10", got)
	}
}

func TestDispatcher_CommandHasPrecedenceOverQuery(t *testing.T) {
	// Same request type registered on both sides: command must win.
	d := NewDispatcher()
	d.RegisterCommandHandler(requestKey(createUser{}), func(context.Context, any) (any, error) {
		return "command", nil
	})
	d.RegisterQueryHandler(requestKey(createUser{}), func(context.Context, any) (any, error) {
		return "query", nil
	})

	got, err := d.Dispatch(context.Background(), createUser{Name: "x"})
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if got != "command" {
		t.Fatalf("expected command handler to win, got %v", got)
	}
}

func TestDispatcher_MissingHandler_ReturnsWrappedNotFound(t *testing.T) {
	d := NewDispatcher()

	_, err := d.Dispatch(context.Background(), createUser{Name: "y"})
	if err == nil {
		t.Fatalf("Dispatch with no handler must return an error")
	}

	var de core.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("error must be a DomainError, got %T: %v", err, err)
	}
	if de.Code != core.ErrorCodeNotFound {
		t.Fatalf("code: got %q, want %q", de.Code, core.ErrorCodeNotFound)
	}

	var hnf HandlerNotFoundError
	if !errors.As(err, &hnf) {
		t.Fatalf("error must wrap HandlerNotFoundError for type-aware handling, got: %v", err)
	}
	if hnf.RequestType == "" {
		t.Fatalf("HandlerNotFoundError.RequestType must be populated")
	}
}

func TestHandlerNotFoundError_Error(t *testing.T) {
	got := HandlerNotFoundError{RequestType: "pkg.Foo"}.Error()
	want := "no handler registered for request: pkg.Foo"
	if got != want {
		t.Fatalf("Error(): got %q, want %q", got, want)
	}
}
