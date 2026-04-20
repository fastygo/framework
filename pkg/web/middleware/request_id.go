package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

// ContextWithRequestID returns a child context that carries
// requestID. Useful when a background goroutine needs to log against
// the originating request.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDFromContext returns the request id attached to ctx by
// RequestIDMiddleware (or ContextWithRequestID), or "" if none was set.
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}
	return ""
}

// RequestIDHeader is the canonical HTTP header carrying the request
// id (echoed both ways).
const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware echoes an incoming X-Request-ID header when
// present, otherwise generates a fresh RFC 4122 UUID v4 (via
// crypto/rand, no external dependency) and writes it to both the
// request context and the response header.
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = newRequestID()
				r.Header.Set(RequestIDHeader, requestID)
			}

			ctx := ContextWithRequestID(r.Context(), requestID)
			w.Header().Set(RequestIDHeader, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// newRequestID returns a freshly generated, RFC 4122-compliant UUID v4
// rendered in the canonical 36-character hyphenated form
// (xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx, where y is one of 8/9/a/b).
//
// We hand-roll this instead of importing github.com/google/uuid to keep
// the framework's go.mod free of an external dependency that would
// otherwise be pulled in for a single 30-line code path. crypto/rand
// is the same source the upstream library uses; on a rand failure we
// fall back to an empty string so the middleware can degrade
// gracefully (the X-Request-ID header simply won't be set).
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}

	// Set the four version bits to 0100 (UUID v4) and the two variant
	// bits to 10 (RFC 4122) per spec section 4.4.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	// Encode 16 bytes -> 32 hex chars + 4 hyphens = 36 chars.
	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])
	return string(out[:])
}
