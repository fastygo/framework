package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ProviderConfig is the subset of OpenID Connect discovery metadata used by
// OIDCClient.
type ProviderConfig struct {
	// Issuer is the OpenID issuer URL (must equal the iss claim).
	Issuer string `json:"issuer"`
	// AuthorizationEndpoint is the URL the user agent is redirected to.
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	// TokenEndpoint exchanges the authorization code for tokens.
	TokenEndpoint string `json:"token_endpoint"`
	// UserinfoEndpoint returns claims about the authenticated user.
	UserinfoEndpoint string `json:"userinfo_endpoint"`
	// JWKSURI hosts the JSON Web Key Set used to verify ID-token signatures.
	JWKSURI string `json:"jwks_uri"`
}

// TokenResponse is the OAuth2 token endpoint response.
type TokenResponse struct {
	// AccessToken is the bearer token used to call protected APIs.
	AccessToken string `json:"access_token"`
	// TokenType is conventionally "Bearer".
	TokenType string `json:"token_type"`
	// ExpiresIn is the access-token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
	// IDToken is the signed JWT identifying the end user.
	IDToken string `json:"id_token"`
}

// IDTokenClaims captures the standard OIDC claims used by simple SSO flows.
type IDTokenClaims struct {
	// Iss identifies the issuer; must match ProviderConfig.Issuer.
	Iss string `json:"iss"`
	// Sub is the stable, opaque subject identifier (the "user id").
	Sub string `json:"sub"`
	// Aud is the audience and must contain the configured ClientID.
	Aud string `json:"aud"`
	// Exp is the token expiry as Unix seconds; tokens past Exp are rejected.
	Exp int64 `json:"exp"`
	// Iat is the issued-at time as Unix seconds.
	Iat int64 `json:"iat"`
	// Email is the user's email when the "email" scope was granted.
	Email string `json:"email"`
	// Nonce echoes the per-request value the client sent in the auth
	// URL; verifying it pins the ID token to the originating request.
	Nonce string `json:"nonce"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// OIDCClient is a tiny stdlib-only OpenID Connect client supporting the
// authorization-code flow and RS256 ID-token verification via JWKS.
type OIDCClient struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	httpClient   *http.Client

	mu       sync.RWMutex
	provider *ProviderConfig
	fetched  time.Time
}

// OIDCConfig groups the constructor arguments for NewOIDCClient.
type OIDCConfig struct {
	// Issuer is the OpenID Connect issuer URL.
	Issuer string
	// ClientID is the registered client identifier at the issuer.
	ClientID string
	// ClientSecret is the confidential client secret. Treat as PII.
	ClientSecret string
	// RedirectURI is the absolute callback URL the issuer sends the
	// authorization code back to.
	RedirectURI string
	// Scopes defaults to ["openid", "email"] when empty.
	Scopes []string
	// HTTPClient is the transport used for discovery, token exchange,
	// userinfo, and JWKS fetches. nil installs a 10-second-timeout client.
	HTTPClient *http.Client
}

// NewOIDCClient constructs an OIDCClient from the supplied configuration.
func NewOIDCClient(cfg OIDCConfig) *OIDCClient {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email"}
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &OIDCClient{
		issuer:       strings.TrimRight(cfg.Issuer, "/"),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURI:  cfg.RedirectURI,
		scopes:       scopes,
		httpClient:   httpClient,
	}
}

// AuthorizationURL builds the URL to redirect the user agent to begin login.
func (c *OIDCClient) AuthorizationURL(state, nonce string) (string, error) {
	provider, err := c.Discovery()
	if err != nil {
		return "", err
	}

	u, err := url.Parse(provider.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("auth: parse authorization endpoint: %w", err)
	}

	q := u.Query()
	q.Set("client_id", c.clientID)
	q.Set("redirect_uri", c.redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(c.scopes, " "))
	q.Set("state", state)
	if nonce != "" {
		q.Set("nonce", nonce)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// LogoutURL builds the issuer logout URL with an optional return-to.
func (c *OIDCClient) LogoutURL(returnTo string) string {
	logoutURL := c.issuer + "/logout"
	if returnTo == "" {
		return logoutURL
	}
	return logoutURL + "?return_to=" + url.QueryEscape(returnTo)
}

// AppOrigin extracts scheme://host from the configured redirect URI.
func (c *OIDCClient) AppOrigin() string {
	u, err := url.Parse(c.redirectURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// Discovery loads (and caches for 10 minutes) the issuer metadata.
func (c *OIDCClient) Discovery() (*ProviderConfig, error) {
	c.mu.RLock()
	if c.provider != nil && time.Since(c.fetched) < 10*time.Minute {
		defer c.mu.RUnlock()
		return c.provider, nil
	}
	c.mu.RUnlock()

	resp, err := c.httpClient.Get(c.issuer + "/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("auth: fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth: discovery returned status %d", resp.StatusCode)
	}

	var pc ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&pc); err != nil {
		return nil, fmt.Errorf("auth: parse discovery: %w", err)
	}

	c.mu.Lock()
	c.provider = &pc
	c.fetched = time.Now()
	c.mu.Unlock()

	return &pc, nil
}

// ExchangeCode swaps an authorization code for tokens.
func (c *OIDCClient) ExchangeCode(code string) (*TokenResponse, error) {
	provider, err := c.Discovery()
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.redirectURI},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	resp, err := c.httpClient.PostForm(provider.TokenEndpoint, form)
	if err != nil {
		return nil, fmt.Errorf("auth: token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth: token endpoint returned status %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("auth: parse token response: %w", err)
	}
	return &tr, nil
}

// VerifyIDToken validates the RS256 signature and standard claims.
func (c *OIDCClient) VerifyIDToken(idToken string) (*IDTokenClaims, error) {
	provider, err := c.Discovery()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("auth: invalid JWT format")
	}

	headerJSON, err := b64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("auth: decode header: %w", err)
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("auth: parse header: %w", err)
	}

	pubKey, err := c.fetchPublicKey(provider.JWKSURI, header.Kid)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch public key: %w", err)
	}

	sigBytes, err := b64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("auth: decode signature: %w", err)
	}

	signed := parts[0] + "." + parts[1]
	h := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sigBytes); err != nil {
		return nil, fmt.Errorf("auth: invalid signature: %w", err)
	}

	payloadJSON, err := b64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("auth: decode payload: %w", err)
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("auth: parse claims: %w", err)
	}
	if claims.Iss != c.issuer {
		return nil, fmt.Errorf("auth: issuer mismatch: got %q want %q", claims.Iss, c.issuer)
	}
	if claims.Aud != c.clientID {
		return nil, fmt.Errorf("auth: audience mismatch: got %q want %q", claims.Aud, c.clientID)
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("auth: token expired")
	}

	return &claims, nil
}

func (c *OIDCClient) fetchPublicKey(jwksURI, kid string) (*rsa.PublicKey, error) {
	resp, err := c.httpClient.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("auth: fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var ks jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return nil, fmt.Errorf("auth: parse JWKS: %w", err)
	}

	for _, k := range ks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if kid != "" && k.Kid != kid {
			continue
		}
		return jwkToRSA(k)
	}

	return nil, fmt.Errorf("auth: key %q not found in JWKS", kid)
}

func jwkToRSA(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := b64URLDecode(k.N)
	if err != nil {
		return nil, fmt.Errorf("auth: decode N: %w", err)
	}
	eBytes, err := b64URLDecode(k.E)
	if err != nil {
		return nil, fmt.Errorf("auth: decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}

func b64URLDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
