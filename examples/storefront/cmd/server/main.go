// examples/storefront is a static commerce prototype built from Framework,
// Blocks, Elements, and UI8Kit.
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
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/security"

	homefeature "github.com/fastygo/framework/examples/storefront/internal/site/home"
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

	application := app.New(cfg).
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
		WithFeature(homefeature.New()).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
