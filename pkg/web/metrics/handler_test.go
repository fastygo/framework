package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ContentType(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Counter("c_total", "h").Inc()

	rec := httptest.NewRecorder()
	Handler(r).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain prefix", got)
	}
	if !strings.Contains(rec.Body.String(), "c_total 1") {
		t.Fatalf("body missing counter:\n%s", rec.Body.String())
	}
}

func TestHandler_NilRegistry(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	Handler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even with nil registry", rec.Code)
	}
}
