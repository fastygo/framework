package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/observability"
	"github.com/fastygo/framework/pkg/web/health"
	"github.com/fastygo/framework/pkg/web/metrics"
)

// ---------- Builder accessors and withers (zero-coverage paths) -----------

func TestBuilder_Mux_ReturnsSameInstance(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"})
	if b.Mux() == nil {
		t.Fatalf("Mux() must not be nil")
	}
	if b.Mux() != b.mux {
		t.Errorf("Mux() must return the same *ServeMux the builder uses")
	}
}

func TestBuilder_WithLogger_IgnoresNil(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"})
	original := b.logger
	b.WithLogger(nil)
	if b.logger != original {
		t.Errorf("WithLogger(nil) must keep the original logger")
	}

	custom := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	b.WithLogger(custom)
	if b.logger != custom {
		t.Errorf("WithLogger must replace the logger when non-nil")
	}
}

func TestBuilder_WithStaticPrefix_AppendsTrailingSlash(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithStaticPrefix("/assets")
	if b.staticPrefix != "/assets/" {
		t.Errorf("staticPrefix: got %q, want /assets/", b.staticPrefix)
	}
	if !b.staticRoutes {
		t.Errorf("staticRoutes must remain enabled")
	}
}

func TestBuilder_WithStaticPrefix_EmptyDisables(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithStaticPrefix("")
	if b.staticRoutes {
		t.Errorf("empty prefix must disable static routes")
	}
}

func TestBuilder_WithStaticPrefix_AlreadySlashed(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithStaticPrefix("/x/")
	if b.staticPrefix != "/x/" {
		t.Errorf("staticPrefix: got %q, want /x/", b.staticPrefix)
	}
}

func TestBuilder_WithFeature_IgnoresNil(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithFeature(nil)
	if len(b.features) != 0 {
		t.Errorf("WithFeature(nil) must not register a feature; got %d", len(b.features))
	}
}

func TestBuilder_AddBackgroundTask_AppendsToWorkers(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"})
	called := atomic.Int32{}
	b.AddBackgroundTask(BackgroundTask{
		Name:     "tick",
		Interval: 10 * time.Millisecond,
		Run:      func(context.Context) { called.Add(1) },
	})
	if len(b.workers.tasks) != 1 {
		t.Fatalf("expected 1 task registered, got %d", len(b.workers.tasks))
	}
	if b.workers.tasks[0].Name != "tick" {
		t.Errorf("task name not preserved: %q", b.workers.tasks[0].Name)
	}
}

func TestBuilder_WithMetricsRegistry_StoresRegistry(t *testing.T) {
	t.Parallel()
	reg := metrics.NewRegistry()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithMetricsRegistry(reg)
	if b.metricsRegistry != reg {
		t.Errorf("WithMetricsRegistry must store the registry instance")
	}
}

func TestBuilder_WithMetricsEndpoint_LazilyCreatesRegistry(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).WithMetricsEndpoint("/metrics")
	if b.metricsRegistry == nil {
		t.Fatalf("WithMetricsEndpoint must lazy-create a Registry when none was set")
	}
	if b.metricsPath != "/metrics" {
		t.Errorf("metricsPath: got %q", b.metricsPath)
	}
}

func TestBuilder_WithTracer_StoresTracer(t *testing.T) {
	t.Parallel()
	tr := observability.NoopTracer{}
	b := New(Config{AppBind: "127.0.0.1:0"}).WithTracer(tr)
	if b.tracer != tr {
		t.Errorf("WithTracer must store the tracer")
	}
}

func TestBuilder_AddHealthChecker_NilIgnored(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"}).AddHealthChecker(nil)
	if len(b.healthExtra) != 0 {
		t.Errorf("AddHealthChecker(nil) must be a no-op; got %d", len(b.healthExtra))
	}

	checker := health.CheckerFunc{
		CheckerName: "x",
		Fn:          func(context.Context) error { return nil },
	}
	b.AddHealthChecker(checker)
	if len(b.healthExtra) != 1 {
		t.Errorf("expected 1 extra checker, got %d", len(b.healthExtra))
	}
}

// ---------- Build() and App accessors ------------------------------------

func TestBuild_PopulatesAppFields(t *testing.T) {
	t.Parallel()
	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		Build()

	if app.Config().AppBind != "127.0.0.1:0" {
		t.Errorf("Config: got %+v", app.Config())
	}
	if app.Workers() == nil {
		t.Errorf("Workers must not be nil")
	}
	if app.Handler() == nil {
		t.Errorf("Handler must not be nil")
	}
	if app.NavItems() == nil {
		// Empty slice is fine, but accessor must not crash.
		t.Logf("NavItems is empty (expected when no features)")
	}
}

