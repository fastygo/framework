package locale

import (
	"net/http"
	"strings"
)

// PathPrefixStrategy resolves locale from the first URL path segment
// and rewrites the request path by stripping the locale segment for handlers.
type PathPrefixStrategy struct {
	Available      []string
	Default        string
	RedirectMissing bool
}

// Resolve reads the first path segment. If it is an available locale,
// that locale is used and removed from the route path.
func (s *PathPrefixStrategy) Resolve(r *http.Request) (string, *http.Request) {
	locale, rewritten, _ := s.resolveExplicit(r)
	return locale, rewritten
}

func (s *PathPrefixStrategy) resolveExplicit(r *http.Request) (string, *http.Request, bool) {
	if r == nil || r.URL == nil {
		return s.defaultLocale(), r, false
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	segment, remainder := splitPath(path)
	if segment != "" {
		if locale := s.matchLocale(segment); locale != "" {
			rewritten := cloneRequest(r)
			if remainder == "" {
				rewritten.URL.Path = "/"
			} else {
				rewritten.URL.Path = remainder
			}
			return locale, rewritten, true
		}
	}

	return s.resolveWithFallback(r), cloneRequest(r), false
}

// Href builds a locale-aware URL by prefixing the target locale.
func (s *PathPrefixStrategy) Href(r *http.Request, lang string) string {
	locale := s.matchLocale(lang)
	if locale == "" {
		return ""
	}

	path := "/"
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.EscapedPath()
		if path == "" {
			path = "/"
		}
		path = stripKnownLocalePrefix(path, s.availableLocales())
	}

	target := "/" + locale
	if path != "/" {
		target += "/" + strings.TrimPrefix(path, "/")
	}

	if r != nil && r.URL != nil && r.URL.RawQuery != "" {
		query = "?" + r.URL.RawQuery
	}

	return target + query
}

// Persist is no-op for path-strategy unless wrapped by decorators.
func (s *PathPrefixStrategy) Persist(_ http.ResponseWriter, _ string) {}

func (s *PathPrefixStrategy) availableLocales() []string {
	return Normalize(s.Available...)
}

func (s *PathPrefixStrategy) defaultLocale() string {
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

func (s *PathPrefixStrategy) resolveWithFallback(r *http.Request) string {
	for _, candidate := range parseAcceptLanguage(r.Header.Get("Accept-Language")) {
		if locale := s.matchLocale(candidate); locale != "" {
			return locale
		}
	}
	return s.defaultLocale()
}

func (s *PathPrefixStrategy) matchLocale(raw string) string {
	return matchLocale(raw, nil, s.availableLocales())
}

func splitPath(path string) (segment string, remainder string) {
	if path == "" {
		return "", ""
	}
	head, tail, found := strings.Cut(path, "/")
	if !found {
		return head, ""
	}
	if tail == "" {
		return head, ""
	}
	if !strings.HasPrefix(tail, "/") {
		return head, "/" + tail
	}
	return head, strings.TrimPrefix(tail, "/")
}

func stripKnownLocalePrefix(path string, locales []string) string {
	if path == "" || path == "/" {
		return "/"
	}

	segment, rest := splitPath(strings.TrimPrefix(path, "/"))
	normalized := matchLocale(segment, nil, locales)
	if normalized == "" {
		return "/" + strings.TrimPrefix(path, "/")
	}

	if rest == "" {
		return "/"
	}
	return rest
}

func cloneRequest(r *http.Request) *http.Request {
	if r == nil {
		return nil
	}
	return r.Clone(r.Context())
}
