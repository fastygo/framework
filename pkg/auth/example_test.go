package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fastygo/framework/pkg/auth"
)

// ExampleCookieSession demonstrates the round-trip lifecycle of an
// HMAC-signed cookie session: Issue stores the typed payload in a
// Set-Cookie header, the client returns the cookie on the next
// request, and Read recovers the original value.
//
// In production:
//   - Set Secret from the SESSION_KEY env var (>=32 random bytes).
//   - Set Secure: true once the site is fully HTTPS.
//   - Pick TTL based on your idle-session policy (8h is a common UX choice).
func ExampleCookieSession() {
	type sessionData struct {
		UserID string
		Email  string
	}

	session := auth.CookieSession[sessionData]{
		Name:     "sid",
		Path:     "/",
		Secret:   "test-secret-do-not-use-in-production",
		TTL:      time.Hour,
		Secure:   false, // set true in production (HTTPS only)
		SameSite: http.SameSiteLaxMode,
		HTTPOnly: true,
	}

	// 1. Login handler issues the session.
	rec := httptest.NewRecorder()
	if err := session.Issue(rec, sessionData{UserID: "42", Email: "ada@example.com"}); err != nil {
		fmt.Println("issue error:", err)
		return
	}

	cookieHeader := rec.Header().Get("Set-Cookie")
	fmt.Println("Set-Cookie present:", cookieHeader != "")

	// 2. Next request from the client carries the cookie back.
	follow := httptest.NewRequest(http.MethodGet, "/profile", nil)
	for _, c := range rec.Result().Cookies() {
		follow.AddCookie(c)
	}

	got, ok := session.Read(follow)
	fmt.Println("read ok:", ok)
	fmt.Println("user id:", got.UserID)
	fmt.Println("email:", got.Email)

	// Output:
	// Set-Cookie present: true
	// read ok: true
	// user id: 42
	// email: ada@example.com
}

// ExampleCookieSession_tampered shows the security signal Read emits
// when the cookie is forged or modified after issue: the call returns
// (zero, false) and a "session_tampered" audit event is recorded so
// SIEM alerting can pick it up.
func ExampleCookieSession_tampered() {
	type token struct{ Sub string }

	session := auth.CookieSession[token]{
		Name:   "sid",
		Path:   "/",
		Secret: "test-secret",
		TTL:    time.Hour,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "totally-not-a-real-signature"})

	_, ok := session.Read(req)
	fmt.Println("tampered cookie accepted:", ok)

	// Output:
	// tampered cookie accepted: false
}
