package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/fastygo/framework/pkg/observability"
	"github.com/fastygo/framework/pkg/web/locale"
	"github.com/fastygo/framework/pkg/web/health"
	"github.com/fastygo/framework/pkg/web/metrics"
	webmiddleware "github.com/fastygo/framework/pkg/web/middleware"
	"github.com/fastygo/framework/pkg/web/security"
)

// App is a fully assembled application: configuration, HTTP handler chain,
// background workers, and registered features.
type App struct {
	cfg      Config
	mux      *http.ServeMux
	features []Feature
	handler  http.Handler
	navItems []NavItem
	workers  *WorkerService
}

// ServeHTTP makes App usable as an http.Handler in tests or embedded scenarios.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Handler().ServeHTTP(w, r)
}

// Handler returns the fully chained http.Handler (security middleware + mux).
func (a *App) Handler() http.Handler {
	return a.handler
}

// NavItems returns a copy of the navigation list aggregated from all features.
func (a *App) NavItems() []NavItem {
	return append([]NavItem{}, a.navItems...)
}

// Config exposes the resolved application configuration to consumers
// (mainly useful for tests or for composition roots that need access
// to runtime values after building the App).
func (a *App) Config() Config {
	return a.cfg
}

// Workers returns the underlying WorkerService so that callers can register
// extra background tasks before the App starts.
func (a *App) Workers() *WorkerService {
	return a.workers
}

// AppBuilder is a fluent constructor for App. It accumulates features,
// security configuration, and worker tasks then materialises a final App
// in Build().
type AppBuilder struct {
	cfg      Config
	secCfg   security.Config
	features []Feature
	mux      *http.ServeMux
	workers  *WorkerService
	logger   *slog.Logger
	locale     locale.LocaleStrategy
	localeSPAMode bool

	staticPrefix string
	staticRoutes bool

	healthLivePath  string
	healthReadyPath string
	healthExtra     []health.Checker

	metricsRegistry *metrics.Registry
	metricsPath     string

	tracer observability.Tracer
}

// New constructs a new AppBuilder with default security configuration and
// the standard `/static/` static-asset route enabled.
//
// If cfg was assembled by hand (rather than via LoadConfig) any zero-value
// HTTP timeout fields are backfilled with the same defaults LoadConfig
// would have produced, so the resulting http.Server is always protected
// against Slowloris and slow-write attacks.
func New(cfg Config) *AppBuilder {
	return &AppBuilder{
		cfg:          applyHTTPDefaults(cfg),
		secCfg:       security.DefaultConfig(),
		mux:          http.NewServeMux(),
		workers:      &WorkerService{},
		logger:       slog.Default(),
		staticPrefix: "/static/",
		staticRoutes: true,
	}
}

// applyHTTPDefaults fills in zero-value HTTP timeouts so that callers who
// build Config literally (without LoadConfig) still get safe defaults.
func applyHTTPDefaults(cfg Config) Config {
	if cfg.HTTPReadTimeout <= 0 {
		cfg.HTTPReadTimeout = defaultHTTPReadTimeout
	}
	if cfg.HTTPReadHeaderTimeout <= 0 {
		cfg.HTTPReadHeaderTimeout = defaultHTTPReadHeaderTimeout
	}
	if cfg.HTTPWriteTimeout <= 0 {
		cfg.HTTPWriteTimeout = defaultHTTPWriteTimeout
	}
	if cfg.HTTPIdleTimeout <= 0 {
		cfg.HTTPIdleTimeout = defaultHTTPIdleTimeout
	}
	if cfg.HTTPMaxHeaderBytes <= 0 {
		cfg.HTTPMaxHeaderBytes = defaultHTTPMaxHeaderBytes
	}
	if cfg.HTTPShutdownTimeout <= 0 {
		cfg.HTTPShutdownTimeout = defaultHTTPShutdownTimeout
	}
	return cfg
}

// WithFeature registers a feature module. Order matters when features
// compete for the same route — the first one wins.
func (b *AppBuilder) WithFeature(feature Feature) *AppBuilder {
	if feature == nil {
		return b
	}
	b.features = append(b.features, feature)
	return b
}

// WithSecurity overrides the default security configuration.
func (b *AppBuilder) WithSecurity(cfg security.Config) *AppBuilder {
	b.secCfg = cfg
	return b
}

// WithLogger replaces the default slog.Default() logger used for app-level
// events (start, listen, worker lifecycle).
func (b *AppBuilder) WithLogger(logger *slog.Logger) *AppBuilder {
	if logger != nil {
		b.logger = logger
	}
	return b
}

