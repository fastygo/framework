package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default HTTP server timeouts. Tuned for typical SSR workloads; override
// via APP_HTTP_* environment variables or AppBuilder.WithHTTPServerOptions.
const (
	defaultHTTPReadTimeout       = 5 * time.Second
	defaultHTTPReadHeaderTimeout = 2 * time.Second
	defaultHTTPWriteTimeout      = 10 * time.Second
	defaultHTTPIdleTimeout       = 120 * time.Second
	defaultHTTPMaxHeaderBytes    = 1 << 20
	defaultHTTPShutdownTimeout   = 5 * time.Second
)

// Config is the framework's 12-factor configuration surface.
//
// Every field is loaded from a single environment variable in
// LoadConfig and exposed as plain data — pkg/app does not act on
// these values itself (no slog.SetDefault, no MetricsPath listener
// installation). The composition root reads the populated Config and
// wires AppBuilder accordingly. This keeps tests trivial: build a
// Config literal, hand it in.
//
// New fields must:
//   - have a default in LoadConfig that keeps the framework usable
//     out of the box (no required env vars beyond the documented set);
//   - be listed in docs/12-FACTOR.md so operators have one source of truth;
//   - never carry secrets through logs (DomainError.StatusCode pattern).
type Config struct {
	// AppBind is the host:port the HTTP server listens on. Default 127.0.0.1:8080.
	AppBind string
	// DataSource is an opaque application-level identifier (e.g. "fixture",
	// "postgres://...") interpreted by the example apps; framework code never reads it.
	DataSource string
	// StaticDir is the on-disk root for the secure static-file server.
	StaticDir string
	// DefaultLocale is the fallback locale when no negotiation match is found.
	DefaultLocale string
	// AvailableLocales is the closed set of locales the app supports;
	// negotiation never returns anything outside this list.
	AvailableLocales []string

	// OIDCIssuer is the OpenID Connect issuer URL (matches the iss claim).
	OIDCIssuer string
	// OIDCClientID is the registered client identifier at the issuer.
	OIDCClientID string
	// OIDCClientSecret is the confidential client secret. Treat as PII.
	OIDCClientSecret string
	// OIDCRedirectURI is the absolute callback URL the issuer sends the
	// authorization code back to.
	OIDCRedirectURI string
	// SessionKey is the HMAC secret for pkg/auth.CookieSession. Treat as PII.
	// At least 32 bytes are recommended.
	SessionKey string

	// HTTPReadTimeout caps the total request read time, including body.
	HTTPReadTimeout time.Duration
	// HTTPReadHeaderTimeout caps the time to read request headers.
	// Acts as the primary Slowloris defence; keep it short.
	HTTPReadHeaderTimeout time.Duration
	// HTTPWriteTimeout caps the time spent writing the response.
	HTTPWriteTimeout time.Duration
	// HTTPIdleTimeout caps how long a keep-alive connection may sit idle.
	HTTPIdleTimeout time.Duration
	// HTTPMaxHeaderBytes bounds the total size of request headers.
	HTTPMaxHeaderBytes int
	// HTTPShutdownTimeout bounds graceful shutdown (Server.Shutdown plus
	// WorkerService.Stop) on ctx cancellation.
	HTTPShutdownTimeout time.Duration

	// LogLevel is the minimum slog level emitted by the application.
	// Accepts "debug", "info", "warn", "error" (case-insensitive). The
	// framework does not call slog.SetDefault — it only exposes the
	// parsed value so the composition root can build the handler.
	LogLevel string
	// LogFormat selects "text" or "json" for the slog handler. The same
	// caveat applies: pkg/app reads but does not install the handler.
	LogFormat string

	// HealthLivePath is the URL path served by the liveness probe
	// (typically /healthz). Empty disables the probe at the AppBuilder
	// layer.
	HealthLivePath string
	// HealthReadyPath is the URL path served by the readiness probe
	// (typically /readyz). Empty disables the probe at the AppBuilder
	// layer.
	HealthReadyPath string

	// MetricsPath enables the /metrics endpoint when non-empty. Empty
	// disables the endpoint and the metrics middleware (no registry,
	// no overhead).
	MetricsPath string

	// OTelServiceName surfaces the OTEL_SERVICE_NAME env var so the
	// composition root can pass it to a future github.com/fastygo/otel
	// adapter. The framework itself never imports go.opentelemetry.io/*.
	OTelServiceName string
	// OTelExporterEndpoint mirrors OTEL_EXPORTER_OTLP_ENDPOINT for the
	// same reason. Both fields are pure data: nothing in pkg/app reads
	// them. They live here only to centralise 12-factor configuration.
	OTelExporterEndpoint string
}

