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

// ----- BodyLimit -----------------------------------------------------------

func TestBodyLimit_Disabled_PassesThrough(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxBodySize: 0}
	called := false
	handler := BodyLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(strings.Repeat("x", 1024)))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("handler must run when MaxBodySize is 0")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204", rr.Code)
	}
}

func TestBodyLimit_AllowsRequestUnderLimit(t *testing.T) {
	t.Parallel()
	cfg := Config{MaxBodySize: 16}
	handler := BodyLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 32)
		n, _ := r.Body.Read(buf)
		_, _ = w.Write(buf[:n])
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("hello"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if rr.Body.String() != "hello" {
		t.Fatalf("body: got %q, want hello", rr.Body.String())
	}
}

// ----- RateLimiter constructor ---------------------------------------------

func TestNewRateLimiter_ClampsNonPositive(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(-5, -1)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	if rl.rate != 1 {
		t.Errorf("rate: got %v, want 1 (clamped)", rl.rate)
	}
	if rl.burst != 1 {
		t.Errorf("burst: got %v, want 1 (clamped)", rl.burst)
	}
	if !rl.Allow("ip-1") {
		t.Errorf("clamped limiter must still allow first request")
	}
}

func TestRateLimiter_Cleanup_DropsStaleVisitors(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 1)

	rl.Allow("a")
	rl.Allow("b")

	// Wait a moment, then run cleanup with a very short staleness
	// window so both visitors are dropped.
	time.Sleep(20 * time.Millisecond)
	rl.Cleanup(5 * time.Millisecond)

	// Count remaining visitors across all shards.
	total := 0
	for i := range rl.shards {
		rl.shards[i].mu.Lock()
		total += len(rl.shards[i].visitors)
		rl.shards[i].mu.Unlock()
	}
	if total != 0 {
		t.Fatalf("Cleanup must drop all stale visitors, %d remain", total)
	}
}

func TestRateLimiter_Cleanup_KeepsFreshVisitors(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 1)
	rl.Allow("fresh")

	// Long staleAfter ⇒ nothing dropped.
	rl.Cleanup(time.Hour)

	total := 0
	for i := range rl.shards {
		rl.shards[i].mu.Lock()
		total += len(rl.shards[i].visitors)
		rl.shards[i].mu.Unlock()
	}
	if total == 0 {
		t.Fatalf("Cleanup with long staleAfter must keep fresh visitors")
	}
}

func TestRateLimitMiddleware_NilLimiter_PassesThrough(t *testing.T) {
	t.Parallel()
	handler := RateLimitMiddleware(nil, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204 (nil limiter is no-op)", rr.Code)
	}
}

func TestRateLimitMiddleware_UnknownIP_StillRateLimited(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1000, 1)
	handler := RateLimitMiddleware(rl, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// Strip RemoteAddr so ClientIP returns "" → bucket key "unknown".
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ""
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("first unknown request: got %d, want 204", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second unknown request: got %d, want 429", rr.Code)
	}
}

// ----- MethodGuard / containsSuspiciousPath --------------------------------

func TestMethodGuard_AllSuspiciousPatterns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
	}{
		// path.Clean does not collapse ".." tokens that are part of
		// a filename (not a separate segment) — those survive into
		// containsSuspiciousPath and trigger the .. detector.
		{"double-dot embedded in name", "/foo..bar"},
		{"nul-byte", "/foo\x00bar"},
		{"env file", "/.env"},
		{"git directory", "/.git/config"},
		{"wp-admin", "/wp-admin/setup.php"},
		{"wp-login", "/wp-login.php"},
		{"case insensitive", "/WP-LOGIN.PHP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !containsSuspiciousPath(tc.path) {
				t.Fatalf("containsSuspiciousPath(%q) = false, want true", tc.path)
			}
		})
	}
}

// Documented behaviour: well-formed traversal segments are normalised
// by path.Clean before MethodGuard sees them, so they do NOT trigger
// the .. detector. Path traversal is enforced again at the file
// server layer (SecureFileServer) where it matters.
func TestMethodGuard_NormalisedTraversal_NotFlagged(t *testing.T) {
	t.Parallel()
	if containsSuspiciousPath("/foo/../etc/passwd") {
		t.Fatalf("containsSuspiciousPath should NOT flag normalisable traversal — SecureFileServer is the second line of defence")
	}
}

