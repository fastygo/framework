package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/web/health"
)

type fakeHealthFeature struct {
	id  string
	err error
}

func (f *fakeHealthFeature) ID() string                          { return f.id }
func (f *fakeHealthFeature) Routes(mux *http.ServeMux)            {}
func (f *fakeHealthFeature) NavItems() []NavItem                  { return nil }
func (f *fakeHealthFeature) HealthCheck(ctx context.Context) error { return f.err }

func TestApp_HealthEndpoints_FeatureCheckersWired(t *testing.T) {
	t.Parallel()

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithHealthEndpoints("/healthz", "/readyz").
		WithFeature(&fakeHealthFeature{id: "ok-feature"}).
		WithFeature(&fakeHealthFeature{id: "broken-feature", err: errors.New("simulated failure")}).
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("/readyz = %d, want 503", rec.Code)
	}

	var body struct {
		Status health.Status   `json:"status"`
		Checks []health.Result `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != health.StatusDown {
		t.Errorf("body.Status = %s, want down", body.Status)
	}
	names := make(map[string]health.Status)
	for _, r := range body.Checks {
		names[r.Name] = r.Status
	}
	if names["ok-feature"] != health.StatusUp {
		t.Errorf("ok-feature status = %s, want up", names["ok-feature"])
	}
	if names["broken-feature"] != health.StatusDown {
		t.Errorf("broken-feature status = %s, want down", names["broken-feature"])
	}
	if !strings.Contains(rec.Body.String(), "simulated failure") {
		t.Errorf("expected error string in body, got %s", rec.Body.String())
	}
}

func TestApp_HealthEndpoints_DisabledByDefault(t *testing.T) {
	t.Parallel()

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when health endpoints not opted-in, got %d", rec.Code)
	}
}

func TestApp_HealthEndpoints_ExtraChecker(t *testing.T) {
	t.Parallel()

	called := 0
	extra := health.CheckerFunc{
		CheckerName: "infra-db",
		Fn: func(ctx context.Context) error {
			called++
			return nil
		},
	}

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithHealthEndpoints("", "/readyz").
		AddHealthChecker(extra).
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/readyz = %d, want 200", rec.Code)
	}
	if called != 1 {
		t.Errorf("extra checker called %d times, want 1", called)
	}
}
