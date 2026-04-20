package cqrs

import (
	"context"
	"errors"
	"testing"

	"github.com/fastygo/framework/pkg/core"
)

type updateUser struct{ ID, Name string }
type updateUserResult struct{ Updated bool }

type updateUserHandler struct {
	called bool
	got    updateUser
	out    updateUserResult
	err    error
}

func (h *updateUserHandler) Handle(_ context.Context, c updateUser) (updateUserResult, error) {
	h.called = true
	h.got = c
	return h.out, h.err
}

func TestRegisterCommand_DispatchCommand_RoundTrip(t *testing.T) {
	d := NewDispatcher()
	h := &updateUserHandler{out: updateUserResult{Updated: true}}
	RegisterCommand[updateUser, updateUserResult](d, h)

	got, err := DispatchCommand[updateUser, updateUserResult](
		context.Background(), d, updateUser{ID: "1", Name: "ada"},
	)
	if err != nil {
		t.Fatalf("DispatchCommand: %v", err)
	}
	if !h.called {
		t.Fatalf("typed handler was never invoked")
	}
	if h.got.ID != "1" || h.got.Name != "ada" {
		t.Fatalf("typed handler received %+v", h.got)
	}
	if !got.Updated {
		t.Fatalf("DispatchCommand result: got %+v", got)
	}
}

type readUser struct{ ID string }
type readUserResult struct{ Email string }

type readUserHandler struct {
	out readUserResult
}

func (h readUserHandler) Handle(_ context.Context, q readUser) (readUserResult, error) {
	return h.out, nil
}

func TestRegisterQuery_DispatchQuery_RoundTrip(t *testing.T) {
	d := NewDispatcher()
	RegisterQuery[readUser, readUserResult](d, readUserHandler{
		out: readUserResult{Email: "ada@example.com"},
	})

	got, err := DispatchQuery[readUser, readUserResult](
		context.Background(), d, readUser{ID: "1"},
	)
	if err != nil {
		t.Fatalf("DispatchQuery: %v", err)
	}
	if got.Email != "ada@example.com" {
		t.Fatalf("DispatchQuery result: got %+v", got)
	}
}

func TestDispatchCommand_NilResultReturnsZero(t *testing.T) {
	d := NewDispatcher()
	// Register a raw HandlerFunc that returns (nil, nil) so the typed
	// dispatch path falls into the "result == nil" branch.
	d.RegisterCommandHandler(requestKey(updateUser{}), func(context.Context, any) (any, error) {
		return nil, nil
	})

	got, err := DispatchCommand[updateUser, updateUserResult](
		context.Background(), d, updateUser{ID: "x"},
	)
	if err != nil {
		t.Fatalf("DispatchCommand: %v", err)
	}
	if got != (updateUserResult{}) {
		t.Fatalf("nil result must yield zero value, got %+v", got)
	}
}

func TestDispatchCommand_HandlerWrongResultType_ReturnsInternal(t *testing.T) {
	d := NewDispatcher()
	d.RegisterCommandHandler(requestKey(updateUser{}), func(context.Context, any) (any, error) {
		return 12345, nil // not updateUserResult
	})

	_, err := DispatchCommand[updateUser, updateUserResult](
		context.Background(), d, updateUser{ID: "x"},
	)
	if err == nil {
		t.Fatalf("DispatchCommand with wrong-typed result must return an error")
	}

	var de core.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("error must be a DomainError, got %T: %v", err, err)
	}
	if de.Code != core.ErrorCodeInternal {
		t.Fatalf("code: got %q, want %q", de.Code, core.ErrorCodeInternal)
	}
}

func TestDispatchQuery_HandlerWrongResultType_ReturnsInternal(t *testing.T) {
	d := NewDispatcher()
	d.RegisterQueryHandler(requestKey(readUser{}), func(context.Context, any) (any, error) {
		return "not the right type", nil
	})

	_, err := DispatchQuery[readUser, readUserResult](
		context.Background(), d, readUser{ID: "x"},
	)
	if err == nil {
		t.Fatalf("DispatchQuery with wrong-typed result must return an error")
	}

	var de core.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("error must be a DomainError, got %T: %v", err, err)
	}
	if de.Code != core.ErrorCodeInternal {
		t.Fatalf("code: got %q, want %q", de.Code, core.ErrorCodeInternal)
	}
}

// wrongShape is registered for updateUser but the wrapped fn expects a
// different request type — exercise the wrapHandler type-mismatch path.
type wrongShape struct{}
type wrongShapeResult struct{}

func TestWrapHandler_TypeMismatch_ReturnsValidationError(t *testing.T) {
	d := NewDispatcher()
	// Register a handler that expects wrongShape, but route the
	// dispatch under updateUser's key. wrapHandler must reject the
	// incoming request type.
	wrapped := wrapHandler[wrongShape](func(_ context.Context, _ wrongShape) (any, error) {
		return wrongShapeResult{}, nil
	})
	d.RegisterCommandHandler(requestKey(updateUser{}), wrapped)

	_, err := d.Dispatch(context.Background(), updateUser{ID: "x"})
	if err == nil {
		t.Fatalf("Dispatch with mismatched typed handler must return an error")
	}

	var de core.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("error must be a DomainError, got %T: %v", err, err)
	}
	if de.Code != core.ErrorCodeValidation {
		t.Fatalf("code: got %q, want %q", de.Code, core.ErrorCodeValidation)
	}
}

func TestDispatchCommand_HandlerError_Propagates(t *testing.T) {
	d := NewDispatcher()
	want := errors.New("db down")
	h := &updateUserHandler{err: want}
	RegisterCommand[updateUser, updateUserResult](d, h)

	_, err := DispatchCommand[updateUser, updateUserResult](
		context.Background(), d, updateUser{ID: "x"},
	)
	if !errors.Is(err, want) {
		t.Fatalf("expected handler error to propagate, got %v", err)
	}
}
