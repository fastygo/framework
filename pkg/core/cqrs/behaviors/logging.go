package behaviors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fastygo/framework/pkg/core/cqrs"
)

// Logging is a PipelineBehavior that emits an info-level slog event
// before and after each handler invocation, including the handled
// type, duration, and error (when any). Use slog.New(...) with a
// JSON handler in production for structured ingestion.
type Logging struct {
	// Logger is the destination logger. nil falls back to slog.Default().
	Logger *slog.Logger
}

// Handle implements cqrs.PipelineBehavior.
func (l Logging) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error) {
	logger := l.Logger
	if logger == nil {
		logger = slog.Default()
	}

	requestType := fmt.Sprintf("%T", request)
	start := time.Now()
	logger.InfoContext(ctx, "cqrs:request:start", "type", requestType)

	result, err := next(ctx, request)
	elapsed := time.Since(start)

	if err != nil {
		logger.InfoContext(
			ctx,
			"cqrs:request:error",
			"type",
			requestType,
			"duration_ms",
			float64(elapsed.Milliseconds()),
			"error",
			err,
		)
		return result, err
	}

	logger.InfoContext(
		ctx,
		"cqrs:request:complete",
		"type",
		requestType,
		"duration_ms",
		float64(elapsed.Milliseconds()),
	)

	return result, nil
}