// WithStaticPrefix changes the URL prefix used for serving static assets.
// Pass an empty string to disable the built-in static route entirely.
func (b *AppBuilder) WithStaticPrefix(prefix string) *AppBuilder {
	if prefix == "" {
		b.staticRoutes = false
		return b
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	b.staticPrefix = prefix
	b.staticRoutes = true
	return b
}

// DisableStatic prevents the builder from registering the default static
// route. Useful for API-only deployments.
func (b *AppBuilder) DisableStatic() *AppBuilder {
	b.staticRoutes = false
	return b
}

// AddBackgroundTask appends a background task to be supervised at runtime.
func (b *AppBuilder) AddBackgroundTask(task BackgroundTask) *AppBuilder {
	b.workers.Add(task)
	return b
}

// HTTPServerOptions overrides the http.Server timeouts that App.Run reads
// from Config. Only non-zero fields are applied so callers may tweak a
// single value without restating the rest:
//
//	builder.WithHTTPServerOptions(app.HTTPServerOptions{
//	    ReadHeaderTimeout: 1 * time.Second, // tighter Slowloris guard
//	})
//
// Zero fields fall through to the values resolved from APP_HTTP_*
// environment variables (or built-in defaults).
type HTTPServerOptions struct {
	// ReadTimeout caps the total time spent reading the request,
	// including body. See net/http.Server.ReadTimeout.
	ReadTimeout time.Duration
	// ReadHeaderTimeout caps the time to read request headers and is
	// the primary Slowloris defence; keep it short.
	ReadHeaderTimeout time.Duration
	// WriteTimeout caps the time spent writing the response.
	WriteTimeout time.Duration
	// IdleTimeout caps how long a keep-alive connection may sit idle.
	IdleTimeout time.Duration
	// ShutdownTimeout bounds graceful shutdown (Server.Shutdown plus
	// WorkerService.Stop) on context cancellation.
	ShutdownTimeout time.Duration
	// MaxHeaderBytes bounds the total size of request headers.
	MaxHeaderBytes int
}

// WithHTTPServerOptions overrides one or more HTTP server timeouts on top
// of the values loaded from Config. See HTTPServerOptions.
func (b *AppBuilder) WithHTTPServerOptions(opts HTTPServerOptions) *AppBuilder {
	if opts.ReadTimeout > 0 {
		b.cfg.HTTPReadTimeout = opts.ReadTimeout
	}
	if opts.ReadHeaderTimeout > 0 {
		b.cfg.HTTPReadHeaderTimeout = opts.ReadHeaderTimeout
	}
	if opts.WriteTimeout > 0 {
		b.cfg.HTTPWriteTimeout = opts.WriteTimeout
	}
	if opts.IdleTimeout > 0 {
		b.cfg.HTTPIdleTimeout = opts.IdleTimeout
	}
	if opts.ShutdownTimeout > 0 {
		b.cfg.HTTPShutdownTimeout = opts.ShutdownTimeout
	}
	if opts.MaxHeaderBytes > 0 {
		b.cfg.HTTPMaxHeaderBytes = opts.MaxHeaderBytes
	}
	return b
}

// WithHealthEndpoints publishes liveness and readiness probes.
//
// The liveness path always returns 200 once the binary is serving traffic
// (it does not run any checks). The readiness path runs every Feature
// that implements HealthChecker plus any explicit health.Checker added via
// AddHealthChecker, in parallel, with a per-check timeout. Pass an empty
// string to disable that endpoint.
//
// Both endpoints are registered before the security middleware chain so
// they are not subject to rate limiting, anti-bot challenges, or body
// limits. They still get request-id, recover, and structured logging.
func (b *AppBuilder) WithHealthEndpoints(livePath, readyPath string) *AppBuilder {
	b.healthLivePath = livePath
	b.healthReadyPath = readyPath
	return b
}

// AddHealthChecker registers an explicit health.Checker that is not tied
// to a Feature. Useful for infrastructure probes (database ping, broker
// connectivity) wired in the composition root.
func (b *AppBuilder) AddHealthChecker(c health.Checker) *AppBuilder {
	if c != nil {
		b.healthExtra = append(b.healthExtra, c)
	}
	return b
}

// WithTracer installs an observability.Tracer. When set (and not the
// no-op), the builder inserts TracingMiddleware at the outer edge of
// the request pipeline so every request opens a span and downstream
// LoggerMiddleware can decorate log lines with trace_id/span_id.
//
// Pass observability.NoopTracer{} explicitly only when you want to
// document the choice; otherwise simply skip the call — the builder
// treats nil as no-op.
func (b *AppBuilder) WithTracer(t observability.Tracer) *AppBuilder {
	b.tracer = t
	return b
}

// WithMetricsRegistry installs a metrics.Registry. When set, the builder
// wraps the request pipeline with MetricsMiddleware so http_requests_*
// metrics are emitted automatically. Pass a custom Registry when the
// composition root needs to register additional application-level
// metrics; pass nil (or skip the call) to disable metrics entirely.
//
// Calling WithMetricsRegistry without WithMetricsEndpoint is valid: the
// registry is still populated, just not exposed over HTTP. The
// composition root can then mount the metrics handler on a private
// listener for security reasons.
func (b *AppBuilder) WithMetricsRegistry(r *metrics.Registry) *AppBuilder {
	b.metricsRegistry = r
	return b
}

// WithMetricsEndpoint publishes the Prometheus text exposition handler
// at path. If WithMetricsRegistry was not called, a fresh Registry is
// created automatically. Like the health probes, the metrics endpoint
// is registered before the security middleware chain so scrapes are
// not subject to rate limiting or anti-bot challenges.
func (b *AppBuilder) WithMetricsEndpoint(path string) *AppBuilder {
	b.metricsPath = path
	if b.metricsRegistry == nil {
		b.metricsRegistry = metrics.NewRegistry()
	}
	return b
}

// Mux gives access to the underlying ServeMux for advanced composition.
// Most applications should rely on Feature.Routes(...) instead.
func (b *AppBuilder) Mux() *http.ServeMux {
	return b.mux
}

// Build wires everything up and returns a runnable App.
func (b *AppBuilder) Build() *App {
	navItems := collectNavItems(b.features)

	if b.staticRoutes && b.cfg.StaticDir != "" {
		b.mux.Handle(b.staticPrefix, http.StripPrefix(b.staticPrefix,
			security.SecureFileServer(b.cfg.StaticDir, 86400),
		))
	}

	for _, feature := range b.features {
		if navAware, ok := any(feature).(NavProvider); ok {
			navAware.SetNavItems(navItems)
		}
		feature.Routes(b.mux)

		if provider, ok := any(feature).(BackgroundProvider); ok {
			for _, task := range provider.BackgroundTasks() {
				b.workers.Add(task)
			}
		}
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
	if b.metricsRegistry != nil {
		// Metrics sit at the outer edge of the security chain so they
		// observe every request that survived to the handler, including
		// those still rejected by inner middleware (status codes will
		// reflect the ultimate response).
		chain = append(webmiddleware.Chain{webmiddleware.MetricsMiddleware(b.metricsRegistry)}, chain...)
	}
	if b.tracer != nil {
		// Tracing wraps everything else so the SpanContext (and the
		// derived Correlation) is visible to metrics, security, and
		// logger middleware below it. The TracingMiddleware itself
		// short-circuits to a passthrough when given a NoopTracer.
		chain = append(webmiddleware.Chain{webmiddleware.TracingMiddleware(b.tracer)}, chain...)
	}
	handler := chain.Then(b.mux)

	if b.healthLivePath != "" || b.healthReadyPath != "" {
		handler = wrapWithHealth(
			handler,
			b.healthLivePath,
			b.healthReadyPath,
			collectHealthCheckers(b.features, b.healthExtra),
		)
	}

	if b.metricsPath != "" && b.metricsRegistry != nil {
		handler = wrapWithMetrics(handler, b.metricsPath, b.metricsRegistry)
	}
	if b.locale != nil {
		handler = locale.MiddlewareWithSPAMode(b.locale, b.localeSPAMode)(handler)
	}

	return &App{
		cfg:      b.cfg,
		mux:      b.mux,
		features: append([]Feature{}, b.features...),
		handler:  handler,
		navItems: navItems,
		workers:  b.workers,
	}
}

// LocaleStrategy returns the locale strategy installed by WithLocales.
func (b *AppBuilder) LocaleStrategy() locale.LocaleStrategy {
	return b.locale
}

// LocaleSPAMode reports whether locale middleware was configured for SPA
// enhancement support in client-side language switching.
func (b *AppBuilder) LocaleSPAMode() bool {
	return b.localeSPAMode
}

// wrapWithHealth returns an http.Handler that serves liveness and
// readiness probes directly (with only request-id + recover + logger),
// bypassing the full security chain. Anything that does not match a
// probe path falls through to the inner handler.
func wrapWithHealth(inner http.Handler, livePath, readyPath string, checkers []health.Checker) http.Handler {
	agg := health.NewAggregator()
	for _, c := range checkers {
		agg.Add(c)
	}

	probeChain := webmiddleware.Chain{
		webmiddleware.RequestIDMiddleware(),
		webmiddleware.RecoverMiddleware(),
		webmiddleware.LoggerMiddleware(),
	}
	liveHandler := probeChain.Then(health.LiveHandler())
	readyHandler := probeChain.Then(health.ReadyHandler(agg))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case livePath:
			if livePath != "" {
				liveHandler.ServeHTTP(w, r)
				return
			}
		case readyPath:
			if readyPath != "" {
				readyHandler.ServeHTTP(w, r)
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
}

// wrapWithMetrics returns an http.Handler that serves the Prometheus
// scrape endpoint directly (with only request-id + recover + logger),
// bypassing the security and metrics-recording chains. This keeps
// scrape traffic out of the http_requests_total counter so it does
// not pollute application-level RPS dashboards.
func wrapWithMetrics(inner http.Handler, path string, reg *metrics.Registry) http.Handler {
	probeChain := webmiddleware.Chain{
		webmiddleware.RequestIDMiddleware(),
		webmiddleware.RecoverMiddleware(),
		webmiddleware.LoggerMiddleware(),
	}
	metricsHandler := probeChain.Then(metrics.Handler(reg))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			metricsHandler.ServeHTTP(w, r)
			return
		}
		inner.ServeHTTP(w, r)
	})
}

