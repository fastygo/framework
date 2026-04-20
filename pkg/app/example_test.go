package app_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/health"
)

// helloFeature is a tiny Feature used by the AppBuilder example. A
// real feature usually lives in internal/site/<name>/feature.go and
// is constructed by the composition root in cmd/server/main.go.
type helloFeature struct{}

func (helloFeature) ID() string { return "hello" }

func (helloFeature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/hello", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello, world"))
	})
}

func (helloFeature) NavItems() []app.NavItem {
	return []app.NavItem{{Label: "Hello", Path: "/hello", Order: 10}}
}

// ExampleAppBuilder shows the canonical composition-root pattern for
// the framework: load configuration, register one or more features
// through the fluent builder, then call Build to obtain a fully
// wired *App that satisfies http.Handler.
//
// Inside main() the resulting handler is normally passed to
// http.ListenAndServe (or to the worker-aware Run helpers). In tests
// we use httptest.ResponseRecorder to drive the same handler without
// opening a real socket — proof that the framework adds no hidden
// global state.
func ExampleAppBuilder() {
	cfg := app.Config{
		AppBind:          "127.0.0.1:0",
		DefaultLocale:    "en",
		AvailableLocales: []string{"en"},
	}

	a := app.New(cfg).
		WithFeature(helloFeature{}).
		Build()

	// Drive the assembled handler exactly as the real http.Server would.
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("User-Agent", "example/1.0") // AntiBot rejects empty UA.

	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	fmt.Println("status:", rec.Code)
	fmt.Println("body:", string(body))

	for _, item := range a.NavItems() {
		fmt.Printf("nav: %s -> %s\n", item.Label, item.Path)
	}

	// Output:
	// status: 200
	// body: hello, world
	// nav: Hello -> /hello
}

// ExampleAppBuilder_health demonstrates the built-in liveness and
// readiness endpoints. Once enabled they are served by the same mux
// the features use, so Kubernetes / Docker probes need no extra
// listener and inherit the same security middleware chain.
func ExampleAppBuilder_health() {
	cfg := app.Config{
		AppBind:          "127.0.0.1:0",
		DefaultLocale:    "en",
		AvailableLocales: []string{"en"},
	}

	a := app.New(cfg).
		WithHealthEndpoints("/healthz", "/readyz").
		AddHealthChecker(health.CheckerFunc{
			CheckerName: "example",
			Fn:          func(_ context.Context) error { return nil },
		}).
		Build()

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("User-Agent", "kube-probe/1.30")
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		fmt.Printf("%s -> %d\n", path, rec.Code)
	}

	// Output:
	// /healthz -> 200
	// /readyz -> 200
}
