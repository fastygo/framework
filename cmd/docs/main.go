package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fastygo/framework/internal/application/docs"
	docsfeature "github.com/fastygo/framework/internal/infra/features/docs"
	docscontent "github.com/fastygo/framework/internal/site/docs/content"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/core/cqrs/behaviors"
	"github.com/fastygo/framework/pkg/web/security"
)

func main() {
	if _, ok := os.LookupEnv("APP_BIND"); !ok {
		os.Setenv("APP_BIND", "127.0.0.1:8081")
	}
	if _, ok := os.LookupEnv("APP_STATIC_DIR"); !ok {
		os.Setenv("APP_STATIC_DIR", "internal/site/docs/web/static")
	}

	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	dispatcher := cqrs.NewDispatcher(
		&behaviors.Logging{Logger: slog.Default()},
		&behaviors.Validation{},
	)

	pagesHandler, err := docs.NewDocsPageQueryHandler(docscontent.FS, cfg.AvailableLocales, cfg.DefaultLocale)
	if err != nil {
		slog.Error("failed to initialize docs pages", "error", err)
		os.Exit(1)
	}

	cqrs.RegisterQuery(dispatcher, docs.DocsListQueryHandler{})
	cqrs.RegisterQuery(dispatcher, pagesHandler)

	application := app.New(cfg).
		WithSecurity(security.DefaultConfig()).
		WithFeature(docsfeature.New(dispatcher, cfg.DefaultLocale, cfg.AvailableLocales)).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
