package locale

import (
	"net/http"
	"strings"
	"time"
)

// CookieOptions configures cookie persistence for locale selection.
type CookieOptions struct {
	Enabled  bool
	Name     string
	MaxAge   time.Duration
	Path     string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// CookieDecorator decorates any LocaleStrategy with cookie persistence.
type CookieDecorator struct {
	Inner    LocaleStrategy
	Name     string
	MaxAge   time.Duration
	Path     string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// Resolve first asks the inner strategy to resolve locale.
// If the strategy did not use an explicit hint, a valid locale cookie is used first.
func (d *CookieDecorator) Resolve(r *http.Request) (string, *http.Request) {
	if d == nil || d.Inner == nil {
		return "", r
	}

	locale, rewritten, explicit := resolveLocaleWithHint(d.Inner, r)
	if explicit {
		return locale, rewritten
	}

	candidate := d.cookieCandidate(r)
	if candidate == "" {
		return locale, rewritten
	}

	if matcher, ok := d.Inner.(localeCandidateMatcher); ok {
		if matched := matcher.matchLocale(candidate); matched != "" {
			return matched, rewritten
		}
	}

	return locale, rewritten
}

// Href is delegated to the inner strategy.
func (d *CookieDecorator) Href(r *http.Request, lang string) string {
	return d.Inner.Href(r, lang)
}

// Persist stores locale in a cookie, then delegates.
func (d *CookieDecorator) Persist(w http.ResponseWriter, lang string) {
	if d == nil {
		return
	}
	if d.Inner != nil {
		d.Inner.Persist(w, lang)
	}

	if w == nil || d.Inner == nil {
		return
	}
	name := d.cookieName()
	if name == "" {
		return
	}

	matched := lang
	if matcher, ok := d.Inner.(localeCandidateMatcher); ok {
		matched = matcher.matchLocale(lang)
	} else {
		matched = strings.ToLower(strings.TrimSpace(matched))
		if len(matched) > 2 {
			matched = matched[:2]
		}
	}
	if matched == "" {
		return
	}

	c := &http.Cookie{
		Name:     name,
		Value:    strings.TrimSpace(matched),
		Path:     d.cookiePath(),
		Secure:   d.Secure,
		HttpOnly: d.HttpOnly,
		SameSite: d.SameSiteMode(),
	}

	maxAge := d.MaxAge
	if maxAge > 0 {
		c.MaxAge = int(maxAge.Seconds())
		c.Expires = time.Now().Add(maxAge)
	} else if maxAge < 0 {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}

	http.SetCookie(w, c)
}

func (d *CookieDecorator) cookieCandidate(r *http.Request) string {
	if d == nil || r == nil {
		return ""
	}
	name := d.cookieName()
	if name == "" {
		return ""
	}

	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func (d *CookieDecorator) cookieName() string {
	if d.Name == "" {
		return "lang"
	}
	return d.Name
}

func (d *CookieDecorator) cookiePath() string {
	if d.Path == "" {
		return "/"
	}
	return d.Path
}

func (d *CookieDecorator) SameSiteMode() http.SameSite {
	if d.SameSite == 0 {
		return http.SameSiteLaxMode
	}
	return d.SameSite
}

func (d *CookieDecorator) availableLocales() []string {
	if d == nil || d.Inner == nil {
		return nil
	}
	return availableLocales(d.Inner)
}

func (d *CookieDecorator) defaultLocale() string {
	if d == nil || d.Inner == nil {
		return "en"
	}
	return defaultLocale(d.Inner)
}
