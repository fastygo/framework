package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fastygo/framework/internal/features/welcome"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/core/cqrs/behaviors"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	dispatcher := cqrs.NewDispatcher(
		&behaviors.Logging{Logger: slog.Default()},
		&behaviors.Validation{},
	)
	cqrs.RegisterQuery(dispatcher, welcome.WelcomeQueryHandler{})

	application := app.New(cfg).
		WithFeature(welcome.New(dispatcher, cfg.DefaultLocale, cfg.AvailableLocales)).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
