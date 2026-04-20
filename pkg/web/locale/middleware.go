package locale

import (
	"context"
	"net/http"
)

type localeContextKey struct{}
type availableContextKey struct{}
type defaultContextKey struct{}
type strategyContextKey struct{}
type requestContextKey struct{}
type spaModeContextKey struct{}

// Middleware stores locale resolution results into request context and persists locale.
func Middleware(strategy LocaleStrategy) func(http.Handler) http.Handler {
	return MiddlewareWithSPAMode(strategy, false)
}

// MiddlewareWithSPAMode stores locale context and marks whether locale SPA mode
// is enabled for request-side behavior.
func MiddlewareWithSPAMode(strategy LocaleStrategy, spaMode bool) func(http.Handler) http.Handler {
	if strategy == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale, rewrittenRequest, _ := resolveLocaleWithHint(strategy, r)
			if rewrittenRequest == nil {
				rewrittenRequest = r
			}

			strategy.Persist(w, locale)

			ctx := rewrittenRequest.Context()
			ctx = context.WithValue(ctx, localeContextKey{}, locale)
			ctx = context.WithValue(ctx, strategyContextKey{}, strategy)
			ctx = context.WithValue(ctx, requestContextKey{}, rewrittenRequest)
			ctx = context.WithValue(ctx, defaultContextKey{}, defaultLocale(strategy))
			ctx = context.WithValue(ctx, availableContextKey{}, availableLocales(strategy))
			ctx = context.WithValue(ctx, spaModeContextKey{}, spaMode)

			next.ServeHTTP(w, rewrittenRequest.WithContext(ctx))
		})
	}
}

// From returns the active locale from context, or "en" when missing.
func From(ctx context.Context) string {
	if ctx == nil {
		return "en"
	}
	locale, ok := ctx.Value(localeContextKey{}).(string)
	if ok && locale != "" {
		return locale
	}
	return Default(ctx)
}

// HrefFor returns a strategy-specific URL for switching to lang.
func HrefFor(ctx context.Context, lang string) string {
	if ctx == nil {
		return ""
	}
	req, _ := ctx.Value(requestContextKey{}).(*http.Request)
	strategy, _ := ctx.Value(strategyContextKey{}).(LocaleStrategy)
	if strategy != nil && req != nil {
		return strategy.Href(req, lang)
	}
	if req != nil {
		return BuildLangHref(req, lang, Default(ctx))
	}
	return ""
}

// RequestFromContext returns the request pointer attached by Middleware.
func RequestFromContext(ctx context.Context) *http.Request {
	if ctx == nil {
		return nil
	}
	req, _ := ctx.Value(requestContextKey{}).(*http.Request)
	return req
}

// SPAMode reports whether locale SPA mode is enabled in request context.
func SPAMode(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	spaModeValue, ok := ctx.Value(spaModeContextKey{}).(bool)
	return ok && spaModeValue
}

// Available returns the resolved list of locales from context (or nil).
func Available(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	value := ctx.Value(availableContextKey{})
	raw, ok := value.([]string)
	if !ok || len(raw) == 0 {
		return nil
	}
	available := make([]string, len(raw))
	copy(available, raw)
	return available
}

// Default returns the configured default locale from context, or "en".
func Default(ctx context.Context) string {
	if ctx == nil {
		return "en"
	}
	defaultLocaleValue, ok := ctx.Value(defaultContextKey{}).(string)
	if ok && defaultLocaleValue != "" {
		return defaultLocaleValue
	}
	return "en"
}

func defaultLocale(strategy LocaleStrategy) string {
	if defaults, ok := strategy.(interface{ defaultLocale() string }); ok {
		return defaults.defaultLocale()
	}
	return "en"
}

func availableLocales(strategy LocaleStrategy) []string {
	if available, ok := strategy.(interface{ availableLocales() []string }); ok {
		out := available.availableLocales()
		copied := make([]string, len(out))
		copy(copied, out)
		return copied
	}
	return nil
}
