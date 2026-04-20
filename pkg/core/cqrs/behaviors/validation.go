package behaviors

import (
	"context"

	"github.com/fastygo/framework/pkg/core"
	"github.com/fastygo/framework/pkg/core/cqrs"
)

// Validation is a PipelineBehavior that calls Validate() on the
// request when the request implements a `Validate() error` method.
// A non-nil error is wrapped in a DomainError(ErrorCodeValidation).
// Requests without a Validate method pass through unchanged.
type Validation struct{}

type validator interface {
	Validate() error
}

// Handle implements cqrs.PipelineBehavior.
func (Validation) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error) {
	if request == nil {
		return next(ctx, request)
	}

	if v, ok := request.(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, core.WrapDomainError(core.ErrorCodeValidation, "validation failed", err)
		}
	}

	return next(ctx, request)
}
