// Package cqrs provides a tiny, opinionated Command/Query/Result
// dispatcher. It is intentionally minimal: there is no event bus,
// no saga, no inbox/outbox — just a typed pipeline with optional
// behaviors (logging, validation, ...).
//
// Use it when you want to keep handler code free of cross-cutting
// concerns; skip it entirely if your features call domain services
// directly. The framework does not require it.
package cqrs

import "context"

// Command is the marker type for write-side requests. Concrete
// commands are user-defined structs; the marker exists for
// documentation only.
type Command interface{}

// Query is the marker type for read-side requests, mirroring Command.
type Query interface{}

// CommandHandler handles a single write-side request type T and
// returns a result R. Implementations must be safe for concurrent
// use (the dispatcher fans calls in from any goroutine).
type CommandHandler[T any, R any] interface {
	Handle(ctx context.Context, cmd T) (R, error)
}

// QueryHandler handles a single read-side request type T and
// returns a result R. Same concurrency contract as CommandHandler.
type QueryHandler[T any, R any] interface {
	Handle(ctx context.Context, query T) (R, error)
}
