package app

import (
	"time"

	"github.com/fastygo/framework/pkg/web/locale"
)

// LocalesConfig wires locale negotiation and persistence for an application.
type LocalesConfig struct {
	Default   string
	Available []string

	// Strategy is a custom locale strategy. If nil, QueryStrategy is used.
	Strategy locale.LocaleStrategy

	// Cookie enables locale persistence and custom cookie settings.
	Cookie locale.CookieOptions

	// SPA is propagated as metadata for features that want to opt-in.
	SPA bool
}

// WithLocales installs locale strategy middleware at the front of the request chain.
// It applies defaults from the builder Config when cfg fields are empty.
func (b *AppBuilder) WithLocales(cfg LocalesConfig) *AppBuilder {
	if b == nil {
		return nil
	}

	cfg.Default = normalizeLocale(cfg.Default, b.cfg.DefaultLocale)
	cfg.Available = normalizeLocales(cfg.Available, b.cfg.AvailableLocales)

	strategy := cfg.Strategy
	if strategy == nil {
		strategy = &locale.QueryStrategy{
			Default:   cfg.Default,
			Available: cfg.Available,
		}
	}

	if cfg.Cookie.Enabled {
		strategy = &locale.CookieDecorator{
			Inner:    strategy,
			Name:     cookieName(cfg.Cookie),
			MaxAge:   cookieMaxAge(cfg.Cookie),
			Path:     cookiePath(cfg.Cookie),
			Secure:   cfg.Cookie.Secure,
			HttpOnly: cfg.Cookie.HttpOnly,
			SameSite: cfg.Cookie.SameSite,
		}
	}

	b.locale = strategy
	b.localeSPAMode = cfg.SPA
	return b
}

func normalizeLocale(explicit, fallback string) string {
	if explicit == "" {
		return fallback
	}
	return explicit
}

func normalizeLocales(explicit, fallback []string) []string {
	if len(explicit) == 0 {
		return locale.Normalize(fallback...)
	}
	return locale.Normalize(explicit...)
}

func cookieName(options locale.CookieOptions) string {
	if options.Name == "" {
		return "lang"
	}
	return options.Name
}

func cookieMaxAge(options locale.CookieOptions) time.Duration {
	if options.MaxAge > 0 {
		return options.MaxAge
	}
	return 30 * 24 * time.Hour
}

func cookiePath(options locale.CookieOptions) string {
	if options.Path == "" {
		return "/"
	}
	return options.Path
}
