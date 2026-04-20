package cqrs

import (
	"context"
	"fmt"

	"github.com/fastygo/framework/pkg/core"
)

func requestKey(request any) string {
	return fmt.Sprintf("%T", request)
}

func wrapHandler[T any](fn func(context.Context, T) (any, error)) HandlerFunc {
	return func(ctx context.Context, req any) (any, error) {
		typed, ok := req.(T)
		if !ok {
			return nil, core.NewDomainError(core.ErrorCodeValidation, "request type mismatch for handler")
		}
		return fn(ctx, typed)
	}
}

// RegisterCommand binds a typed CommandHandler[T, R] into dispatcher.
// The request type T is used as the routing key, so each T may have
// at most one registered handler.
func RegisterCommand[T any, R any](dispatcher *Dispatcher, handler CommandHandler[T, R]) {
	var req T
	dispatcher.RegisterCommandHandler(requestKey(req), wrapHandler(func(ctx context.Context, command T) (any, error) {
		return handler.Handle(ctx, command)
	}))
}

// RegisterQuery is the read-side counterpart to RegisterCommand.
func RegisterQuery[T any, R any](dispatcher *Dispatcher, handler QueryHandler[T, R]) {
	var req T
	dispatcher.RegisterQueryHandler(requestKey(req), wrapHandler(func(ctx context.Context, query T) (any, error) {
		return handler.Handle(ctx, query)
	}))
}

// DispatchCommand sends command through dispatcher and returns the
// typed result. It surfaces ErrorCodeInternal as a DomainError if the
// handler returns a value that does not match R.
func DispatchCommand[T any, R any](ctx context.Context, dispatcher *Dispatcher, command T) (R, error) {
	var zero R
	result, err := dispatchTyped(ctx, dispatcher, command, "command")
	if err != nil {
		return zero, err
	}
	if result == nil {
		return zero, nil
	}
	typed, ok := result.(R)
	if !ok {
		return zero, core.NewDomainError(core.ErrorCodeInternal, "command handler returned unexpected type")
	}
	return typed, nil
}

// DispatchQuery is the read-side counterpart to DispatchCommand.
func DispatchQuery[T any, R any](ctx context.Context, dispatcher *Dispatcher, query T) (R, error) {
	var zero R
	result, err := dispatchTyped(ctx, dispatcher, query, "query")
	if err != nil {
		return zero, err
	}
	if result == nil {
		return zero, nil
	}
	typed, ok := result.(R)
	if !ok {
		return zero, core.NewDomainError(core.ErrorCodeInternal, "query handler returned unexpected type")
	}
	return typed, nil
}

func dispatchTyped(ctx context.Context, dispatcher *Dispatcher, request any, _ string) (any, error) {
	return dispatcher.Dispatch(ctx, request)
}
