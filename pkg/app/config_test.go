package app

import (
	"testing"
	"time"
)

func TestConfig_HTTPDefaults(t *testing.T) {
	t.Setenv("APP_AVAILABLE_LOCALES", "en")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	checks := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"HTTPReadTimeout", cfg.HTTPReadTimeout, defaultHTTPReadTimeout},
		{"HTTPReadHeaderTimeout", cfg.HTTPReadHeaderTimeout, defaultHTTPReadHeaderTimeout},
		{"HTTPWriteTimeout", cfg.HTTPWriteTimeout, defaultHTTPWriteTimeout},
		{"HTTPIdleTimeout", cfg.HTTPIdleTimeout, defaultHTTPIdleTimeout},
		{"HTTPShutdownTimeout", cfg.HTTPShutdownTimeout, defaultHTTPShutdownTimeout},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if cfg.HTTPMaxHeaderBytes != defaultHTTPMaxHeaderBytes {
		t.Errorf("HTTPMaxHeaderBytes = %d, want %d", cfg.HTTPMaxHeaderBytes, defaultHTTPMaxHeaderBytes)
	}
}

func TestConfig_HTTPEnvOverride(t *testing.T) {
	t.Setenv("APP_AVAILABLE_LOCALES", "en")
	t.Setenv("APP_HTTP_READ_TIMEOUT", "3s")
	t.Setenv("APP_HTTP_READ_HEADER_TIMEOUT", "750ms")
	t.Setenv("APP_HTTP_SHUTDOWN_TIMEOUT", "12s")
	t.Setenv("APP_HTTP_MAX_HEADER_BYTES", "65536")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.HTTPReadTimeout != 3*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 3s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPReadHeaderTimeout != 750*time.Millisecond {
		t.Errorf("HTTPReadHeaderTimeout = %v, want 750ms", cfg.HTTPReadHeaderTimeout)
	}
	if cfg.HTTPShutdownTimeout != 12*time.Second {
		t.Errorf("HTTPShutdownTimeout = %v, want 12s", cfg.HTTPShutdownTimeout)
	}
	if cfg.HTTPMaxHeaderBytes != 65536 {
		t.Errorf("HTTPMaxHeaderBytes = %d, want 65536", cfg.HTTPMaxHeaderBytes)
	}
}

func TestConfig_BadDurationFallsBack(t *testing.T) {
	t.Setenv("APP_AVAILABLE_LOCALES", "en")
	t.Setenv("APP_HTTP_READ_TIMEOUT", "garbage")
	t.Setenv("APP_HTTP_WRITE_TIMEOUT", "-5s")
	t.Setenv("APP_HTTP_MAX_HEADER_BYTES", "not-a-number")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.HTTPReadTimeout != defaultHTTPReadTimeout {
		t.Errorf("garbage duration must fall back: got %v", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != defaultHTTPWriteTimeout {
		t.Errorf("negative duration must fall back: got %v", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPMaxHeaderBytes != defaultHTTPMaxHeaderBytes {
		t.Errorf("garbage int must fall back: got %d", cfg.HTTPMaxHeaderBytes)
	}
}

func TestConfig_ObservabilityDefaults(t *testing.T) {
	t.Setenv("APP_AVAILABLE_LOCALES", "en")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.HealthLivePath != "" || cfg.HealthReadyPath != "" {
		t.Errorf("health paths must be opt-in (got %q / %q)",
			cfg.HealthLivePath, cfg.HealthReadyPath)
	}
	if cfg.MetricsPath != "" {
		t.Errorf("MetricsPath must be opt-in (got %q)", cfg.MetricsPath)
	}
	if cfg.OTelServiceName != "" || cfg.OTelExporterEndpoint != "" {
		t.Errorf("OTel fields must be empty by default")
	}
}

func TestConfig_ObservabilityEnvOverride(t *testing.T) {
	t.Setenv("APP_AVAILABLE_LOCALES", "en")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("LOG_FORMAT", "JSON")
	t.Setenv("HEALTH_LIVE_PATH", "/healthz")
	t.Setenv("HEALTH_READY_PATH", "/readyz")
	t.Setenv("METRICS_PATH", "/metrics")
	t.Setenv("OTEL_SERVICE_NAME", "demo-app")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel:4318")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug (lowercased)", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json (lowercased)", cfg.LogFormat)
	}
	if cfg.HealthLivePath != "/healthz" || cfg.HealthReadyPath != "/readyz" {
		t.Errorf("health paths not propagated: %+v", cfg)
	}
	if cfg.MetricsPath != "/metrics" {
		t.Errorf("MetricsPath = %q", cfg.MetricsPath)
	}
	if cfg.OTelServiceName != "demo-app" {
		t.Errorf("OTelServiceName = %q", cfg.OTelServiceName)
	}
	if cfg.OTelExporterEndpoint != "http://otel:4318" {
		t.Errorf("OTelExporterEndpoint = %q", cfg.OTelExporterEndpoint)
	}
}