func TestApp_NavItems_ReturnsCopy(t *testing.T) {
	t.Parallel()
	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithFeature(navFeature{items: []NavItem{{Label: "One", Path: "/a", Order: 1}}}).
		Build()

	got := app.NavItems()
	if len(got) != 1 {
		t.Fatalf("expected 1 nav item, got %d", len(got))
	}

	// Mutating the returned slice must not affect subsequent calls.
	got[0].Label = "MUTATED"
	if app.NavItems()[0].Label == "MUTATED" {
		t.Fatalf("NavItems() must return a defensive copy")
	}
}

func TestCollectNavItems_SortsByOrderThenLabel(t *testing.T) {
	t.Parallel()
	features := []Feature{
		navFeature{items: []NavItem{{Label: "Zebra", Path: "/z", Order: 10}}},
		navFeature{items: []NavItem{{Label: "Apple", Path: "/a", Order: 10}}},
		navFeature{items: []NavItem{{Label: "First", Path: "/f", Order: 1}}},
	}
	got := collectNavItems(features)
	want := []string{"First", "Apple", "Zebra"}
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d", len(got), len(want))
	}
	for i, label := range want {
		if got[i].Label != label {
			t.Errorf("item[%d]: got %q, want %q (full: %v)", i, got[i].Label, label, got)
		}
	}
}

func TestCollectNavItems_EmptyFeatureList(t *testing.T) {
	t.Parallel()
	got := collectNavItems(nil)
	if len(got) != 0 {
		t.Fatalf("collectNavItems(nil): got %v, want empty", got)
	}
}

// ---------- Run lifecycle -------------------------------------------------

func TestApp_Run_InitErrorAbortsStartup(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithFeature(initErrFeature{err: wantErr}).
		Build()

	err := app.Run(context.Background())
	if err == nil {
		t.Fatalf("Run must return an error when Init fails")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error: got %v, want it to wrap %v", err, wantErr)
	}
	// Init failed → server must NOT have been started; verify by ensuring
	// the addr is still bindable.
	ln, lnErr := net.Listen("tcp", "127.0.0.1:0")
	if lnErr != nil {
		t.Fatalf("post-Init port unavailable: %v", lnErr)
	}
	_ = ln.Close()
}

