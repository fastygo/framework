package security

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHeadersMiddlewareSetsSecurityHeaders(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.CSP = "default-src 'self'"
	cfg.HSTS = true

	req := httptest.NewRequest(http.MethodGet, "/security", nil)
	rr := httptest.NewRecorder()

	handler := HeadersMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options=nosniff, got %q", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != cfg.FrameOptions {
		t.Fatalf("expected X-Frame-Options=%q, got %q", cfg.FrameOptions, got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got == "" {
		t.Fatalf("expected Referrer-Policy header")
	}
	if got := rr.Header().Get("Content-Security-Policy"); got != cfg.CSP {
		t.Fatalf("expected CSP=%q, got %q", cfg.CSP, got)
	}
	if got := rr.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatalf("expected HSTS header when enabled")
	}
}

func TestBodyLimitMiddleware(t *testing.T) {
	t.Parallel()

	cfg := Config{MaxBodySize: 4}
	handler := BodyLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("12345"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}
}

func TestMethodGuardMiddleware(t *testing.T) {
	t.Parallel()

	handler := MethodGuardMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodTrace, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/.env", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected not found, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/safe", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAntiBotMiddleware(t *testing.T) {
	t.Parallel()

	cfg := Config{BlockEmptyUA: true}
	handler := AntiBotMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d for empty User-Agent, got %d", http.StatusForbidden, rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "sqlmap")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d for scanner User-Agent, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1000, 1)
	if !rl.Allow("127.0.0.1") {
		t.Fatalf("expected first request to be allowed")
	}
	if rl.Allow("127.0.0.1") {
		t.Fatalf("expected burst limit to block second request")
	}
	time.Sleep(5 * time.Millisecond)
	if !rl.Allow("127.0.0.1") {
		t.Fatalf("expected refill to allow request after short time window")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1000, 1)
	handler := RateLimitMiddleware(rl, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
}

func TestClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		xff        string
		trustProxy bool
		wantIP     string
	}{
		{
			name:       "RemoteAddr only, no proxy trust",
			remoteAddr: "192.0.2.1:4321",
			trustProxy: false,
			wantIP:     "192.0.2.1",
		},
		{
			name:       "X-Real-IP ignored when trustProxy=false",
			remoteAddr: "192.0.2.1:4321",
			xRealIP:    "10.0.0.1",
			trustProxy: false,
			wantIP:     "192.0.2.1",
		},
		{
			name:       "X-Real-IP used when trustProxy=true",
			remoteAddr: "192.0.2.1:4321",
			xRealIP:    "10.0.0.1",
			trustProxy: true,
			wantIP:     "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "192.0.2.1:4321",
			xff:        "203.0.113.50",
			trustProxy: true,
			wantIP:     "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For multiple IPs takes leftmost",
			remoteAddr: "192.0.2.1:4321",
			xff:        "203.0.113.50, 70.41.3.18, 150.172.238.178",
			trustProxy: true,
			wantIP:     "203.0.113.50",
		},
		{
			name:       "X-Real-IP takes priority over X-Forwarded-For",
			remoteAddr: "192.0.2.1:4321",
			xRealIP:    "10.0.0.1",
			xff:        "203.0.113.50",
			trustProxy: true,
			wantIP:     "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.0.2.1",
			trustProxy: false,
			wantIP:     "192.0.2.1",
		},
		{
			name:       "Fallback to RemoteAddr when proxy headers empty",
			remoteAddr: "192.0.2.1:4321",
			trustProxy: true,
			wantIP:     "192.0.2.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			got := ClientIP(req, tc.trustProxy)
			if got != tc.wantIP {
				t.Fatalf("ClientIP() = %q, want %q", got, tc.wantIP)
			}
		})
	}
}

func TestRateLimitMiddlewareWithTrustProxy(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(1000, 1)
	handler := RateLimitMiddleware(rl, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Real-IP", "203.0.113.50")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	req2.Header.Set("X-Real-IP", "198.51.100.10")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req2)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected different proxy IP to be allowed, got %d", rr.Code)
	}
}

func TestSecureFileServer(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "app.css"), []byte("body {}"), 0o644)
	_ = os.WriteFile(filepath.Join(root, ".env"), []byte("secret"), 0o644)
	handler := SecureFileServer(root, 120)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); !strings.Contains(got, "max-age=120") {
		t.Fatalf("expected cache-control max-age=120, got %q", got)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/.env", nil)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Fatalf("expected forbidden or not found for dotfile, got %d", rr.Code)
	}
}
