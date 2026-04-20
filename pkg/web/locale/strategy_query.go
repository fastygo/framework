package locale

import (
	"net/http"
	"strings"
)

// QueryStrategy resolves locale from query parameters and falls back to
// Accept-Language and the configured default.
type QueryStrategy struct {
	Param    string
	Aliases  []string
	Available []string
	Default  string
	ValueMap map[string]string
}

// Resolve reads the locale from request query parameters when present.
// If no explicit query strategy is found, it falls back to Accept-Language and default.
func (s *QueryStrategy) Resolve(r *http.Request) (string, *http.Request) {
	locale, rewritten, _ := s.resolveExplicit(r)
	return locale, rewritten
}

func (s *QueryStrategy) resolveExplicit(r *http.Request) (string, *http.Request, bool) {
	if r == nil || r.URL == nil {
		return s.defaultLocale(), r, false
	}

	if candidate := queryCandidate(r, s.paramNames()); candidate != "" {
		locale := s.matchLocale(candidate)
		if locale != "" {
			return locale, r, true
		}
	}

	for _, candidate := range parseAcceptLanguage(r.Header.Get("Accept-Language")) {
		locale := s.matchLocale(candidate)
		if locale != "" {
			return locale, r, false
		}
	}

	return s.defaultLocale(), r, false
}

// Href builds a URL using the strategy's query param with the locale value applied.
func (s *QueryStrategy) Href(r *http.Request, lang string) string {
	param := s.param()
	locale := s.matchLocale(lang)
	if r == nil || r.URL == nil {
		return "/"
	}

	u := *r.URL
	q := u.Query()
	if locale != "" {
		q.Set(param, locale)
	} else {
		q.Del(param)
	}
	u.RawQuery = q.Encode()

	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery == "" {
		return path
	}
	return path + "?" + u.RawQuery
}

// Persist is a no-op for the pure-query strategy.
func (s *QueryStrategy) Persist(_ http.ResponseWriter, _ string) {}

func (s *QueryStrategy) availableLocales() []string {
	return Normalize(s.Available...)
}

func (s *QueryStrategy) defaultLocale() string {
	if s.Default != "" {
		if locale := s.matchLocale(s.Default); locale != "" {
			return locale
		}
	}

	available := s.availableLocales()
	if len(available) > 0 {
		return available[0]
	}

	return "en"
}

func (s *QueryStrategy) matchLocale(raw string) string {
	return matchLocale(raw, s.ValueMap, s.availableLocales())
}

func (s *QueryStrategy) param() string {
	if strings.TrimSpace(s.Param) == "" {
		return "lang"
	}
	return strings.TrimSpace(s.Param)
}

func (s *QueryStrategy) paramNames() []string {
	names := make([]string, 0, 1+len(s.Aliases))
	param := s.param()
	names = append(names, param)
	for _, alias := range s.Aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" || alias == param {
			continue
		}
		names = append(names, alias)
	}
	return names
}

func queryCandidate(r *http.Request, names []string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	q := r.URL.Query()
	for _, name := range names {
		candidate := strings.TrimSpace(q.Get(name))
		if candidate == "" {
			continue
		}
		return candidate
	}
	return ""
}

func matchLocale(raw string, valueMap map[string]string, available []string) string {
	if raw == "" {
		return ""
	}

	normalized := strings.ToLower(strings.TrimSpace(raw))
	if mapped, ok := valueMap[normalized]; ok && mapped != "" {
		normalized = strings.ToLower(strings.TrimSpace(mapped))
	}
	if len(normalized) > 2 {
		normalized = normalized[:2]
	}

	if len(available) == 0 {
		return normalized
	}
	for _, locale := range available {
		if locale == normalized {
			return normalized
		}
	}
	return ""
}
