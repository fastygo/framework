package cqrs

import (
	"context"

	"github.com/fastygo/framework/pkg/core"
)

type Dispatcher struct {
	commandHandlers map[string]HandlerFunc
	queryHandlers   map[string]HandlerFunc
	behaviors       []PipelineBehavior
}

func NewDispatcher(behaviors ...PipelineBehavior) *Dispatcher {
	return &Dispatcher{
		commandHandlers: make(map[string]HandlerFunc),
		queryHandlers:   make(map[string]HandlerFunc),
		behaviors:       behaviors,
	}
}

func (d *Dispatcher) RegisterCommandHandler(requestType string, handler HandlerFunc) {
	d.commandHandlers[requestType] = handler
}

func (d *Dispatcher) RegisterQueryHandler(requestType string, handler HandlerFunc) {
	d.queryHandlers[requestType] = handler
}

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
