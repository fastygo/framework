// examples/blog renders embedded markdown posts behind a UI8Kit shell.
//
// Routes:
//
//	/                — list of posts
//	/posts/{slug}    — rendered article
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
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/security"

	blogcontent "github.com/fastygo/framework/examples/blog/content"
	blogfeature "github.com/fastygo/framework/examples/blog/internal/site/blog"
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

	library, err := content.NewLibrary(content.LibraryOptions{
		FS:            blogcontent.FS,
		Pages:         blogfeature.PageMetas(),
		Locales:       cfg.AvailableLocales,
		DefaultLocale: cfg.DefaultLocale,
		PathTemplate:  "i18n/{locale}/{slug}.md",
	})
	if err != nil {
		slog.Error("failed to load blog content", "error", err)
		os.Exit(1)
	}

	feature := blogfeature.New(library, blogfeature.Options{BrandName: "Acme Blog"})

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
		WithFeature(feature).
		Build()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = application.Run(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
