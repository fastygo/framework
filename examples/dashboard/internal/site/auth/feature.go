// Package auth implements a tiny demo login flow using HMAC-signed cookie
// sessions. It accepts any non-empty email/password combination — replace
// the verification step with your real identity store before shipping.
package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/auth"
	"github.com/fastygo/framework/pkg/web"
	"github.com/fastygo/framework/pkg/web/locale"

	dashboardi18n "github.com/fastygo/framework/examples/dashboard/internal/site/i18n"
	"github.com/fastygo/framework/examples/dashboard/internal/site/views"
)

// Session is the value persisted into the cookie session.
type Session struct {
	Email string `json:"email"`
}

// Feature exposes /auth/login and /auth/logout, plus a Middleware to
// enforce authentication on protected routes.
type Feature struct {
	cookie auth.CookieSession[Session]
}

// New constructs an Auth feature.
//
// secret must be a long random string — see `openssl rand -base64 32`.
func New(secret string) *Feature {
	return &Feature{
		cookie: auth.CookieSession[Session]{
			Name:     "dashboard_session",
			Path:     "/",
			Secret:   secret,
			TTL:      8 * time.Hour,
			HTTPOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		},
	}
}

func (f *Feature) ID() string              { return "auth" }
func (f *Feature) NavItems() []app.NavItem { return nil }

func (f *Feature) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", f.handleLoginPage)
	mux.HandleFunc("POST /auth/login", f.handleLoginSubmit)
	mux.HandleFunc("POST /auth/logout", f.handleLogout)
}

// Middleware redirects unauthenticated requests to /auth/login.
func (f *Feature) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := f.cookie.Read(r); !ok {
			http.Redirect(w, r, "/auth/login?return_to="+r.URL.RequestURI(), http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CurrentUser returns the email of the logged-in user (empty when none).
func (f *Feature) CurrentUser(r *http.Request) string {
	if sess, ok := f.cookie.Read(r); ok {
		return sess.Email
	}
	return ""
}

func (f *Feature) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	loc := locale.From(r.Context())
	bundle, err := dashboardi18n.Load(loc)
	if err != nil {
		bundle, _ = dashboardi18n.Load("en")
	}
	login := bundle.Dashboard.Login

	errorCode := strings.TrimSpace(r.URL.Query().Get("error"))
	errorText := ""
	switch errorCode {
	case "":
		errorText = ""
	case "session_error":
		errorText = fallbackText(bundle.Dashboard.Login.SessionError, "Unable to sign in. Please try again.")
	case "missing_credentials":
		errorText = fallbackText(bundle.Common.ErrorText.MissingCredentials, "Email and password are required.")
	default:
		errorText = errorCode
	}

	if err := web.Render(r.Context(), w, views.LoginPage(views.LoginPageData{
		Lang:          loc,
		Title:         fallbackText(login.PageTitle, "Sign in"),
		ReturnTo:      r.URL.Query().Get("return_to"),
		ErrorText:     errorText,
		Subtitle:      fallbackText(login.Subtitle, "Use any email and password — this is a demo stub."),
		EmailLabel:    fallbackText(login.EmailLabel, "Email"),
		PasswordLabel: fallbackText(login.PasswordLabel, "Password"),
		SubmitText:    fallbackText(login.SubmitButtonText, "Sign in"),
	})); err != nil {
		web.HandleError(w, err)
	}
}

func (f *Feature) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data.", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.PostForm.Get("email"))
	password := r.PostForm.Get("password")
	returnTo := r.PostForm.Get("return_to")

	if email == "" || password == "" {
		http.Redirect(w, r, "/auth/login?error=missing_credentials", http.StatusFound)
		return
	}

	if err := f.cookie.Issue(w, Session{Email: email}); err != nil {
		http.Redirect(w, r, "/auth/login?error=session_error", http.StatusFound)
		return
	}

	if returnTo == "" || !strings.HasPrefix(returnTo, "/") {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (f *Feature) handleLogout(w http.ResponseWriter, r *http.Request) {
	f.cookie.Clear(w)
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func fallbackText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
