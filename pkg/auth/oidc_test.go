package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// b64URL replicates the JWT base64-URL encoding without padding so
// tests can build tokens by hand.
func b64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// signIDToken builds a real RS256 ID token signed with key for the
// supplied claims. The header advertises kid for JWKS lookup.
func signIDToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(claims)

	signingInput := b64URL(headerJSON) + "." + b64URL(payloadJSON)
	h := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + b64URL(sig)
}

// jwksFromKey serialises the public part of key as a single-entry
// JWKS document with the supplied kid.
func jwksFromKey(t *testing.T, key *rsa.PrivateKey, kid string) []byte {
	t.Helper()
	pub := &key.PublicKey
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	doc := map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"kid": kid,
			"use": "sig",
			"alg": "RS256",
			"n":   b64URL(pub.N.Bytes()),
			"e":   b64URL(eBytes),
		}},
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return out
}

// oidcMock is a minimal OpenID Provider stub: discovery, JWKS, and
// token endpoints. Each handler counts hits so tests can assert
// caching behaviour.
type oidcMock struct {
	server        *httptest.Server
	discoveryHits atomic.Int32
	jwksHits      atomic.Int32
	tokenHits     atomic.Int32

	jwks          []byte
	tokenStatus   int // override for token endpoint
	tokenBody     []byte
	failDiscovery bool
}

