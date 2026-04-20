// Package cab implements a small cabinet feature that demonstrates how to
// plug an OpenID Connect SSO flow on top of pkg/auth.
package cab

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/auth"
	"github.com/fastygo/framework/pkg/web"

	"github.com/fastygo/framework/examples/web/internal/site/views"
)

// SiteSession is the value persisted into the cookie session.
type SiteSession struct {
	Email string `json:"email"`
}

type oauthState struct {
	State string `json:"state"`
	Exp   int64  `json:"exp"`
}

// Feature exposes the cabinet routes.
type Feature struct {
	oidc        *auth.OIDCClient
	sessionKey  string
	session     auth.CookieSession[SiteSession]
	stateCookie auth.CookieSession[oauthState]
}

// New constructs a cab feature from the resolved application configuration.
func New(cfg app.Config) *Feature {
	common := commonCookie(cfg.SessionKey)
	return &Feature{
		oidc: auth.NewOIDCClient(auth.OIDCConfig{
			Issuer:       cfg.OIDCIssuer,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURI:  cfg.OIDCRedirectURI,
		}),
		sessionKey:  cfg.SessionKey,
		session:     applySession(common, "fastygo_session", 8*time.Hour),
		stateCookie: applyState(common, "fastygo_oauth_state", 10*time.Minute),
	}
}

func commonCookie(secret string) auth.CookieSession[SiteSession] {
	return auth.CookieSession[SiteSession]{
		Secret:   secret,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func applySession(base auth.CookieSession[SiteSession], name string, ttl time.Duration) auth.CookieSession[SiteSession] {
	base.Name = name
	base.TTL = ttl
	return base
}

func applyState(common auth.CookieSession[SiteSession], name string, ttl time.Duration) auth.CookieSession[oauthState] {
	return auth.CookieSession[oauthState]{
		Name:     name,
		Secret:   common.Secret,
		Path:     common.Path,
		TTL:      ttl,
		HTTPOnly: common.HTTPOnly,
		Secure:   common.Secure,
		SameSite: common.SameSite,
	}
}

func (f *Feature) ID() string                  { return "cab" }
func (f *Feature) NavItems() []app.NavItem     { return nil }

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /cab/", f.handleCab)
	mux.HandleFunc("GET /auth/login", f.handleLogin)
	mux.HandleFunc("GET /auth/callback", f.handleCallback)
	mux.HandleFunc("GET /auth/logout", f.handleLogout)
}

func (f *Feature) handleCab(w http.ResponseWriter, r *http.Request) {
	sess, ok := f.session.Read(r)
	if !ok {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	layout := views.LayoutData{
		Title:     "Cabinet",
		Locale:    "en",
		Active:    "/cab/",
		BrandName: "FastyGo",
	}

	if err := web.Render(r.Context(), w, views.Layout(layout, templ.NopComponent, views.CabPage(sess.Email))); err != nil {
		slog.Error("render cab page", "error", err)
	}
}

func (f *Feature) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := auth.RandomToken(16)
	if err := f.stateCookie.Issue(w, oauthState{State: state, Exp: time.Now().Add(10 * time.Minute).Unix()}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	authURL, err := f.oidc.AuthorizationURL(state, "")
	if err != nil {
		slog.Error("build authorization URL", "error", err)
		http.Error(w, "SSO configuration error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (f *Feature) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}
	saved, ok := f.stateCookie.Read(r)
	if !ok || saved.State != state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	f.stateCookie.Clear(w)

	tokenResp, err := f.oidc.ExchangeCode(code)
	if err != nil {
		slog.Error("token exchange", "error", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}
	claims, err := f.oidc.VerifyIDToken(tokenResp.IDToken)
	if err != nil {
		slog.Error("id_token verification", "error", err)
		http.Error(w, "Token verification failed", http.StatusUnauthorized)
		return
	}
	if err := f.session.Issue(w, SiteSession{Email: claims.Email}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cab/", http.StatusFound)
}

func (f *Feature) handleLogout(w http.ResponseWriter, r *http.Request) {
	f.session.Clear(w)
	f.stateCookie.Clear(w)

	returnTo := "/"
	if origin := f.oidc.AppOrigin(); origin != "" {
		returnTo = origin + "/"
	}
	http.Redirect(w, r, f.oidc.LogoutURL(returnTo), http.StatusFound)
}
