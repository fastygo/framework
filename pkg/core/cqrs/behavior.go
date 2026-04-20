package cqrs

import (
	"context"
)

// HandlerFunc is the untyped low-level handler shape used inside the
// Dispatcher pipeline. The typed RegisterCommand / RegisterQuery
// helpers wrap user handlers into a HandlerFunc; user code rarely
// constructs one directly.
type HandlerFunc func(context.Context, any) (any, error)

// PipelineBehavior wraps the handler chain. Implementations may
// short-circuit (return without calling next), pre/post-process, or
// retry. Multiple behaviors compose: see NewDispatcher for ordering.
type PipelineBehavior interface {
	Handle(ctx context.Context, request any, next HandlerFunc) (any, error)
}

// HandlerNotFoundError is returned (wrapped in a DomainError) when
// Dispatch receives a request type with no registered handler. It is
// exported so callers can use errors.As to extract the request type.
type HandlerNotFoundError struct {
	// RequestType is the textual type identifier of the unhandled
	// request (typically "package.TypeName").
	RequestType string
}

// Error implements the error interface.
func (e HandlerNotFoundError) Error() string {
	return "no handler registered for request: " + e.RequestType
}
