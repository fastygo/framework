package cab

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/fastygo/framework/internal/site/web/views"
	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web"
)

type Module struct {
	oidc       *OIDCClient
	sessionKey string
}

func New(cfg app.Config) *Module {
	return &Module{
		oidc:       NewOIDCClient(cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret, cfg.OIDCRedirectURI),
		sessionKey: cfg.SessionKey,
	}
}

func (m *Module) ID() string { return "cab" }

func (m *Module) NavItems() []app.NavItem { return nil }

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /cab/", m.handleCab)
	mux.HandleFunc("GET /auth/login", m.handleLogin)
	mux.HandleFunc("GET /auth/callback", m.handleCallback)
	mux.HandleFunc("GET /auth/logout", m.handleLogout)
}

func (m *Module) handleCab(w http.ResponseWriter, r *http.Request) {
	sess := getSiteSession(r, m.sessionKey)
	if sess == nil {
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

func (m *Module) handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	saveOAuthState(w, state, m.sessionKey)

	authURL, err := m.oidc.AuthorizationURL(state, "")
	if err != nil {
		slog.Error("build authorization URL", "error", err)
		http.Error(w, "SSO configuration error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusFound)
}

func (m *Module) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	savedState := getOAuthState(r, m.sessionKey)
	if savedState == "" || savedState != state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	clearOAuthState(w)

	tokenResp, err := m.oidc.ExchangeCode(code)
	if err != nil {
		slog.Error("token exchange", "error", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	claims, err := m.oidc.VerifyIDToken(tokenResp.IDToken)
	if err != nil {
		slog.Error("id_token verification", "error", err)
		http.Error(w, "Token verification failed", http.StatusUnauthorized)
		return
	}

	createSiteSession(w, claims.Email, m.sessionKey)
	http.Redirect(w, r, "/cab/", http.StatusFound)
}

func (m *Module) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSiteSession(w)
	clearOAuthState(w)

	returnTo := "/"
	if origin := m.oidc.AppOrigin(); origin != "" {
		returnTo = origin + "/"
	}

	http.Redirect(w, r, m.oidc.LogoutURL(returnTo), http.StatusFound)
}
