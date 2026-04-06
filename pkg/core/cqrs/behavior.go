package cqrs

import (
	"context"
)

type HandlerFunc func(context.Context, any) (any, error)

type PipelineBehavior interface {
	Handle(ctx context.Context, request any, next HandlerFunc) (any, error)
}

type HandlerNotFoundError struct {
	RequestType string
}

func (e HandlerNotFoundError) Error() string {
	return "no handler registered for request: " + e.RequestType
}
