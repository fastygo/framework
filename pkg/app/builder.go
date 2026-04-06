package app

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	webmiddleware "github.com/fastygo/framework/pkg/web/middleware"
)

type App struct {
	cfg      Config
	mux      *http.ServeMux
	features []Feature
	handler  http.Handler
	navItems []NavItem
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Handler().ServeHTTP(w, r)
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) NavItems() []NavItem {
	return append([]NavItem{}, a.navItems...)
}

type NavProvider interface {
	SetNavItems([]NavItem)
}

type AppBuilder struct {
	cfg      Config
	features []Feature
	mux      *http.ServeMux
}

func New(cfg Config) *AppBuilder {
	return &AppBuilder{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
}

func (b *AppBuilder) WithFeature(feature Feature) *AppBuilder {
	b.features = append(b.features, feature)
	return b
}

func (b *AppBuilder) Build() *App {
	navItems := collectNavItems(b.features)
	middleware := webmiddleware.Chain{
		webmiddleware.RequestIDMiddleware(),
		webmiddleware.RecoverMiddleware(),
		webmiddleware.LoggerMiddleware(),
	}

	for _, feature := range b.features {
		if navAware, ok := any(feature).(NavProvider); ok {
			navAware.SetNavItems(navItems)
		}
		feature.Routes(b.mux)
	}

	var handler http.Handler = middleware.Then(b.mux)

	b.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(b.cfg.StaticDir))))

	return &App{
		cfg:      b.cfg,
		mux:      b.mux,
		features: append([]Feature{}, b.features...),
		handler:  handler,
		navItems: navItems,
	}
}

func collectNavItems(features []Feature) []NavItem {
	var items []NavItem
	for _, feature := range features {
		items = append(items, feature.NavItems()...)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Order == items[j].Order {
			return items[i].Label < items[j].Label
		}
		return items[i].Order < items[j].Order
	})
	return items
}

func (a *App) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:         a.cfg.AppBind,
		Handler:      a.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("app:listen", "addr", a.cfg.AppBind)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
