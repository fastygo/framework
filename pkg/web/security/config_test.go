package security

import (
	"testing"
)

func TestDefaultConfig_Sane(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Errorf("Enabled: default must be true")
	}
	if cfg.MaxBodySize != 1<<20 {
		t.Errorf("MaxBodySize: got %d, want %d", cfg.MaxBodySize, 1<<20)
	}
	if cfg.PageRateLimit != 50 || cfg.PageRateBurst != 100 {
		t.Errorf("rate defaults: got %v/%d, want 50/100", cfg.PageRateLimit, cfg.PageRateBurst)
	}
	if cfg.FrameOptions != "DENY" {
		t.Errorf("FrameOptions: got %q, want DENY", cfg.FrameOptions)
	}
	if !cfg.BlockEmptyUA {
		t.Errorf("BlockEmptyUA: default must be true")
	}
	if cfg.TrustProxy {
		t.Errorf("TrustProxy: default must be false")
	}
}

func TestLoadConfig_DefaultsWhenNoEnv(t *testing.T) {
	clearSecurityEnv(t)
	got := LoadConfig()
	want := DefaultConfig()
	if got != want {
		t.Fatalf("LoadConfig with no env: got %+v, want %+v", got, want)
	}
}

func TestLoadConfig_OverlaysEnv(t *testing.T) {
	clearSecurityEnv(t)
	t.Setenv("APP_SECURITY_HSTS", "true")
	t.Setenv("APP_SECURITY_FRAME_OPTIONS", "SAMEORIGIN")
	t.Setenv("APP_SECURITY_CSP", "default-src 'self'")
	t.Setenv("APP_SECURITY_PERMISSIONS", "camera=()")
	t.Setenv("APP_SECURITY_MAX_BODY_BYTES", "2048")
	t.Setenv("APP_SECURITY_RATE_PER_IP", "10.5")
	t.Setenv("APP_SECURITY_RATE_BURST", "20")
	t.Setenv("APP_SECURITY_TRUST_PROXY", "true")
	t.Setenv("APP_SECURITY_BLOCK_EMPTY_UA", "false")
	t.Setenv("APP_SECURITY_ENABLED", "false")

	got := LoadConfig()

	if !got.HSTS {
		t.Errorf("HSTS: got false, want true")
	}
	if got.FrameOptions != "SAMEORIGIN" {
		t.Errorf("FrameOptions: got %q, want SAMEORIGIN", got.FrameOptions)
	}
	if got.CSP != "default-src 'self'" {
		t.Errorf("CSP: got %q", got.CSP)
	}
	if got.Permissions != "camera=()" {
		t.Errorf("Permissions: got %q", got.Permissions)
	}
	if got.MaxBodySize != 2048 {
		t.Errorf("MaxBodySize: got %d, want 2048", got.MaxBodySize)
	}
	if got.PageRateLimit != 10.5 {
		t.Errorf("PageRateLimit: got %v, want 10.5", got.PageRateLimit)
	}
	if got.PageRateBurst != 20 {
		t.Errorf("PageRateBurst: got %d, want 20", got.PageRateBurst)
	}
	if !got.TrustProxy {
		t.Errorf("TrustProxy: got false, want true")
	}
	if got.BlockEmptyUA {
		t.Errorf("BlockEmptyUA: got true, want false")
	}
	if got.Enabled {
		t.Errorf("Enabled: got true, want false")
	}
}

func TestLoadConfig_MalformedFallsBackToDefaults(t *testing.T) {
	// Garbage in any of the typed env vars must fall back to the
	// default value, never silently disable a limiter.
	clearSecurityEnv(t)
	t.Setenv("APP_SECURITY_HSTS", "notabool")
	t.Setenv("APP_SECURITY_MAX_BODY_BYTES", "notanint")
	t.Setenv("APP_SECURITY_RATE_PER_IP", "notafloat")
	t.Setenv("APP_SECURITY_RATE_BURST", "notanint")

	got := LoadConfig()
	def := DefaultConfig()

	if got.HSTS != def.HSTS {
		t.Errorf("HSTS: got %v, want default %v", got.HSTS, def.HSTS)
	}
	if got.MaxBodySize != def.MaxBodySize {
		t.Errorf("MaxBodySize: got %d, want default %d", got.MaxBodySize, def.MaxBodySize)
	}
	if got.PageRateLimit != def.PageRateLimit {
		t.Errorf("PageRateLimit: got %v, want default %v", got.PageRateLimit, def.PageRateLimit)
	}
	if got.PageRateBurst != def.PageRateBurst {
		t.Errorf("PageRateBurst: got %d, want default %d", got.PageRateBurst, def.PageRateBurst)
	}
}

func TestLoadConfig_NonPositiveNumericsFallBack(t *testing.T) {
	// Zero and negative values for sized limiters are treated as
	// malformed: we never want a config that silently disables rate
	// limiting or body cap by typoing a "0".
	clearSecurityEnv(t)
	t.Setenv("APP_SECURITY_MAX_BODY_BYTES", "0")
	t.Setenv("APP_SECURITY_RATE_PER_IP", "-1")
	t.Setenv("APP_SECURITY_RATE_BURST", "0")

	got := LoadConfig()
	def := DefaultConfig()

	if got.MaxBodySize != def.MaxBodySize {
		t.Errorf("MaxBodySize: got %d, want default %d", got.MaxBodySize, def.MaxBodySize)
	}
	if got.PageRateLimit != def.PageRateLimit {
		t.Errorf("PageRateLimit: got %v, want default %v", got.PageRateLimit, def.PageRateLimit)
	}
	if got.PageRateBurst != def.PageRateBurst {
		t.Errorf("PageRateBurst: got %d, want default %d", got.PageRateBurst, def.PageRateBurst)
	}
}

func TestLoadConfig_CSPEmptyEnvIsEmptyString(t *testing.T) {
	// CSP intentionally uses os.Getenv (not getEnv) so an empty env
	// erases the default — we want apps to opt in to a CSP, never
	// inherit a wrong one.
	clearSecurityEnv(t)
	got := LoadConfig()
	if got.CSP != "" {
		t.Errorf("CSP default: got %q, want empty", got.CSP)
	}
}

func TestGetEnv_FallsBackOnEmpty(t *testing.T) {
	clearSecurityEnv(t)
	if got := getEnv("APP_SECURITY_DOES_NOT_EXIST", "fallback"); got != "fallback" {
		t.Errorf("getEnv unset: got %q, want fallback", got)
	}
	t.Setenv("APP_SECURITY_TEST_GETENV", "value")
	if got := getEnv("APP_SECURITY_TEST_GETENV", "fallback"); got != "value" {
		t.Errorf("getEnv set: got %q, want value", got)
	}
}

// clearSecurityEnv unsets every APP_SECURITY_* variable observed by
// LoadConfig, restoring the originals on test teardown via t.Setenv.
func clearSecurityEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_SECURITY_HSTS",
		"APP_SECURITY_FRAME_OPTIONS",
		"APP_SECURITY_CSP",
		"APP_SECURITY_PERMISSIONS",
		"APP_SECURITY_MAX_BODY_BYTES",
		"APP_SECURITY_RATE_PER_IP",
		"APP_SECURITY_RATE_BURST",
		"APP_SECURITY_TRUST_PROXY",
		"APP_SECURITY_BLOCK_EMPTY_UA",
		"APP_SECURITY_ENABLED",
	}
	for _, k := range keys {
		t.Setenv(k, "")
		// t.Setenv with "" still defines the var; for parseBool
		// emptiness is treated as "use fallback", which is exactly
		// what we want for a clean baseline.
	}
}
