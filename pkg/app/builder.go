package app

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/fastygo/framework/pkg/web/security"
	webmiddleware "github.com/fastygo/framework/pkg/web/middleware"
)

type App struct {
	cfg      Config
	mux      *http.ServeMux
	features []Feature
	handler  http.Handler
	navItems []NavItem
	workers  *WorkerService
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
	secCfg   security.Config
	features []Feature
	mux      *http.ServeMux
	workers  *WorkerService
}

func New(cfg Config) *AppBuilder {
	return &AppBuilder{
		cfg:     cfg,
		secCfg:  security.DefaultConfig(),
		mux:     http.NewServeMux(),
		workers: &WorkerService{},
	}
}

func (b *AppBuilder) WithFeature(feature Feature) *AppBuilder {
	b.features = append(b.features, feature)
	return b
}

func (b *AppBuilder) WithSecurity(cfg security.Config) *AppBuilder {
	b.secCfg = cfg
	return b
}

func (b *AppBuilder) Build() *App {
	navItems := collectNavItems(b.features)

	b.mux.Handle("/static/", http.StripPrefix("/static/",
		security.SecureFileServer(b.cfg.StaticDir, 86400),
	))

	for _, feature := range b.features {
		if navAware, ok := any(feature).(NavProvider); ok {
			navAware.SetNavItems(navItems)
		}
		feature.Routes(b.mux)
	}

	var rateLimiter *security.RateLimiter
	if b.secCfg.Enabled {
		rateLimiter = security.NewRateLimiter(b.secCfg.PageRateLimit, b.secCfg.PageRateBurst)
		b.workers.Add(BackgroundTask{
			Name:     "ratelimit-cleanup",
			Interval: 1 * time.Minute,
			Run:      func(ctx context.Context) { rateLimiter.Cleanup(5 * time.Minute) },
		})
	}

	chain := createSecurityChain(b.secCfg, rateLimiter)
	handler := chain.Then(b.mux)

	return &App{
		cfg:      b.cfg,
		mux:      b.mux,
		features: append([]Feature{}, b.features...),
		handler:  handler,
		navItems: navItems,
		workers:  b.workers,
	}
}

func createSecurityChain(cfg security.Config, rateLimiter *security.RateLimiter) webmiddleware.Chain {
	if !cfg.Enabled {
		return webmiddleware.Chain{
			webmiddleware.RequestIDMiddleware(),
			webmiddleware.RecoverMiddleware(),
			webmiddleware.LoggerMiddleware(),
		}
	}

	chain := webmiddleware.Chain{
		webmiddleware.RequestIDMiddleware(),
		webmiddleware.RecoverMiddleware(),
		security.HeadersMiddleware(cfg),
		security.BodyLimitMiddleware(cfg),
		security.MethodGuardMiddleware(),
		security.AntiBotMiddleware(cfg),
	}
	if rateLimiter != nil {
		chain = append(chain, security.RateLimitMiddleware(rateLimiter, cfg.TrustProxy))
	}
	chain = append(chain, webmiddleware.LoggerMiddleware())
	return chain
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
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if a.workers != nil {
		a.workers.Start(runCtx)
	}

	server := &http.Server{
		Addr:              a.cfg.AppBind,
		Handler:           a.Handler(),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
		cancel()
		return err
	}
}
