package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/web/locale"
)

func TestWithLocales_NilBuilderReturnsNil(t *testing.T) {
	t.Parallel()
	var b *AppBuilder
	if got := b.WithLocales(LocalesConfig{}); got != nil {
		t.Fatalf("WithLocales on nil receiver must return nil, got %#v", got)
	}
}

func TestWithLocales_DefaultsFromConfigBuildQueryStrategy(t *testing.T) {
	t.Parallel()
	b := New(Config{
		AppBind:          "127.0.0.1:0",
		DefaultLocale:    "en",
		AvailableLocales: []string{"en", "ru"},
	})
	got := b.WithLocales(LocalesConfig{})
	if got != b {
		t.Fatalf("WithLocales must return the same builder for chaining")
	}
	strat, ok := b.LocaleStrategy().(*locale.QueryStrategy)
	if !ok {
		t.Fatalf("expected *locale.QueryStrategy, got %T", b.LocaleStrategy())
	}
	if strat.Default != "en" {
		t.Fatalf("default locale: got %q want %q", strat.Default, "en")
	}
	if len(strat.Available) != 2 || strat.Available[0] != "en" || strat.Available[1] != "ru" {
		t.Fatalf("available locales: got %#v", strat.Available)
	}
	if b.LocaleSPAMode() {
		t.Fatalf("expected SPA mode disabled by default")
	}
}

func TestWithLocales_RespectsExplicitOverrides(t *testing.T) {
	t.Parallel()
	b := New(Config{AppBind: "127.0.0.1:0"})
	custom := &locale.QueryStrategy{Default: "de", Available: []string{"de"}}

	b.WithLocales(LocalesConfig{
		Default:   "ru",
		Available: []string{"ru", "EN", " ru "},
		Strategy:  custom,
		SPA:       true,
	})

	if b.LocaleStrategy() != custom {
		t.Fatalf("custom strategy must be used as-is")
	}
	if !b.LocaleSPAMode() {
		t.Fatalf("expected SPA mode enabled")
	}
}

func TestWithLocales_CookieDecoratorWrapsStrategy(t *testing.T) {
	t.Parallel()
	b := New(Config{
		AppBind:          "127.0.0.1:0",
		DefaultLocale:    "en",
		AvailableLocales: []string{"en", "ru"},
	})
	b.WithLocales(LocalesConfig{
		Cookie: locale.CookieOptions{
			Enabled:  true,
			Name:     "lc",
			MaxAge:   2 * time.Hour,
			Path:     "/app",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	deco, ok := b.LocaleStrategy().(*locale.CookieDecorator)
	if !ok {
		t.Fatalf("expected *locale.CookieDecorator, got %T", b.LocaleStrategy())
	}
	if deco.Name != "lc" || deco.MaxAge != 2*time.Hour || deco.Path != "/app" {
		t.Fatalf("cookie options not propagated: %+v", deco)
	}
	if !deco.Secure || !deco.HttpOnly || deco.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security flags not propagated: %+v", deco)
	}
	if _, ok := deco.Inner.(*locale.QueryStrategy); !ok {
		t.Fatalf("inner strategy should default to QueryStrategy, got %T", deco.Inner)
	}
}

func TestWithLocales_CookieDefaultsAppliedForZeroValues(t *testing.T) {
	t.Parallel()
	b := New(Config{
		AppBind:          "127.0.0.1:0",
		DefaultLocale:    "en",
		AvailableLocales: []string{"en"},
	})
	b.WithLocales(LocalesConfig{
		Cookie: locale.CookieOptions{Enabled: true},
	})
	deco := b.LocaleStrategy().(*locale.CookieDecorator)
	if deco.Name != "lang" {
		t.Fatalf("cookie name default: got %q", deco.Name)
	}
	if deco.Path != "/" {
		t.Fatalf("cookie path default: got %q", deco.Path)
	}
	if deco.MaxAge != 30*24*time.Hour {
		t.Fatalf("cookie max age default: got %s", deco.MaxAge)
	}
}

func TestNormalizeLocaleHelpers(t *testing.T) {
	t.Parallel()
	if got := normalizeLocale("", "ru"); got != "ru" {
		t.Fatalf("normalizeLocale empty: got %q", got)
	}
	if got := normalizeLocale("de", "ru"); got != "de" {
		t.Fatalf("normalizeLocale explicit: got %q", got)
	}

	got := normalizeLocales(nil, []string{"EN", "ru"})
	if len(got) != 2 || got[0] != "en" || got[1] != "ru" {
		t.Fatalf("normalizeLocales fallback: got %#v", got)
	}

	got = normalizeLocales([]string{"De", "en"}, []string{"ru"})
	if len(got) != 2 || got[0] != "de" || got[1] != "en" {
		t.Fatalf("normalizeLocales explicit: got %#v", got)
	}
}

func TestCookieHelpers(t *testing.T) {
	t.Parallel()
	if got := cookieName(locale.CookieOptions{}); got != "lang" {
		t.Fatalf("cookieName default: got %q", got)
	}
	if got := cookieName(locale.CookieOptions{Name: "x"}); got != "x" {
		t.Fatalf("cookieName explicit: got %q", got)
	}

	if got := cookieMaxAge(locale.CookieOptions{}); got != 30*24*time.Hour {
		t.Fatalf("cookieMaxAge default: got %s", got)
	}
	if got := cookieMaxAge(locale.CookieOptions{MaxAge: time.Minute}); got != time.Minute {
		t.Fatalf("cookieMaxAge explicit: got %s", got)
	}

	if got := cookiePath(locale.CookieOptions{}); got != "/" {
		t.Fatalf("cookiePath default: got %q", got)
	}
	if got := cookiePath(locale.CookieOptions{Path: "/api"}); got != "/api" {
		t.Fatalf("cookiePath explicit: got %q", got)
	}
}
