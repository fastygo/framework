package cab

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

type ProviderConfig struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

type IDTokenClaims struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	Email string `json:"email"`
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

type OIDCClient struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURI  string
	httpClient   *http.Client

	mu       sync.RWMutex
	provider *ProviderConfig
	fetched  time.Time
}

func NewOIDCClient(issuer, clientID, clientSecret, redirectURI string) *OIDCClient {
	return &OIDCClient{
		issuer:       strings.TrimRight(issuer, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *OIDCClient) AuthorizationURL(state, nonce string) (string, error) {
	provider, err := c.Discovery()
	if err != nil {
		return "", err
	}

	u, err := url.Parse(provider.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse auth endpoint: %w", err)
	}

	q := u.Query()
	q.Set("client_id", c.clientID)
	q.Set("redirect_uri", c.redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid email")
	q.Set("state", state)
	if nonce != "" {
		q.Set("nonce", nonce)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *OIDCClient) LogoutURL(returnTo string) string {
	logoutURL := c.issuer + "/logout"
	if returnTo == "" {
		return logoutURL
	}
	return logoutURL + "?return_to=" + url.QueryEscape(returnTo)
}

func (c *OIDCClient) AppOrigin() string {
	u, err := url.Parse(c.redirectURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func (c *OIDCClient) Discovery() (*ProviderConfig, error) {
	c.mu.RLock()
	if c.provider != nil && time.Since(c.fetched) < 10*time.Minute {
		defer c.mu.RUnlock()
		return c.provider, nil
	}
	c.mu.RUnlock()

	resp, err := c.httpClient.Get(c.issuer + "/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery returned status %d", resp.StatusCode)
	}

	var pc ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&pc); err != nil {
		return nil, fmt.Errorf("parse discovery: %w", err)
	}

	c.mu.Lock()
	c.provider = &pc
	c.fetched = time.Now()
	c.mu.Unlock()

	return &pc, nil
}

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
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	return &tr, nil
}

func (c *OIDCClient) VerifyIDToken(idToken string) (*IDTokenClaims, error) {
	provider, err := c.Discovery()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	headerJSON, err := b64Decode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	pubKey, err := c.fetchPublicKey(provider.JWKSURI, header.Kid)
	if err != nil {
		return nil, fmt.Errorf("fetch key: %w", err)
	}

	sigBytes, err := b64Decode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	signed := parts[0] + "." + parts[1]
	h := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sigBytes); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	payloadJSON, err := b64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	if claims.Iss != c.issuer {
		return nil, fmt.Errorf("issuer mismatch: got %q, want %q", claims.Iss, c.issuer)
	}
	if claims.Aud != c.clientID {
		return nil, fmt.Errorf("audience mismatch: got %q, want %q", claims.Aud, c.clientID)
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func (c *OIDCClient) fetchPublicKey(jwksURI, kid string) (*rsa.PublicKey, error) {
	resp, err := c.httpClient.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var ks jwkSet
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return nil, fmt.Errorf("parse JWKS: %w", err)
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

	return nil, fmt.Errorf("key %q not found in JWKS", kid)
}

func jwkToRSA(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := b64Decode(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := b64Decode(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

func b64Decode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
