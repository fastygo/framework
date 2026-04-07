package security

import (
	"os"
	"strconv"
)

type Config struct {
	HSTS         bool
	FrameOptions string
	CSP          string
	Permissions  string
	MaxBodySize  int64

	PageRateLimit float64
	PageRateBurst int

	TrustProxy bool
	BlockEmptyUA bool
	Enabled bool
}

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
