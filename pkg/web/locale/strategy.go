package locale

import "net/http"

// LocaleStrategy defines request locale resolution and locale-aware URL building.
type LocaleStrategy interface {
	Resolve(r *http.Request) (string, *http.Request)
	Href(r *http.Request, lang string) string
	Persist(w http.ResponseWriter, lang string)
}

// explicitLocaleResolver is implemented by built-in strategies that can identify whether
// locale resolution used an explicit strategy hint from the request.
type explicitLocaleResolver interface {
	resolveExplicit(*http.Request) (string, *http.Request, bool)
}

// localeCandidateMatcher normalizes and validates locale candidates for a strategy.
type localeCandidateMatcher interface {
	matchLocale(string) string
}

func resolveLocaleWithHint(strategy LocaleStrategy, r *http.Request) (string, *http.Request, bool) {
	if strategy == nil {
		return "", r, false
	}
	if explicit, ok := strategy.(explicitLocaleResolver); ok {
		return explicit.resolveExplicit(r)
	}
	locale, rewritten := strategy.Resolve(r)
	return locale, rewritten, false
}