func TestMethodGuard_BenignPaths_NotSuspicious(t *testing.T) {
	t.Parallel()
	for _, p := range []string{"/", "/api/users", "/static/app.css", "/blog/post-1"} {
		if containsSuspiciousPath(p) {
			t.Errorf("containsSuspiciousPath(%q) = true, want false", p)
		}
	}
}

func TestMethodGuard_Connect_Rejected(t *testing.T) {
	t.Parallel()
	handler := MethodGuardMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodConnect, "/", nil)
	req.Host = "example.com:443"
	req.URL.Scheme = "https"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("CONNECT must be rejected: got %d", rr.Code)
	}
}

// ----- SecureFileServer ----------------------------------------------------

func TestSecureFileServer_NonImmutable_ShortMaxAge(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "page.html"), []byte("<h1>x</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := SecureFileServer(root, 999)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/page.html", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	cc := rr.Header().Get("Cache-Control")
	if cc != "public, max-age=60" {
		t.Errorf("non-immutable extension Cache-Control: got %q, want public, max-age=60", cc)
	}
}

func TestSecureFileServer_DefaultsMaxAgeOneDayWhenNonPositive(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.css"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := SecureFileServer(root, 0) // non-positive

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	cc := rr.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age=86400") || !strings.Contains(cc, "immutable") {
		t.Fatalf("expected one-day immutable cache, got %q", cc)
	}
}

func TestSecureFileServer_NotModified_OnIfNoneMatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := SecureFileServer(root, 60)

	// First request to capture ETag.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("no ETag returned")
	}

	// Second request with matching If-None-Match → 304.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	req2.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match match: got %d, want 304", rr2.Code)
	}
}

func TestSecureFileServer_DotDotRejected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	handler := SecureFileServer(root, 60)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/../etc/passwd", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("path-traversal: got %d, want 404", rr.Code)
	}
}

func TestSecureFileServer_MissingFile_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	handler := SecureFileServer(root, 60)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/missing.css", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing file: got %d, want 404", rr.Code)
	}
}

func TestSecureFileServer_DirectoryListing_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := SecureFileServer(root, 60)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/subdir/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("directory request: got %d, want 404", rr.Code)
	}
}

func TestSecureFileServer_RootRequest_NotFound(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	handler := SecureFileServer(root, 60)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	// Root resolves to the directory itself; SecureFileServer treats
	// directories as 404 (no listing).
	if rr.Code != http.StatusNotFound {
		t.Fatalf("root request: got %d, want 404", rr.Code)
	}
}

func TestCacheControlValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		path   string
		maxAge int
		want   string
	}{
		{"css immutable", "app.css", 600, "public, max-age=600, immutable"},
		{"woff2 immutable", "font.woff2", 600, "public, max-age=600, immutable"},
		{"non-immutable html", "page.html", 600, "public, max-age=60"},
		{"immutable defaults to one day", "logo.svg", 0, "public, max-age=86400, immutable"},
		{"immutable defaults to one day on negative", "logo.svg", -1, "public, max-age=86400, immutable"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheControlValue(tc.path, tc.maxAge)
			if got != tc.want {
				t.Errorf("cacheControlValue(%q, %d): got %q, want %q",
					tc.path, tc.maxAge, got, tc.want)
			}
		})
	}
}

// Known-issue regression test: filepath.Ext("robots") returns "" and
// strings.Contains(defaultImmutableExtensions, "") is true, so files
// without an extension are currently classified as immutable. We
// document this here so the day someone tightens cacheControlValue
// (e.g. by switching to an exact-match lookup) the test breaks loudly
// and the change is intentional.
func TestCacheControlValue_KnownIssue_NoExtensionTreatedAsImmutable(t *testing.T) {
	t.Parallel()
	got := cacheControlValue("robots", 600)
	if got != "public, max-age=600, immutable" {
		t.Fatalf("known-issue snapshot changed; please update the test "+
			"and add a CHANGELOG note. got=%q", got)
	}
}

func TestHasDotSegment(t *testing.T) {
	t.Parallel()
	tests := map[string]bool{
		"app.css":        false,
		".env":           true,
		"foo/.git/HEAD":  true,
		"foo/bar/baz":    false,
		".":              true,
		"":               false, // strings.Split returns [""] which doesn't start with "."
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			if got := hasDotSegment(path); got != want {
				t.Errorf("hasDotSegment(%q): got %v, want %v", path, got, want)
			}
		})
	}
}

// silence unused import lint quirks for time when tests are pruned.
var _ = time.Now
