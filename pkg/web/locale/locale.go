// Package locale exposes small helpers to negotiate request locales and
// keep them normalized across an application. It has no dependency on any
// HTTP framework other than the standard library.
package locale

import (
	"net/http"
	"strings"
)

// Negotiator resolves the active locale for an incoming HTTP request.
//
// The default implementation looks at the `?lang=` query parameter, falls
// back to the first matching value of the Accept-Language header, and
// finally to the configured default locale. All locales are matched and
// returned in lower-case, two-letter form (e.g. "en", "ru").
type Negotiator struct {
	// Default is the locale returned when no negotiation source matches.
	Default string
	// Available is the closed set of locales the application supports;
	// negotiation never returns anything outside this list.
	Available []string
	// QueryParam is the URL query parameter inspected first. Defaults to "lang".
	QueryParam string
	// AcceptLang controls whether the Accept-Language header is consulted.
	AcceptLang bool
	// CookieName, when non-empty, is checked before Accept-Language.
	CookieName string
}

// New returns a Negotiator with sensible defaults for the supplied locales.
func New(defaultLocale string, available []string) *Negotiator {
	defaults := Normalize(defaultLocale)
	def := ""
	if len(defaults) > 0 {
		def = defaults[0]
	}
	return &Negotiator{
		Default:    def,
		Available:  Normalize(available...),
		QueryParam: "lang",
		AcceptLang: true,
	}
}

// Resolve returns the most appropriate locale for the request.
func (n *Negotiator) Resolve(r *http.Request) string {
	if r != nil && n.QueryParam != "" {
		if candidate := truncate(r.URL.Query().Get(n.QueryParam)); candidate != "" {
			if locale := n.match(candidate); locale != "" {
				return locale
			}
		}
	}

	if r != nil && n.CookieName != "" {
		if cookie, err := r.Cookie(n.CookieName); err == nil {
			if locale := n.match(truncate(cookie.Value)); locale != "" {
				return locale
			}
		}
	}

	if r != nil && n.AcceptLang {
		for _, candidate := range parseAcceptLanguage(r.Header.Get("Accept-Language")) {
			if locale := n.match(candidate); locale != "" {
				return locale
			}
		}
	}

	if n.Default != "" {
		return n.Default
	}
	if len(n.Available) > 0 {
		return n.Available[0]
	}
	return "en"
}

func (n *Negotiator) match(candidate string) string {
	if candidate == "" {
		return ""
	}
	for _, locale := range n.Available {
		if locale == candidate {
			return locale
		}
	}
	return ""
}

// Normalize lowercases and trims locale strings, dropping empties and
// duplicates while preserving order.
func Normalize(values ...string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		normalized := normalizeOne(v)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

// Contains reports whether locale is part of the available list.
func Contains(available []string, locale string) bool {
	target := normalizeOne(locale)
	for _, value := range available {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeOne(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 2 {
		value = value[:2]
	}
	return value
}

func truncate(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > 2 {
		raw = raw[:2]
	}
	return strings.ToLower(raw)
}

// parseAcceptLanguage extracts an ordered list of two-letter language tags
// from a raw Accept-Language header, ignoring quality factors for simplicity
// (the order of appearance is preserved).
func parseAcceptLanguage(header string) []string {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	out := make([]string, 0, len(parts))
	for _, raw := range parts {
		segment := strings.TrimSpace(raw)
		if idx := strings.IndexByte(segment, ';'); idx >= 0 {
			segment = segment[:idx]
		}
		if idx := strings.IndexByte(segment, '-'); idx >= 0 {
			segment = segment[:idx]
		}
		segment = strings.ToLower(strings.TrimSpace(segment))
		if segment == "" || segment == "*" {
			continue
		}
		if len(segment) > 2 {
			segment = segment[:2]
		}
		out = append(out, segment)
	}
	return out
}
