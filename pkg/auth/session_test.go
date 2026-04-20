package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testPayload struct {
	UserID string `json:"u"`
	Email  string `json:"e"`
}

func newTestSession(t *testing.T) CookieSession[testPayload] {
	t.Helper()
	return CookieSession[testPayload]{
		Name:     "sid",
		Path:     "/",
		Secret:   "test-secret-32-bytes-min-........",
		TTL:      time.Hour,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		HTTPOnly: true,
	}
}

func TestCookieSession_RoundTrip(t *testing.T) {
	s := newTestSession(t)

	rec := httptest.NewRecorder()
	want := testPayload{UserID: "42", Email: "ada@example.com"}
	if err := s.Issue(rec, want); err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	got, ok := s.Read(req)
	if !ok {
		t.Fatalf("Read returned ok=false on a freshly issued cookie")
	}
	if got != want {
		t.Fatalf("payload round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestCookieSession_Issue_MissingSecret(t *testing.T) {
	s := newTestSession(t)
	s.Secret = ""

	err := s.Issue(httptest.NewRecorder(), testPayload{})
	if err == nil {
		t.Fatalf("Issue with empty Secret must return an error")
	}
	if !strings.Contains(err.Error(), "Secret") {
		t.Fatalf("error should mention missing Secret, got: %v", err)
	}
}

func TestCookieSession_Issue_NonPositiveTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"zero", 0},
		{"negative", -time.Second},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSession(t)
			s.TTL = tc.ttl
			err := s.Issue(httptest.NewRecorder(), testPayload{})
			if err == nil {
				t.Fatalf("Issue with TTL=%v must return an error", tc.ttl)
			}
			if !strings.Contains(err.Error(), "TTL") {
				t.Fatalf("error should mention TTL, got: %v", err)
			}
		})
	}
}

func TestCookieSession_CookieAttributes(t *testing.T) {
	s := newTestSession(t)

	rec := httptest.NewRecorder()
	if err := s.Issue(rec, testPayload{UserID: "1"}); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly 1 Set-Cookie, got %d", len(cookies))
	}
	c := cookies[0]

	if c.Name != "sid" {
		t.Errorf("Name: got %q, want %q", c.Name, "sid")
	}
	if c.Path != "/" {
		t.Errorf("Path: got %q, want %q", c.Path, "/")
	}
	if !c.HttpOnly {
		t.Error("HttpOnly: got false, want true")
	}
	if !c.Secure {
		t.Error("Secure: got false, want true")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite: got %v, want Lax", c.SameSite)
	}
	wantMaxAge := int(time.Hour.Seconds())
	if c.MaxAge != wantMaxAge {
		t.Errorf("MaxAge: got %d, want %d", c.MaxAge, wantMaxAge)
	}
}

func TestCookieSession_Read_NoCookie(t *testing.T) {
	s := newTestSession(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if _, ok := s.Read(req); ok {
		t.Fatalf("Read must return ok=false when the cookie is absent")
	}
}

func TestCookieSession_Read_TamperedSignature(t *testing.T) {
	s := newTestSession(t)

	rec := httptest.NewRecorder()
	if err := s.Issue(rec, testPayload{UserID: "victim"}); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		// Flip the last byte of the signature half.
		// SignedEncode emits "<payload>.<sig>".
		idx := strings.LastIndexByte(c.Value, '.')
		if idx < 0 || idx == len(c.Value)-1 {
			t.Fatalf("malformed test cookie value: %q", c.Value)
		}
		bytesValue := []byte(c.Value)
		last := bytesValue[len(bytesValue)-1]
		if last == 'A' {
			bytesValue[len(bytesValue)-1] = 'B'
		} else {
			bytesValue[len(bytesValue)-1] = 'A'
		}
		c.Value = string(bytesValue)
		req.AddCookie(c)
	}

	if _, ok := s.Read(req); ok {
		t.Fatalf("Read must reject a cookie with a tampered signature")
	}
}

func TestCookieSession_Read_MalformedEnvelope(t *testing.T) {
	s := newTestSession(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "no-dot-no-signature"})

	if _, ok := s.Read(req); ok {
		t.Fatalf("Read must reject a cookie without the signature separator")
	}
}

func TestCookieSession_Read_Expired(t *testing.T) {
	// Envelope.Exp is stored as Unix seconds, so we need a TTL of one
	// second and a sleep of slightly more to cross the boundary
	// reliably without making the test flaky.
	s := newTestSession(t)
	s.TTL = time.Second

	rec := httptest.NewRecorder()
	if err := s.Issue(rec, testPayload{UserID: "stale"}); err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Sleep long enough that Now-Unix exceeds Exp by a full second
	// regardless of when in the second Issue ran (Exp = floor(now+TTL)).
	time.Sleep(2100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}

	if _, ok := s.Read(req); ok {
		t.Fatalf("Read must reject an envelope past its Exp")
	}
}

func TestCookieSession_Clear(t *testing.T) {
	s := newTestSession(t)

	rec := httptest.NewRecorder()
	s.Clear(rec)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly 1 Set-Cookie from Clear, got %d", len(cookies))
	}
	c := cookies[0]

	if c.Name != "sid" {
		t.Errorf("Name: got %q, want %q", c.Name, "sid")
	}
	if c.Value != "" {
		t.Errorf("Value: got %q, want empty", c.Value)
	}
	if c.MaxAge != -1 {
		t.Errorf("MaxAge: got %d, want -1 (immediate expiry)", c.MaxAge)
	}
}

func TestCookieSession_Defaults(t *testing.T) {
	// All zero-value fields must produce sane defaults so a literal
	// CookieSession[T]{Secret: ..., TTL: ...} still works.
	s := CookieSession[testPayload]{
		Secret: "x-secret",
		TTL:    time.Hour,
	}

	rec := httptest.NewRecorder()
	if err := s.Issue(rec, testPayload{UserID: "1"}); err != nil {
		t.Fatalf("Issue with zero-valued attributes: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]

	if c.Name != "session" {
		t.Errorf("Name default: got %q, want %q", c.Name, "session")
	}
	if c.Path != "/" {
		t.Errorf("Path default: got %q, want %q", c.Path, "/")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite default: got %v, want Lax", c.SameSite)
	}
}
