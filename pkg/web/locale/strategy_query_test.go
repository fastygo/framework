package locale

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryStrategyResolveAlias(t *testing.T) {
	strategy := &QueryStrategy{
		Param:      "lang",
		Aliases:    []string{"translate"},
		Available:  []string{"en", "ru"},
		Default:    "en",
		ValueMap:   map[string]string{"english": "en", "russian": "ru"},
	}

	req := httptest.NewRequest(http.MethodGet, "/?translate=russian&foo=1", nil)
	got, _ := strategy.Resolve(req)
	if got != "ru" {
		t.Fatalf("expected ru, got %q", got)
	}
}

func TestQueryStrategyResolveAcceptLanguageFallback(t *testing.T) {
	strategy := &QueryStrategy{
		Param:     "lang",
		Available: []string{"en", "ru"},
		Default:   "en",
	}

	req := httptest.NewRequest(http.MethodGet, "/?foo=1", nil)
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.7")

	got, _ := strategy.Resolve(req)
	if got != "ru" {
		t.Fatalf("expected ru, got %q", got)
	}
}

func TestQueryStrategyHref(t *testing.T) {
	strategy := &QueryStrategy{
		Param:      "lang",
		Aliases:    []string{"translate"},
		Available:  []string{"en", "ru"},
		Default:    "en",
		ValueMap:   map[string]string{"english": "en", "russian": "ru"},
	}

	req := httptest.NewRequest(http.MethodGet, "/docs?foo=1", nil)
	got := strategy.Href(req, "ru")
	if got != "/docs?foo=1&lang=ru" {
		t.Fatalf("expected /docs?foo=1&lang=ru, got %q", got)
	}
}
