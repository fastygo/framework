package locale

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathPrefixStrategyResolveStripsLocalePrefix(t *testing.T) {
	strategy := &PathPrefixStrategy{
		Available: []string{"en", "ru"},
		Default:   "en",
	}

	req := httptest.NewRequest(http.MethodGet, "/ru/docs/quickstart?from=home", nil)
	gotLocale, rewritten := strategy.Resolve(req)

	if gotLocale != "ru" {
		t.Fatalf("expected ru locale, got %q", gotLocale)
	}
	if rewritten == nil {
		t.Fatal("expected rewritten request")
	}
	if rewritten.URL.Path != "/docs/quickstart" {
		t.Fatalf("expected rewritten path /docs/quickstart, got %q", rewritten.URL.Path)
	}
}

func TestPathPrefixStrategyHrefPrefixesLocale(t *testing.T) {
	strategy := &PathPrefixStrategy{
		Available: []string{"en", "ru"},
		Default:   "en",
	}

	req := httptest.NewRequest(http.MethodGet, "/docs/quickstart?from=home", nil)
	got := strategy.Href(req, "ru")
	if got != "/ru/docs/quickstart?from=home" {
		t.Fatalf("expected /ru/docs/quickstart?from=home, got %q", got)
	}
}

func TestPathPrefixStrategyHrefStripsExistingLocalePrefix(t *testing.T) {
	strategy := &PathPrefixStrategy{
		Available: []string{"en", "ru"},
		Default:   "en",
	}

	req := httptest.NewRequest(http.MethodGet, "/en/docs/quickstart?from=home", nil)
	got := strategy.Href(req, "ru")
	if got != "/ru/docs/quickstart?from=home" {
		t.Fatalf("expected /ru/docs/quickstart?from=home, got %q", got)
	}
}
