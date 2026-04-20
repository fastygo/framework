package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/web/security"
)

func TestBuilder_AppliesHTTPDefaultsToLiteralConfig(t *testing.T) {
	t.Parallel()

	// Config built without LoadConfig — every HTTP timeout is zero.
	cfg := Config{AppBind: "127.0.0.1:0"}
	builder := New(cfg)

	got := builder.cfg
	if got.HTTPReadTimeout != defaultHTTPReadTimeout {
		t.Errorf("HTTPReadTimeout = %v, want default %v", got.HTTPReadTimeout, defaultHTTPReadTimeout)
	}
	if got.HTTPShutdownTimeout != defaultHTTPShutdownTimeout {
		t.Errorf("HTTPShutdownTimeout = %v, want default %v", got.HTTPShutdownTimeout, defaultHTTPShutdownTimeout)
	}
	if got.HTTPMaxHeaderBytes != defaultHTTPMaxHeaderBytes {
		t.Errorf("HTTPMaxHeaderBytes = %d, want default %d", got.HTTPMaxHeaderBytes, defaultHTTPMaxHeaderBytes)
	}
}

func TestBuilder_WithHTTPServerOptionsPartialOverride(t *testing.T) {
	t.Parallel()

	builder := New(Config{AppBind: "127.0.0.1:0"}).
		WithHTTPServerOptions(HTTPServerOptions{
			WriteTimeout:    25 * time.Second,
			ShutdownTimeout: 1 * time.Second,
		})

	if builder.cfg.HTTPWriteTimeout != 25*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want 25s", builder.cfg.HTTPWriteTimeout)
	}
	if builder.cfg.HTTPShutdownTimeout != time.Second {
		t.Errorf("HTTPShutdownTimeout = %v, want 1s", builder.cfg.HTTPShutdownTimeout)
	}
	// Untouched fields keep their (default) value.
	if builder.cfg.HTTPReadTimeout != defaultHTTPReadTimeout {
		t.Errorf("HTTPReadTimeout = %v, want default %v", builder.cfg.HTTPReadTimeout, defaultHTTPReadTimeout)
	}
}

// TestApp_ShutdownHonoursTimeout verifies that App.Run respects
// HTTPShutdownTimeout when a stubborn handler refuses to finish.
//
// The blocking handler is released and the test HTTP transport's idle
// connections are closed before returning, so goleak in the package-wide
// TestMain still passes.
func TestApp_ShutdownHonoursTimeout(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	feature := &blockingFeature{block: make(chan struct{})}
	t.Cleanup(func() { close(feature.block) })

	app := New(Config{AppBind: addr}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithHTTPServerOptions(HTTPServerOptions{ShutdownTimeout: 50 * time.Millisecond}).
		WithFeature(feature).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)

	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Timeout: 200 * time.Millisecond, Transport: transport}
	go func() { _, _ = client.Get("http://" + addr + "/block") }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("App.Run did not return within 2s after ctx cancellation")
	}
}

type blockingFeature struct {
	block chan struct{}
}

func (f *blockingFeature) ID() string              { return "blocking" }
func (f *blockingFeature) NavItems() []NavItem     { return nil }
func (f *blockingFeature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
		<-f.block
	})
}

// disableSecurity returns a minimal security.Config that skips the full
// rate-limiter/antibot pipeline so shutdown tests stay fast and isolated.
func disableSecurity() security.Config {
	cfg := security.DefaultConfig()
	cfg.Enabled = false
	return cfg
}