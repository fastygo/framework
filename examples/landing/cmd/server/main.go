// examples/landing is the smallest possible feature/app composition: one
// page, one feature, one composition root. Use it as a starter for static
// marketing-style sites where you do not need i18n or CQRS yet.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/cache"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/framework/pkg/web/view"

	landingi18n "github.com/fastygo/framework/examples/landing/internal/site/i18n"
	"github.com/fastygo/framework/examples/landing/internal/site/views"
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

	feature := newLandingFeature()

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

type landingFeature struct {
	htmlCache *cache.Cache[[]byte]
}

func newLandingFeature() *landingFeature {
	return &landingFeature{
		htmlCache: cache.New[[]byte](10 * 60),
	}
}

func (f *landingFeature) ID() string              { return "landing" }
func (f *landingFeature) NavItems() []app.NavItem { return nil }
func (f *landingFeature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/", f.handle)
}

func (f *landingFeature) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	bundle, err := landingi18n.Load(locale.From(r.Context()))
	if err != nil {
		bundle = landingi18n.Bundle{}
	}
	loc := locale.From(r.Context())
	page := buildLandingPage(r.Context(), loc, bundle)

	if err := web.CachedRender(
		r.Context(),
		w,
		r,
		f.htmlCache,
		"landing:"+locale.From(r.Context()),
		views.Landing(page),
	); err != nil {
		web.HandleError(w, err)
	}
}

func buildLandingPage(ctx context.Context, loc string, bundle landingi18n.Bundle) views.Page {
	landing := bundle.Landing
	if landing.BrandName == "" {
		landing.BrandName = "Acme"
	}
	if landing.Tagline == "" {
		landing.Tagline = "Built with fastygo + ui8kit"
	}
	if landing.Title == "" {
		landing.Title = "Launch fast, stay simple."
	}
	if landing.Subtitle == "" {
		landing.Subtitle = "A one-file landing example showing the absolute minimum needed to ship a server-rendered marketing page."
	}
	if landing.PrimaryCTA == "" {
		landing.PrimaryCTA = "Get the framework"
	}
	if landing.PrimaryHref == "" {
		landing.PrimaryHref = "https://github.com/fastygo/framework"
	}
	if landing.FooterText == "" {
		landing.FooterText = "© Acme — Powered by fastygo/framework"
	}
	language := view.BuildLanguageToggleFromContext(ctx,
		view.WithLabel(landing.Language.Label),
		view.WithCurrentLabel(landing.Language.CurrentLabel),
		view.WithNextLocale(landing.Language.NextLocale),
		view.WithNextLabel(landing.Language.NextLabel),
		view.WithLocaleLabels(landing.Language.LocaleLabels),
	)

	page := views.Page{
		Lang:                     loc,
		BrandName:                landing.BrandName,
		Tagline:                  landing.Tagline,
		Title:                    landing.Title,
		Subtitle:                 landing.Subtitle,
		PrimaryCTA:               landing.PrimaryCTA,
		PrimaryHref:              landing.PrimaryHref,
		FooterText:               landing.FooterText,
		ThemeLabel:               fallbackText(landing.Theme.Label, "Theme"),
		ThemeDarkLabel:           fallbackText(landing.Theme.SwitchToDarkLabel, "Switch to dark mode"),
		ThemeLightLabel:          fallbackText(landing.Theme.SwitchToLightLabel, "Switch to light mode"),
		LanguageCurrentLocale:    fallbackText(language.CurrentLocale, ""),
		LanguageCurrentLabel:     fallbackText(language.CurrentLabel, strings.ToUpper(language.CurrentLocale)),
		LanguageNextLocale:       fallbackText(language.NextLocale, ""),
		LanguageNextLabel:        fallbackText(language.NextLabel, strings.ToUpper(language.NextLocale)),
		LanguageDefaultLocale:    fallbackText(language.DefaultLocale, "en"),
		LanguageAvailableLocales: strings.Join(language.AvailableLocales, ","),
		LanguageNextHref:         language.NextHref,
	}

	for _, feature := range landing.Features {
		page.Features = append(page.Features, views.FeatureItem{
			Title:       feature.Title,
			Description: feature.Description,
			Icon:        feature.Icon,
		})
	}

	return page
}

func fallbackText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
