package behaviors

import (
	"context"

	"github.com/fastygo/framework/pkg/core"
	"github.com/fastygo/framework/pkg/core/cqrs"
)

type Validation struct{}

type validator interface {
	Validate() error
}

func (Validation) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error) {
	if request == nil {
		return next(ctx, request)
	}

	if validator, ok := request.(validator); ok {
		if err := validator.Validate(); err != nil {
			return nil, core.WrapDomainError(core.ErrorCodeValidation, "validation failed", err)
		}
	}

	return next(ctx, request)
}
