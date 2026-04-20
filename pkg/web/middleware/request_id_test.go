package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// uuidV4Pattern enforces the canonical 8-4-4-4-12 layout, version
// nibble 4, and the 8/9/a/b variant nibble.
var uuidV4Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

func TestNewRequestID_FormatAndUniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1024)
	for i := 0; i < 1024; i++ {
		id := newRequestID()
		if !uuidV4Pattern.MatchString(id) {
			t.Fatalf("generated id %q does not match UUID v4 pattern", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("collision after %d ids: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestRequestIDMiddleware_GeneratesWhenAbsent(t *testing.T) {
	t.Parallel()

	var seen string
	h := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if seen == "" {
		t.Fatal("request id was not generated when header was absent")
	}
	if got := rec.Header().Get(RequestIDHeader); got != seen {
		t.Errorf("response header = %q, want %q", got, seen)
	}
	if !uuidV4Pattern.MatchString(seen) {
		t.Errorf("generated id %q is not UUID v4", seen)
	}
}

func TestRequestIDMiddleware_PreservesIncomingHeader(t *testing.T) {
	t.Parallel()

	const incoming = "client-correlation-token-42"
	var seen string
	h := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, incoming)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if seen != incoming {
		t.Errorf("context id = %q, want %q (must echo client header)", seen, incoming)
	}
	if got := rec.Header().Get(RequestIDHeader); got != incoming {
		t.Errorf("response header = %q, want %q", got, incoming)
	}
}