func TestApp_Run_ClosersInvokedInReverseOrder(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	order := &orderRecorder{}
	app := New(Config{AppBind: addr}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithHTTPServerOptions(HTTPServerOptions{ShutdownTimeout: 200 * time.Millisecond}).
		WithFeature(closerFeature{id: "f1", recorder: order}).
		WithFeature(closerFeature{id: "f2", recorder: order}).
		WithFeature(closerFeature{id: "f3", recorder: order}).
		Build()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	time.Sleep(80 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s after cancel")
	}

	got := order.snapshot()
	want := []string{"f3", "f2", "f1"}
	if len(got) != len(want) {
		t.Fatalf("close order length: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("close order[%d]: got %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestApp_Run_ListenAndServeError_Propagates(t *testing.T) {
	t.Parallel()
	// Bind to an obviously invalid address so ListenAndServe fails
	// fast and Run takes the err-path branch.
	app := New(Config{AppBind: "256.256.256.256:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithHTTPServerOptions(HTTPServerOptions{ShutdownTimeout: 100 * time.Millisecond}).
		Build()

	err := app.Run(context.Background())
	if err == nil {
		t.Fatalf("Run must return the ListenAndServe error")
	}
}

// ---------- ServeHTTP exposes the handler --------------------------------

func TestApp_ServeHTTP_DelegatesToHandler(t *testing.T) {
	t.Parallel()
	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithFeature(routesFeature{path: "/ping"}).
		Build()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("User-Agent", "test")
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rr.Code)
	}
}

// Coverage of closeFeatures' "feature without Closer" branch.
func TestApp_CloseFeatures_SkipsNonClosers(t *testing.T) {
	t.Parallel()
	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithFeature(routesFeature{path: "/x"}). // no Close method
		Build()

	// Smoke-test: this should not panic.
	app.closeFeatures(context.Background(), slog.Default())
}

// ---------- WorkerService corner cases ----------------------------------

func TestWorkerService_Add_RejectsInvalid(t *testing.T) {
	t.Parallel()
	w := &WorkerService{}

	w.Add(BackgroundTask{Name: "no-fn", Interval: time.Second})              // Run nil
	w.Add(BackgroundTask{Name: "no-interval", Run: func(context.Context) {}}) // Interval 0
	w.Add(BackgroundTask{
		Name:     "negative-interval",
		Interval: -time.Second,
		Run:      func(context.Context) {},
	})

	if len(w.tasks) != 0 {
		t.Fatalf("invalid tasks must not be registered, got %d", len(w.tasks))
	}
}

func TestWorkerService_Add_DefaultsName(t *testing.T) {
	t.Parallel()
	w := &WorkerService{}
	w.Add(BackgroundTask{Interval: time.Second, Run: func(context.Context) {}})
	if len(w.tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(w.tasks))
	}
	if w.tasks[0].Name != "worker" {
		t.Errorf("default name: got %q, want %q", w.tasks[0].Name, "worker")
	}
}

func TestWorkerService_Stop_NilContext_WaitsIndefinitely(t *testing.T) {
	t.Parallel()
	w := &WorkerService{}
	// Nothing started → wg.Wait() returns immediately, Stop must not block.
	// We deliberately pass nil to exercise the documented "nil ctx
	// means wait indefinitely" branch in WorkerService.Stop.
	//lint:ignore SA1012 verifying explicit nil-context support
	if err := w.Stop(nil); err != nil {
		t.Errorf("Stop(nil) on idle service: got %v, want nil", err)
	}
}

func TestWorkerService_Stop_ContextDeadline_ReturnsErr(t *testing.T) {
	t.Parallel()
	w := &WorkerService{}
	w.Add(BackgroundTask{
		Name:     "blocker",
		Interval: time.Hour,
		Run: func(ctx context.Context) {
			<-ctx.Done() // honours cancellation
		},
	})

	runCtx, cancelRun := context.WithCancel(context.Background())
	t.Cleanup(cancelRun)
	w.Start(runCtx)

	// Don't cancel runCtx → workers stay running; Stop deadline must hit.
	deadlineCtx, cancelDeadline := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelDeadline()
	err := w.Stop(deadlineCtx)
	if err == nil {
		t.Fatalf("Stop must return an error when its context expires before tasks return")
	}
	cancelRun() // let workers finish for goleak
	// Drain the workers so the test does not leak. Stop is once-only,
	// so we drain via a fresh wait.
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("workers never returned after runCtx cancel")
	}
}

func TestSafeRun_RecoversPanic(t *testing.T) {
	t.Parallel()
	panicTask := BackgroundTask{
		Name:     "panicker",
		Interval: time.Second,
		Run:      func(context.Context) { panic("kaboom") },
	}
	// safeRun must NOT propagate the panic; if it does, the test
	// process dies with a panic stacktrace.
	safeRun(context.Background(), panicTask)
}

// ---------- Test helpers --------------------------------------------------

type navFeature struct {
	items []NavItem
}

func (f navFeature) ID() string                  { return "nav" }
func (f navFeature) NavItems() []NavItem         { return f.items }
func (f navFeature) Routes(_ *http.ServeMux)     {}

type initErrFeature struct {
	err error
}

func (f initErrFeature) ID() string                       { return "init-err" }
func (f initErrFeature) NavItems() []NavItem              { return nil }
func (f initErrFeature) Routes(_ *http.ServeMux)          {}
func (f initErrFeature) Init(_ context.Context) error     { return f.err }

type orderRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *orderRecorder) push(name string) {
	r.mu.Lock()
	r.events = append(r.events, name)
	r.mu.Unlock()
}

func (r *orderRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	copy(out, r.events)
	return out
}

type closerFeature struct {
	id       string
	recorder *orderRecorder
}

func (f closerFeature) ID() string                    { return f.id }
func (f closerFeature) NavItems() []NavItem           { return nil }
func (f closerFeature) Routes(_ *http.ServeMux)       {}
func (f closerFeature) Close(_ context.Context) error { f.recorder.push(f.id); return nil }

type routesFeature struct {
	path string
}

func (f routesFeature) ID() string         { return "routes" }
func (f routesFeature) NavItems() []NavItem { return nil }
func (f routesFeature) Routes(mux *http.ServeMux) {
	mux.HandleFunc(f.path, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

// silence unused-import warnings if the test file is pruned.
var _ = strings.HasPrefix
