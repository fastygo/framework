package cab

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "fastygo_session"
	stateCookieName   = "fastygo_oauth_state"
	sessionTTL        = 8 * time.Hour
)

type SiteSession struct {
	Email     string `json:"email"`
	ExpiresAt int64  `json:"exp"`
}

func createSiteSession(w http.ResponseWriter, email, sessionKey string) {
	sess := SiteSession{
		Email:     email,
		ExpiresAt: time.Now().Add(sessionTTL).Unix(),
	}
	val := signedEncode(sess, sessionKey)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func getSiteSession(r *http.Request, sessionKey string) *SiteSession {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	var sess SiteSession
	if err := signedDecode(cookie.Value, sessionKey, &sess); err != nil {
		return nil
	}
	if time.Now().Unix() > sess.ExpiresAt {
		return nil
	}
	return &sess
}

func clearSiteSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func saveOAuthState(w http.ResponseWriter, state, sessionKey string) {
	val := signedEncode(map[string]any{
		"state": state,
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	}, sessionKey)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
}

func getOAuthState(r *http.Request, sessionKey string) string {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		return ""
	}
	var data struct {
		State string `json:"state"`
		Exp   int64  `json:"exp"`
	}
	if err := signedDecode(cookie.Value, sessionKey, &data); err != nil {
		return ""
	}
	if time.Now().Unix() > data.Exp {
		return ""
	}
	return data.State
}

func clearOAuthState(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func signedEncode(data any, key string) string {
	payload, _ := json.Marshal(data)
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	mac := computeHMAC(b64, key)
	return b64 + "." + mac
}

func signedDecode(value, key string, dst any) error {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format")
	}
	expected := computeHMAC(parts[0], key)
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dst)
}

func computeHMAC(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
