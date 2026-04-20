package cqrs

import (
	"context"

	"github.com/fastygo/framework/pkg/core"
)

// Dispatcher routes a request to the registered command or query
// handler and runs every PipelineBehavior around the call. It is
// not safe to register handlers concurrently with Dispatch — wire
// it up at startup, freeze it, then share across goroutines.
type Dispatcher struct {
	commandHandlers map[string]HandlerFunc
	queryHandlers   map[string]HandlerFunc
	behaviors       []PipelineBehavior
}

// NewDispatcher returns a Dispatcher with the supplied pipeline
// behaviors applied in order: behaviors[0] is the outermost wrapper
// (sees the request first, the response last).
func NewDispatcher(behaviors ...PipelineBehavior) *Dispatcher {
	return &Dispatcher{
		commandHandlers: make(map[string]HandlerFunc),
		queryHandlers:   make(map[string]HandlerFunc),
		behaviors:       behaviors,
	}
}

// RegisterCommandHandler associates a low-level HandlerFunc with the
// given request type key. Prefer the typed RegisterCommand helper
// over calling this directly.
func (d *Dispatcher) RegisterCommandHandler(requestType string, handler HandlerFunc) {
	d.commandHandlers[requestType] = handler
}

// RegisterQueryHandler is the read-side counterpart to
// RegisterCommandHandler.
func (d *Dispatcher) RegisterQueryHandler(requestType string, handler HandlerFunc) {
	d.queryHandlers[requestType] = handler
}

// Dispatch resolves the handler for request and invokes it through
// the pipeline. Commands are looked up first; queries are tried only
// if no command handler matches. Returns ErrorCodeNotFound wrapped in
// a DomainError when no handler is registered.
func (d *Dispatcher) Dispatch(ctx context.Context, request any) (any, error) {
	requestType := requestKey(request)

	if handler, ok := d.commandHandlers[requestType]; ok {
		return d.execute(ctx, requestType, handler, request)
	}

	if handler, ok := d.queryHandlers[requestType]; ok {
		return d.execute(ctx, requestType, handler, request)
	}

	return nil, core.WrapDomainError(core.ErrorCodeNotFound, "request handler missing", HandlerNotFoundError{RequestType: requestType})
}

func (d *Dispatcher) execute(ctx context.Context, _ string, handler HandlerFunc, request any) (any, error) {
	next := func(c context.Context, r any) (any, error) {
		return handler(c, r)
	}

	for i := len(d.behaviors) - 1; i >= 0; i-- {
		next = wrapBehavior(d.behaviors[i], next)
	}

	return next(ctx, request)
}

func wrapBehavior(behavior PipelineBehavior, next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, req any) (any, error) {
		return behavior.Handle(ctx, req, next)
	}
}
