package locale

import "net/http"

// BuildLangHref returns a relative URL using the legacy "lang" query parameter.
func BuildLangHref(r *http.Request, lang string, _ string) string {
	return (&QueryStrategy{}).Href(r, lang)
}

// WithLangQuery keeps the historical name as a compatibility alias.
func WithLangQuery(r *http.Request, lang, defaultLocale string) string {
	return BuildLangHref(r, lang, defaultLocale)
}
