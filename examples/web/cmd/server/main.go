// examples/web is a small marketing-style site built on top of the
// fastygo framework and UI8Kit. It demonstrates how to wire a feature
// module that owns its own templ views, i18n bundle, and static assets.
//
// Run locally:
//
//	cd examples/web
//	bun install
//	go mod download
//	bun run vendor:assets
//	bun run build:css
//	templ generate ./...
//	go run ./cmd/server
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/core/cqrs"
	"github.com/fastygo/framework/pkg/core/cqrs/behaviors"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/security"

	cabfeature "github.com/fastygo/framework/examples/web/internal/site/cab"
	welcomefeature "github.com/fastygo/framework/examples/web/internal/site/welcome"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if cfg.StaticDir == "internal/site/web/static" {
		cfg.StaticDir = "web/static"
	}

	dispatcher := cqrs.NewDispatcher(
		&behaviors.Logging{Logger: slog.Default()},
		&behaviors.Validation{},
	)
	cqrs.RegisterQuery(dispatcher, welcomefeature.NewQueryHandler())

	builder := app.New(cfg).
		WithSecurity(security.LoadConfig()).
		WithLocales(app.LocalesConfig{
			Strategy: &locale.PathPrefixStrategy{
				Available:       cfg.AvailableLocales,
				Default:         cfg.DefaultLocale,
				RedirectMissing: true,
			},
			Cookie: locale.CookieOptions{
				Enabled: true,
				Name:    "lang",
			},
			SPA: true,
		}).
		WithFeature(welcomefeature.New(dispatcher))

	if cfg.OIDCIssuer != "" && cfg.OIDCClientID != "" {
		builder = builder.WithFeature(cabfeature.New(cfg))
		slog.Info("OIDC cab feature enabled", "issuer", cfg.OIDCIssuer, "client_id", cfg.OIDCClientID)
	}

	application := builder.Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