func newOIDCMock(t *testing.T, key *rsa.PrivateKey, kid string) *oidcMock {
	t.Helper()
	m := &oidcMock{
		jwks:        jwksFromKey(t, key, kid),
		tokenStatus: http.StatusOK,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		m.discoveryHits.Add(1)
		if m.failDiscovery {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		base := "http://" + r.Host
		_ = json.NewEncoder(w).Encode(ProviderConfig{
			Issuer:                base,
			AuthorizationEndpoint: base + "/authorize",
			TokenEndpoint:         base + "/token",
			UserinfoEndpoint:      base + "/userinfo",
			JWKSURI:               base + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		m.jwksHits.Add(1)
		_, _ = w.Write(m.jwks)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		m.tokenHits.Add(1)
		_ = r.ParseForm()
		if m.tokenStatus != http.StatusOK {
			w.WriteHeader(m.tokenStatus)
			return
		}
		if m.tokenBody != nil {
			_, _ = w.Write(m.tokenBody)
			return
		}
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "atk-" + r.FormValue("code"),
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			IDToken:     "stub-id-token",
		})
	})

	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func newClient(t *testing.T, m *oidcMock) *OIDCClient {
	t.Helper()
	return NewOIDCClient(OIDCConfig{
		Issuer:       m.server.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://app.test/callback",
		Scopes:       []string{"openid", "email"},
		HTTPClient:   m.server.Client(),
	})
}

func TestNewOIDCClient_Defaults(t *testing.T) {
	c := NewOIDCClient(OIDCConfig{
		Issuer:      "https://issuer.example/",
		RedirectURI: "https://app.example/callback",
	})
	if c == nil {
		t.Fatal("NewOIDCClient returned nil")
	}
	// Trailing slash on Issuer must be trimmed.
	if c.issuer != "https://issuer.example" {
		t.Errorf("issuer: got %q, want trimmed", c.issuer)
	}
	if len(c.scopes) != 2 || c.scopes[0] != "openid" {
		t.Errorf("default scopes: got %v, want [openid email]", c.scopes)
	}
	if c.httpClient == nil {
		t.Errorf("httpClient must default to a non-nil *http.Client")
	}
}

func TestOIDCClient_AppOrigin(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		want        string
	}{
		{"https origin", "https://app.example/callback", "https://app.example"},
		{"with port", "http://localhost:8080/cb", "http://localhost:8080"},
		{"invalid", "://broken", ""},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewOIDCClient(OIDCConfig{Issuer: "https://x", RedirectURI: tc.redirectURI})
			got := c.AppOrigin()
			if got != tc.want {
				t.Errorf("AppOrigin: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOIDCClient_LogoutURL(t *testing.T) {
	c := NewOIDCClient(OIDCConfig{Issuer: "https://issuer.example", RedirectURI: "x"})

	if got := c.LogoutURL(""); got != "https://issuer.example/logout" {
		t.Errorf("LogoutURL(empty): got %q", got)
	}
	got := c.LogoutURL("https://app.example/")
	want := "https://issuer.example/logout?return_to=" + url.QueryEscape("https://app.example/")
	if got != want {
		t.Errorf("LogoutURL: got %q, want %q", got, want)
	}
}

func TestOIDCClient_AuthorizationURL_BuildsExpectedQuery(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	got, err := c.AuthorizationURL("state-xyz", "nonce-abc")
	if err != nil {
		t.Fatalf("AuthorizationURL: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := u.Query()
	if q.Get("client_id") != "test-client" {
		t.Errorf("client_id: got %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "http://app.test/callback" {
		t.Errorf("redirect_uri: got %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type: got %q", q.Get("response_type"))
	}
	if q.Get("scope") != "openid email" {
		t.Errorf("scope: got %q", q.Get("scope"))
	}
	if q.Get("state") != "state-xyz" {
		t.Errorf("state: got %q", q.Get("state"))
	}
	if q.Get("nonce") != "nonce-abc" {
		t.Errorf("nonce: got %q", q.Get("nonce"))
	}
}

func TestOIDCClient_AuthorizationURL_NoNonceWhenEmpty(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	got, _ := c.AuthorizationURL("s", "")
	u, _ := url.Parse(got)
	if u.Query().Has("nonce") {
		t.Errorf("nonce parameter must be absent when empty: %q", got)
	}
}

func TestOIDCClient_Discovery_CachesResult(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	for i := 0; i < 3; i++ {
		if _, err := c.Discovery(); err != nil {
			t.Fatalf("Discovery #%d: %v", i, err)
		}
	}

	if hits := m.discoveryHits.Load(); hits != 1 {
		t.Errorf("expected discovery to be fetched once and cached, got %d hits", hits)
	}
}

func TestOIDCClient_Discovery_PropagatesNon200(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	m.failDiscovery = true
	c := newClient(t, m)

	_, err := c.Discovery()
	if err == nil {
		t.Fatalf("Discovery must return an error on non-200 response")
	}
	if !strings.Contains(err.Error(), "discovery returned status") {
		t.Errorf("error should mention discovery status, got %v", err)
	}
}

func TestOIDCClient_DiscoveryContext_UsesCancellation(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.DiscoveryContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DiscoveryContext must return context cancellation, got %v", err)
	}
}

func TestOIDCClient_ExchangeCode_RoundTrip(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	tr, err := c.ExchangeCode("auth-code-1")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tr.AccessToken != "atk-auth-code-1" {
		t.Errorf("AccessToken: got %q", tr.AccessToken)
	}
	if tr.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q", tr.TokenType)
	}
}

func TestOIDCClient_ExchangeCodeContext_UsesCancellation(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)
	if _, err := c.Discovery(); err != nil {
		t.Fatalf("Discovery: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.ExchangeCodeContext(ctx, "auth-code-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ExchangeCodeContext must return context cancellation, got %v", err)
	}
}

func TestOIDCClient_ExchangeCode_PropagatesNon200(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	m.tokenStatus = http.StatusBadRequest
	c := newClient(t, m)

	_, err := c.ExchangeCode("bad-code")
	if err == nil {
		t.Fatalf("ExchangeCode must propagate non-200 from token endpoint")
	}
	if !strings.Contains(err.Error(), "token endpoint returned status") {
		t.Errorf("error should mention status, got %v", err)
	}
}

func TestOIDCClient_ExchangeCode_BadJSON(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	m.tokenBody = []byte("not-json")
	c := newClient(t, m)

	_, err := c.ExchangeCode("x")
	if err == nil {
		t.Fatalf("ExchangeCode must error on malformed JSON")
	}
}

func TestOIDCClient_VerifyIDToken_Success(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)

	idToken := signIDToken(t, key, "kid-1", map[string]any{
		"iss":   m.server.URL,
		"sub":   "user-123",
		"aud":   "test-client",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"email": "ada@example.com",
		"nonce": "n-1",
	})

	claims, err := c.VerifyIDToken(idToken)
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	if claims.Sub != "user-123" {
		t.Errorf("Sub: got %q", claims.Sub)
	}
	if claims.Email != "ada@example.com" {
		t.Errorf("Email: got %q", claims.Email)
	}
}

func TestOIDCClient_VerifyIDTokenContext_UsesCancellationForJWKS(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)
	if _, err := c.Discovery(); err != nil {
		t.Fatalf("Discovery: %v", err)
	}

	idToken := signIDToken(t, key, "kid-1", map[string]any{
		"iss": m.server.URL,
		"sub": "user-123",
		"aud": "test-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.VerifyIDTokenContext(ctx, idToken)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("VerifyIDTokenContext must return context cancellation, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_RejectsInvalidFormat(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	_, err := c.VerifyIDToken("only.two-parts")
	if err == nil || !strings.Contains(err.Error(), "JWT format") {
		t.Fatalf("expected JWT format error, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_RejectsBadSignature(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048) // signed with the wrong key
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)

	idToken := signIDToken(t, other, "kid-1", map[string]any{
		"iss": m.server.URL,
		"sub": "u",
		"aud": "test-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := c.VerifyIDToken(idToken)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_RejectsIssuerMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)

	idToken := signIDToken(t, key, "kid-1", map[string]any{
		"iss": "https://attacker.example",
		"sub": "u",
		"aud": "test-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := c.VerifyIDToken(idToken)
	if err == nil || !strings.Contains(err.Error(), "issuer mismatch") {
		t.Fatalf("expected issuer mismatch, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_RejectsAudienceMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)

	idToken := signIDToken(t, key, "kid-1", map[string]any{
		"iss": m.server.URL,
		"sub": "u",
		"aud": "other-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := c.VerifyIDToken(idToken)
	if err == nil || !strings.Contains(err.Error(), "audience mismatch") {
		t.Fatalf("expected audience mismatch, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_RejectsExpired(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "kid-1")
	c := newClient(t, m)

	idToken := signIDToken(t, key, "kid-1", map[string]any{
		"iss": m.server.URL,
		"sub": "u",
		"aud": "test-client",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	_, err := c.VerifyIDToken(idToken)
	if err == nil || !strings.Contains(err.Error(), "token expired") {
		t.Fatalf("expected token expired, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_KidMissingInJWKS(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "right-kid")
	c := newClient(t, m)

	idToken := signIDToken(t, key, "wrong-kid", map[string]any{
		"iss": m.server.URL,
		"sub": "u",
		"aud": "test-client",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	_, err := c.VerifyIDToken(idToken)
	if err == nil || !strings.Contains(err.Error(), "not found in JWKS") {
		t.Fatalf("expected JWKS lookup failure, got %v", err)
	}
}

func TestOIDCClient_VerifyIDToken_BadHeaderBase64(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	m := newOIDCMock(t, key, "k1")
	c := newClient(t, m)

	// Three "parts" but the header is not base64.
	bad := "***." + b64URL([]byte(`{"sub":"x"}`)) + "." + b64URL([]byte("sig"))
	_, err := c.VerifyIDToken(bad)
	if err == nil || !strings.Contains(err.Error(), "decode header") {
		t.Fatalf("expected decode header error, got %v", err)
	}
}

// helper to silence unused-import lint when fmt is only used in the
// guard above for go vet quirks; not actually needed but keeps vet
// quiet on pruned tests.
var _ = fmt.Sprintf
var _ io.Reader = (*strings.Reader)(nil)
