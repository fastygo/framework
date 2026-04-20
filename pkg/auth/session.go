// Package auth implements small, dependency-free authentication building
// blocks: HMAC-signed cookie sessions and an OpenID Connect client suitable
// for SSO logins.
//
// Both pieces are independent and can be wired into any feature; the
// framework deliberately stays out of the routing decisions (login URLs,
// callback URLs, etc.) so each application can own its UX.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// auditEvent emits a structured slog event under the "auth" group.
// Audit fields use stable keys so log pipelines (Loki, ELK) can route
// security events to a dedicated index without parsing message text.
//
// The event level is Info for happy-path actions (issue, clear) and
// Warn for tamper-detection / decode failures so SIEM rules can alert
// on Warn-and-above only.
func auditEvent(ctx context.Context, level slog.Level, event string, attrs ...slog.Attr) {
	logger := slog.Default()
	all := make([]slog.Attr, 0, len(attrs)+1)
	all = append(all, slog.String("event", event))
	all = append(all, attrs...)
	logger.LogAttrs(ctx, level, "auth.audit", all...)
}

// CookieSession describes a strongly-typed session payload persisted into
// an HMAC-signed HTTP cookie. The generic parameter T must be JSON-serialisable.
type CookieSession[T any] struct {
	// Name is the cookie name (e.g. "sid"). Required.
	Name string
	// Path scopes the cookie to a URL prefix (typically "/").
	Path string
	// Secret is the HMAC-SHA256 signing key. At least 32 random bytes
	// recommended. Treat as PII; rotating it invalidates every active
	// session (subsequent Read calls log session_tampered).
	Secret string
	// TTL is the cookie lifetime; expired cookies are rejected by Read.
	TTL time.Duration
	// Secure restricts the cookie to HTTPS connections. Set to true
	// in production.
	Secure bool
	// SameSite controls cross-site cookie attachment. http.SameSiteLaxMode
	// is the recommended default for SSR apps.
	SameSite http.SameSite
	// HTTPOnly hides the cookie from JavaScript (document.cookie).
	// Strongly recommended unless a client-side library specifically
	// needs to read the session cookie.
	HTTPOnly bool
}

// envelope wraps the user-provided payload with an expiry timestamp.
type envelope[T any] struct {
	Data T     `json:"d"`
	Exp  int64 `json:"exp"`
}

// Issue stores value into a Set-Cookie header on w. A successful Issue
// emits an "auth.audit" slog event with event=session_issued; failures
// emit event=session_issue_failed at Warn level.
func (s CookieSession[T]) Issue(w http.ResponseWriter, value T) error {
	if s.Secret == "" {
		auditEvent(context.Background(), slog.LevelWarn, "session_issue_failed",
			slog.String("cookie", s.cookieName()),
			slog.String("reason", "missing_secret"),
		)
		return fmt.Errorf("auth: cookie session requires Secret")
	}
	if s.TTL <= 0 {
		auditEvent(context.Background(), slog.LevelWarn, "session_issue_failed",
			slog.String("cookie", s.cookieName()),
			slog.String("reason", "non_positive_ttl"),
		)
		return fmt.Errorf("auth: cookie session requires positive TTL")
	}
	expires := time.Now().Add(s.TTL).Unix()
	signed, err := SignedEncode(envelope[T]{Data: value, Exp: expires}, s.Secret)
	if err != nil {
		auditEvent(context.Background(), slog.LevelWarn, "session_issue_failed",
			slog.String("cookie", s.cookieName()),
			slog.String("reason", "encode_error"),
			slog.String("error", err.Error()),
		)
		return err
	}

	cookie := &http.Cookie{
		Name:     s.cookieName(),
		Value:    signed,
		Path:     s.cookiePath(),
		HttpOnly: s.HTTPOnly,
		Secure:   s.Secure,
		SameSite: s.sameSite(),
		MaxAge:   int(s.TTL.Seconds()),
	}
	http.SetCookie(w, cookie)

	auditEvent(context.Background(), slog.LevelInfo, "session_issued",
		slog.String("cookie", s.cookieName()),
		slog.Int64("exp", expires),
	)
	return nil
}

// Read returns the typed session value if a valid, non-expired cookie is
// found. Tamper-detection and expiry failures are silently surfaced as
// (zero, false) to keep request handlers simple, but each failure emits
// an "auth.audit" event so security teams can detect attacks.
//
// Audit events:
//   - session_decoded   (Info)  — successful read
//   - session_missing   (Debug) — no cookie present (anonymous request)
//   - session_tampered  (Warn)  — HMAC mismatch or malformed envelope
//   - session_expired   (Info)  — valid signature but past Exp
func (s CookieSession[T]) Read(r *http.Request) (T, bool) {
	var zero T
	ctx := r.Context()
	name := s.cookieName()

	cookie, err := r.Cookie(name)
	if err != nil {
		auditEvent(ctx, slog.LevelDebug, "session_missing",
			slog.String("cookie", name),
		)
		return zero, false
	}
	var env envelope[T]
	if err := SignedDecode(cookie.Value, s.Secret, &env); err != nil {
		auditEvent(ctx, slog.LevelWarn, "session_tampered",
			slog.String("cookie", name),
			slog.String("error", err.Error()),
			slog.String("remote", r.RemoteAddr),
		)
		return zero, false
	}
	if env.Exp > 0 && time.Now().Unix() > env.Exp {
		auditEvent(ctx, slog.LevelInfo, "session_expired",
			slog.String("cookie", name),
			slog.Int64("exp", env.Exp),
		)
		return zero, false
	}
	auditEvent(ctx, slog.LevelDebug, "session_decoded",
		slog.String("cookie", name),
	)
	return env.Data, true
}

// Clear removes the cookie from the client by overwriting with an expired one.
// Emits an "auth.audit" event with event=session_cleared.
func (s CookieSession[T]) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName(),
		Value:    "",
		Path:     s.cookiePath(),
		HttpOnly: s.HTTPOnly,
		Secure:   s.Secure,
		SameSite: s.sameSite(),
		MaxAge:   -1,
	})
	auditEvent(context.Background(), slog.LevelInfo, "session_cleared",
		slog.String("cookie", s.cookieName()),
	)
}

func (s CookieSession[T]) cookieName() string {
	if s.Name == "" {
		return "session"
	}
	return s.Name
}

func (s CookieSession[T]) cookiePath() string {
	if s.Path == "" {
		return "/"
	}
	return s.Path
}

func (s CookieSession[T]) sameSite() http.SameSite {
	if s.SameSite == 0 {
		return http.SameSiteLaxMode
	}
	return s.SameSite
}

// RandomToken returns a hex-encoded cryptographically random token of the
// requested byte length. Useful for OAuth state parameters.
func RandomToken(byteLen int) string {
	if byteLen <= 0 {
		byteLen = 16
	}
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SignedEncode marshals data, base64-encodes it, then appends an HMAC suffix.
func SignedEncode(data any, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("auth: missing secret")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := computeHMAC(encoded, secret)
	return encoded + "." + mac, nil
}

// SignedDecode validates the HMAC suffix and unmarshals the payload into dst.
func SignedDecode(value, secret string, dst any) error {
	if secret == "" {
		return fmt.Errorf("auth: missing secret")
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("auth: invalid signed value")
	}
	expected := computeHMAC(parts[0], secret)
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return fmt.Errorf("auth: invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dst)
}

func computeHMAC(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