// collectHealthCheckers walks the registered features, picks the ones
// implementing HealthChecker, wraps them as health.Checker, and appends
// any explicit checkers added via AddHealthChecker.
func collectHealthCheckers(features []Feature, extra []health.Checker) []health.Checker {
	out := make([]health.Checker, 0, len(features)+len(extra))
	for _, f := range features {
		hc, ok := any(f).(HealthChecker)
		if !ok {
			continue
		}
		fid := f.ID()
		out = append(out, health.CheckerFunc{
			CheckerName: fid,
			Fn:          hc.HealthCheck,
		})
	}
	out = append(out, extra...)
	return out
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

// Run starts the HTTP server and blocks until ctx is cancelled or the
// server exits with an error.
//
// Lifecycle:
//   - Init() is called sequentially on every feature that implements Initializer.
//     If any returns an error the server is not started.
//   - Background workers are launched.
//   - HTTP server starts listening.
//   - On ctx.Done() the server is gracefully shut down and Close() is invoked
//     on every feature that implements Closer.
func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := slog.Default()

	for _, feature := range a.features {
		if init, ok := any(feature).(Initializer); ok {
			if err := init.Init(runCtx); err != nil {
				return fmt.Errorf("feature %q init: %w", feature.ID(), err)
			}
		}
	}

	if a.workers != nil {
		a.workers.Start(runCtx)
	}

	server := &http.Server{
		Addr:              a.cfg.AppBind,
		Handler:           a.Handler(),
		ReadTimeout:       a.cfg.HTTPReadTimeout,
		ReadHeaderTimeout: a.cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      a.cfg.HTTPWriteTimeout,
		IdleTimeout:       a.cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    a.cfg.HTTPMaxHeaderBytes,
	}

	shutdownTimeout := a.cfg.HTTPShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultHTTPShutdownTimeout
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("app:listen", "addr", a.cfg.AppBind)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancelShutdown()
		shutdownErr := server.Shutdown(shutdownCtx)
		if a.workers != nil {
			if err := a.workers.Stop(shutdownCtx); err != nil {
				logger.Warn("workers stop", "error", err)
			}
		}
		a.closeFeatures(shutdownCtx, logger)
		return shutdownErr
	case err := <-errCh:
		cancel()
		if a.workers != nil {
			stopCtx, cancelStop := context.WithTimeout(context.Background(), shutdownTimeout)
			if stopErr := a.workers.Stop(stopCtx); stopErr != nil {
				logger.Warn("workers stop", "error", stopErr)
			}
			cancelStop()
		}
		a.closeFeatures(context.Background(), logger)
		return err
	}
}

func (a *App) closeFeatures(ctx context.Context, logger *slog.Logger) {
	for i := len(a.features) - 1; i >= 0; i-- {
		feature := a.features[i]
		closer, ok := any(feature).(Closer)
		if !ok {
			continue
		}
		if err := closer.Close(ctx); err != nil {
			logger.Warn("feature close error", "feature", feature.ID(), "error", err)
		}
	}
}
