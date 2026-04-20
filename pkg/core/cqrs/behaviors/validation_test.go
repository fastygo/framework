package behaviors

import (
	"context"
	"errors"
	"testing"

	"github.com/fastygo/framework/pkg/core"
)

type validatableOK struct{}

func (validatableOK) Validate() error { return nil }

type validatableBad struct{ Reason string }

func (v validatableBad) Validate() error { return errors.New(v.Reason) }

type plainRequest struct{ Field int }

func TestValidation_PassesWhenValidateReturnsNil(t *testing.T) {
	b := Validation{}
	called := false

	got, err := b.Handle(context.Background(), validatableOK{}, func(context.Context, any) (any, error) {
		called = true
		return "downstream", nil
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !called {
		t.Fatalf("next handler must run when Validate returns nil")
	}
	if got != "downstream" {
		t.Fatalf("result: got %v, want %q", got, "downstream")
	}
}

func TestValidation_WrapsValidateErrorAsValidationDomainError(t *testing.T) {
	b := Validation{}
	called := false

	_, err := b.Handle(context.Background(), validatableBad{Reason: "name too short"},
		func(context.Context, any) (any, error) {
			called = true
			return nil, nil
		},
	)
	if called {
		t.Fatalf("next handler must not run when Validate fails")
	}
	if err == nil {
		t.Fatalf("Handle must return an error when Validate fails")
	}

	var de core.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("error must be a DomainError, got %T: %v", err, err)
	}
	if de.Code != core.ErrorCodeValidation {
		t.Fatalf("code: got %q, want %q", de.Code, core.ErrorCodeValidation)
	}
	if de.Cause == nil || de.Cause.Error() != "name too short" {
		t.Fatalf("DomainError must wrap the original Validate error, got Cause=%v", de.Cause)
	}
}

func TestValidation_PassesThroughWhenNoValidateMethod(t *testing.T) {
	b := Validation{}
	called := false

	if _, err := b.Handle(context.Background(), plainRequest{Field: 1}, func(context.Context, any) (any, error) {
		called = true
		return "ok", nil
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !called {
		t.Fatalf("requests without Validate() must pass through")
	}
}

func TestValidation_NilRequest_PassesThrough(t *testing.T) {
	b := Validation{}
	called := false

	if _, err := b.Handle(context.Background(), nil, func(context.Context, any) (any, error) {
		called = true
		return "ok", nil
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !called {
		t.Fatalf("nil request must pass through to next handler")
	}
}
