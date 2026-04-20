package locale

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNegotiatorResolveQueryWins(t *testing.T) {
	n := New("en", []string{"en", "ru", "de"})

	req := httptest.NewRequest(http.MethodGet, "/page?lang=ru", nil)
	req.Header.Set("Accept-Language", "de,en;q=0.9")

	if got := n.Resolve(req); got != "ru" {
		t.Fatalf("expected ru, got %s", got)
	}
}

func TestNegotiatorAcceptLanguageFallback(t *testing.T) {
	n := New("en", []string{"en", "ru"})

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.7")

	if got := n.Resolve(req); got != "ru" {
		t.Fatalf("expected ru, got %s", got)
	}
}

func TestNegotiatorDefault(t *testing.T) {
	n := New("ru", []string{"ru", "en"})

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	if got := n.Resolve(req); got != "ru" {
		t.Fatalf("expected ru, got %s", got)
	}
}

func TestNormalize(t *testing.T) {
	got := Normalize("EN", "  ru ", "RU", "")
	want := []string{"en", "ru"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestBuildLangHref(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		lang         string
		defaultLang  string
		expectedPath string
	}{
		{
			name:         "handles root path",
			path:         "/",
			lang:         "ru",
			defaultLang:  "en",
			expectedPath: "/?lang=ru",
		},
		{
			name:         "replaces lang param",
			path:         "/welcome?foo=1&lang=en",
			lang:         "ru",
			defaultLang:  "en",
			expectedPath: "/welcome?foo=1&lang=ru",
		},
		{
			name:         "keeps lang for default locale",
			path:         "/welcome?foo=1&lang=ru",
			lang:         "en",
			defaultLang:  "en",
			expectedPath: "/welcome?foo=1&lang=en",
		},
		{
			name:         "preserves nested path query",
			path:         "/docs/quickstart?foo=bar",
			lang:         "de",
			defaultLang:  "en",
			expectedPath: "/docs/quickstart?foo=bar&lang=de",
		},
		{
			name:         "returns root for nil request",
			path:         "",
			lang:         "ru",
			defaultLang:  "en",
			expectedPath: "/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var req *http.Request
			if test.path != "" {
				req = httptest.NewRequest(http.MethodGet, test.path, nil)
			}
			got := BuildLangHref(req, test.lang, test.defaultLang)
			if got != test.expectedPath {
				t.Fatalf("got %q want %q", got, test.expectedPath)
			}
		})
	}
}

func TestWithLangQueryBackwardCompat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/welcome?lang=en", nil)
	if got := WithLangQuery(req, "ru", "en"); got != BuildLangHref(req, "ru", "en") {
		t.Fatalf("got %q want alias result", got)
	}
}
