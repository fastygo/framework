package locale

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareStoresLocaleContextAndPersistsCookie(t *testing.T) {
	strategy := &CookieDecorator{
		Inner: &QueryStrategy{
			Available: []string{"en", "ru", "de"},
			Default:   "en",
		},
		Name: "lang",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := From(r.Context()); got != "ru" {
			t.Fatalf("expected locale to be ru, got %q", got)
		}

		available := Available(r.Context())
		if len(available) != 3 || available[0] != "en" || available[1] != "ru" || available[2] != "de" {
			t.Fatalf("expected available locales to be normalized")
		}

		if SPAMode(r.Context()) {
			t.Fatalf("expected SPA mode to be false by default")
		}

		rewrite := RequestFromContext(r.Context())
		if rewrite == nil {
			t.Fatalf("expected request context to include rewritten request")
		}
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?lang=ru", nil)
	Middleware(strategy)(handler).ServeHTTP(recorder, req)

	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected locale middleware to persist a cookie")
	}
}

func TestMiddlewareSupportsSPAModeTrue(t *testing.T) {
	strategy := &QueryStrategy{
		Available: []string{"en", "ru"},
		Default:   "en",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !SPAMode(r.Context()) {
			t.Fatalf("expected SPA mode true in context")
		}
		if got := From(r.Context()); got != "ru" {
			t.Fatalf("expected locale ru from query, got %q", got)
		}
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?lang=ru", nil)
	MiddlewareWithSPAMode(strategy, true)(handler).ServeHTTP(recorder, req)
	if recorder.Result().StatusCode != 200 {
		t.Fatalf("unexpected status %d", recorder.Result().StatusCode)
	}
}
