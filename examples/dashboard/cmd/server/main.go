// examples/dashboard is a sidebar-shell starter wired with a contacts CRUD,
// an HMAC-signed cookie session, and a tiny domain layer.
//
// Use it as a baseline for internal tools, CRMs, admin panels, or anywhere
// you need an authenticated dashboard with multiple feature modules.
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

	authfeature "github.com/fastygo/framework/examples/dashboard/internal/site/auth"
	"github.com/fastygo/framework/examples/dashboard/internal/site/contacts"
	dashboardfeature "github.com/fastygo/framework/examples/dashboard/internal/site/dashboard"
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

	sessionKey := cfg.SessionKey
	if sessionKey == "" {
		// Demo default. Always set SESSION_KEY in production.
		sessionKey = "demo-session-key-change-me"
	}

	repo := contacts.NewRepository()
	repo.Seed()

	auth := authfeature.New(sessionKey)
	dashboard := dashboardfeature.New(auth, repo, "Acme Dashboard")

	application := app.New(cfg).
		WithSecurity(security.LoadConfig()).
		WithLocales(app.LocalesConfig{
			Strategy: &locale.PathPrefixStrategy{
				Available: cfg.AvailableLocales,
				Default:   cfg.DefaultLocale,
			},
			Cookie: locale.CookieOptions{
				Enabled: true,
				Name:    "lang",
			},
		}).
		WithFeature(auth).
		WithFeature(dashboard).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