// LoadConfig reads the framework's configuration from environment
// variables and returns a populated Config. Missing values fall back
// to safe defaults; only AvailableLocales is validated (must not be
// empty after parsing) because every other field can sensibly default
// to "" in development.
//
// The function is pure: it does not touch the filesystem, does not
// open sockets, and does not call slog.SetDefault. All side effects
// are deferred to the composition root.
func LoadConfig() (Config, error) {
	cfg := Config{
		AppBind:          getEnv("APP_BIND", "127.0.0.1:8080"),
		DataSource:       getEnv("APP_DATA_SOURCE", "fixture"),
		StaticDir:        getEnv("APP_STATIC_DIR", "internal/site/web/static"),
		DefaultLocale:    getEnv("APP_DEFAULT_LOCALE", "en"),
		AvailableLocales: parseLocales(getEnv("APP_AVAILABLE_LOCALES", "en,ru")),

		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURI:  os.Getenv("OIDC_REDIRECT_URI"),
		SessionKey:       os.Getenv("SESSION_KEY"),

		HTTPReadTimeout:       parseDuration("APP_HTTP_READ_TIMEOUT", defaultHTTPReadTimeout),
		HTTPReadHeaderTimeout: parseDuration("APP_HTTP_READ_HEADER_TIMEOUT", defaultHTTPReadHeaderTimeout),
		HTTPWriteTimeout:      parseDuration("APP_HTTP_WRITE_TIMEOUT", defaultHTTPWriteTimeout),
		HTTPIdleTimeout:       parseDuration("APP_HTTP_IDLE_TIMEOUT", defaultHTTPIdleTimeout),
		HTTPMaxHeaderBytes:    parseEnvInt("APP_HTTP_MAX_HEADER_BYTES", defaultHTTPMaxHeaderBytes),
		HTTPShutdownTimeout:   parseDuration("APP_HTTP_SHUTDOWN_TIMEOUT", defaultHTTPShutdownTimeout),

		LogLevel:  strings.ToLower(getEnv("LOG_LEVEL", "info")),
		LogFormat: strings.ToLower(getEnv("LOG_FORMAT", "text")),

		HealthLivePath:  os.Getenv("HEALTH_LIVE_PATH"),
		HealthReadyPath: os.Getenv("HEALTH_READY_PATH"),
		MetricsPath:     os.Getenv("METRICS_PATH"),

		OTelServiceName:      os.Getenv("OTEL_SERVICE_NAME"),
		OTelExporterEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	if len(cfg.AvailableLocales) == 0 {
		return Config{}, fmt.Errorf("APP_AVAILABLE_LOCALES must contain at least one locale")
	}

	return cfg, nil
}

// parseDuration reads an environment variable as time.Duration. A missing
// or malformed value falls back to fallback. Non-positive values are also
// treated as malformed.
func parseDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// parseEnvInt reads a positive int from an environment variable.
func parseEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseLocales(raw string) []string {
	parts := strings.Split(raw, ",")
	unique := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		locale := strings.TrimSpace(strings.ToLower(part))
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}
		seen[locale] = struct{}{}
		unique = append(unique, locale)
	}

	return unique
}
