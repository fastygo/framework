package security

import (
	"os"
	"strconv"
)

// Config aggregates the knobs for the security middlewares (headers,
// body limit, rate limit, anti-bot, method guard, secure file server).
// Use DefaultConfig as a starting point and override individual fields,
// or call LoadConfig to read APP_SECURITY_* environment variables.
type Config struct {
	// HSTS toggles the Strict-Transport-Security header. Enable only
	// when the entire site is served over HTTPS, including subdomains.
	HSTS bool
	// FrameOptions sets the X-Frame-Options header value (e.g. "DENY",
	// "SAMEORIGIN"). Empty disables the header.
	FrameOptions string
	// CSP, when non-empty, populates the Content-Security-Policy
	// header verbatim. Empty disables the header (no policy advertised).
	CSP string
	// Permissions populates the Permissions-Policy header (e.g.
	// "geolocation=(), microphone=()").
	Permissions string
	// MaxBodySize caps request body bytes (rejected with 413 if larger,
	// truncated with http.MaxBytesReader during reads). Zero disables.
	MaxBodySize int64

	// PageRateLimit is the steady-state requests-per-second per IP
	// allowed by the token-bucket rate limiter.
	PageRateLimit float64
	// PageRateBurst is the burst size of the token bucket — i.e. how
	// many tokens may accumulate when an IP is idle.
	PageRateBurst int

	// TrustProxy controls whether ClientIP honours X-Forwarded-For /
	// X-Real-IP. Set to true only when the deployment terminates
	// trusted proxies in front of the application.
	TrustProxy bool
	// BlockEmptyUA rejects requests with an empty User-Agent header
	// with HTTP 403. Helps deter naive scanners; legitimate clients
	// always send a UA.
	BlockEmptyUA bool
	// Enabled is the master switch the AppBuilder reads to decide
	// whether the security middleware chain is wired in at all.
	Enabled bool
}

// DefaultConfig returns the recommended starting configuration:
// HSTS off (enable explicitly when serving full HTTPS), DENY frames,
// no CSP (set per-application), 1MB body cap, 50 req/sec per IP with
// burst 100, proxy headers trusted, empty UA blocked, security on.
func DefaultConfig() Config {
	return Config{
		HSTS:         false,
		FrameOptions: "DENY",
		CSP:          "",
		Permissions:  "geolocation=(), microphone=(), camera=()",
		MaxBodySize:  1 << 20, // 1MB
		PageRateLimit: 50,
		PageRateBurst: 100,
		TrustProxy:   true,
		BlockEmptyUA: true,
		Enabled:      true,
	}
}

// LoadConfig overlays the APP_SECURITY_* environment variables on top
// of DefaultConfig. Missing or malformed values fall through to the
// defaults; non-positive numerics (rate, burst, body size) are also
// rejected as malformed to avoid accidentally disabling the limiter.
func LoadConfig() Config {
	cfg := DefaultConfig()
	cfg.HSTS = parseBool("APP_SECURITY_HSTS", cfg.HSTS)
	cfg.FrameOptions = getEnv("APP_SECURITY_FRAME_OPTIONS", cfg.FrameOptions)
	cfg.CSP = os.Getenv("APP_SECURITY_CSP")
	cfg.Permissions = getEnv("APP_SECURITY_PERMISSIONS", cfg.Permissions)
	cfg.MaxBodySize = parseInt64("APP_SECURITY_MAX_BODY_BYTES", cfg.MaxBodySize)
	cfg.PageRateLimit = parseFloat("APP_SECURITY_RATE_PER_IP", cfg.PageRateLimit)
	cfg.PageRateBurst = parseInt("APP_SECURITY_RATE_BURST", cfg.PageRateBurst)
	cfg.TrustProxy = parseBool("APP_SECURITY_TRUST_PROXY", cfg.TrustProxy)
	cfg.BlockEmptyUA = parseBool("APP_SECURITY_BLOCK_EMPTY_UA", cfg.BlockEmptyUA)
	cfg.Enabled = parseBool("APP_SECURITY_ENABLED", cfg.Enabled)
	return cfg
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
