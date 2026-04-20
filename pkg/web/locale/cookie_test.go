package locale

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieDecoratorResolvesFromCookieWhenNoExplicitHint(t *testing.T) {
	strategy := &CookieDecorator{
		Inner: &QueryStrategy{
			Param:      "lang",
			Available:  []string{"en", "ru"},
			Default:    "en",
			ValueMap:   map[string]string{"russian": "ru"},
		},
		Name: "lang",
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "russian"})

	got, _ := strategy.Resolve(req)
	if got != "ru" {
		t.Fatalf("expected locale from cookie to be ru, got %q", got)
	}
}

func TestCookieDecoratorPersistStoresNormalizedLocale(t *testing.T) {
	strategy := &CookieDecorator{
		Inner: &QueryStrategy{
			Available: []string{"en", "ru"},
			Default:   "en",
		},
		Name: "lang",
	}

	recorder := httptest.NewRecorder()
	strategy.Persist(recorder, "ru")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "lang" || cookies[0].Value != "ru" {
		t.Fatalf("expected lang=ru cookie, got %s=%s", cookies[0].Name, cookies[0].Value)
	}
}
