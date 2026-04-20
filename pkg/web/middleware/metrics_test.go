package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/web/metrics"
)

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	t.Parallel()

	reg := metrics.NewRegistry()
	mw := MetricsMiddleware(reg)

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	for i := 0; i < 3; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", nil))
	}

	var buf bytes.Buffer
	if err := reg.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`http_requests_total{method="POST",status="201"} 3`,
		"http_requests_in_flight 0",
		"http_request_duration_seconds_count{method=\"POST\"} 3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n---\n%s", want, out)
		}
	}
}

func TestMetricsMiddleware_NilIsPassthrough(t *testing.T) {
	t.Parallel()

	called := false
	mw := MetricsMiddleware(nil)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("inner handler not invoked under nil registry")
	}
}

func TestMetricsMiddleware_DefaultStatus(t *testing.T) {
	t.Parallel()

	reg := metrics.NewRegistry()
	h := MetricsMiddleware(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no WriteHeader call -> implicit 200
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	var buf bytes.Buffer
	_ = reg.Write(&buf)
	if !strings.Contains(buf.String(), `status="200"`) {
		t.Fatalf("expected implicit 200 status, got:\n%s", buf.String())
	}
}
