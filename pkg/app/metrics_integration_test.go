package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApp_MetricsEndpoint_ScrapeReturnsExpfmt(t *testing.T) {
	t.Parallel()

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		WithMetricsEndpoint("/metrics").
		Build()

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/non-existing-route", nil))
		if rec.Code == 0 {
			t.Fatalf("expected handler to write a status, got 0")
		}
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		`# TYPE http_requests_total counter`,
		`# TYPE http_requests_in_flight gauge`,
		`# TYPE http_request_duration_seconds histogram`,
		`http_requests_total{method="GET",status="404"} 5`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q\n---\n%s", want, body)
		}
	}

	// /metrics scrapes must NOT increment counters themselves.
	if strings.Contains(body, `path="/metrics"`) {
		t.Error("/metrics scrapes should be excluded from request metrics")
	}
}

func TestApp_MetricsEndpoint_DisabledByDefault(t *testing.T) {
	t.Parallel()

	app := New(Config{AppBind: "127.0.0.1:0"}).
		WithSecurity(disableSecurity()).
		DisableStatic().
		Build()

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when metrics not opted-in, got %d", rec.Code)
	}
}
