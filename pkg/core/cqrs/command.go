package cqrs

import "context"

type Command interface{}
type Query interface{}

type CommandHandler[T any, R any] interface {
	Handle(ctx context.Context, cmd T) (R, error)
}

type QueryHandler[T any, R any] interface {
	Handle(ctx context.Context, query T) (R, error)
}
