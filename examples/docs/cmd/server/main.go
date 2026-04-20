// examples/docs is a documentation site rendered from embedded markdown
// using fastygo/framework + ui8kit.
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
	content "github.com/fastygo/framework/pkg/content-markdown"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/framework/pkg/web/locale"

	docscontent "github.com/fastygo/framework/examples/docs/content"
	docsfeature "github.com/fastygo/framework/examples/docs/internal/site/docs"
)

func main() {
	if _, ok := os.LookupEnv("APP_BIND"); !ok {
		os.Setenv("APP_BIND", "127.0.0.1:8081")
	}
	if _, ok := os.LookupEnv("APP_STATIC_DIR"); !ok {
		os.Setenv("APP_STATIC_DIR", "web/static")
	}

	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	library, err := content.NewLibrary(content.LibraryOptions{
		FS:            docscontent.FS,
		Pages:         docsfeature.Pages,
		Locales:       cfg.AvailableLocales,
		DefaultLocale: cfg.DefaultLocale,
	})
	if err != nil {
		slog.Error("failed to build docs library", "error", err)
		os.Exit(1)
	}

	application := app.New(cfg).
		WithSecurity(security.LoadConfig()).
		WithLocales(app.LocalesConfig{
			Strategy: &locale.QueryStrategy{
				Aliases:    []string{"translate"},
				ValueMap:   map[string]string{"english": "en", "russian": "ru"},
				Available:  cfg.AvailableLocales,
				Default:    cfg.DefaultLocale,
			},
			Cookie: locale.CookieOptions{Enabled: true, Name: "lang"},
		}).
		WithFeature(docsfeature.New(library)).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
