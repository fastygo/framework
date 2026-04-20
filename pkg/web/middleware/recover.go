package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

// sanitizeForLog strips CR/LF/tab so an attacker cannot inject log lines
// through a crafted X-Request-ID header (gosec G706 / log injection).
func sanitizeForLog(s string) string {
	if s == "" {
		return s
	}
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return '_'
		}
		if r < 0x20 || r == 0x7f {
			return '_'
		}
		return r
	}, s)
}

// RecoverMiddleware turns a downstream panic into a 500 response and
// a structured "http.panic" slog entry that includes the request id
// and the captured stack trace. It runs as the outermost wrapper in
// the security chain so even a panic from an earlier middleware is
// caught.
func RecoverMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					// gosec G706: request_id is sanitised via sanitizeForLog
					// (CR/LF/control chars stripped) before logging, so log
					// injection is not possible. The taint analyzer cannot
					// see through the helper; suppress it explicitly.
					//nolint:gosec // G706: input is sanitized via sanitizeForLog
					slog.Error(
						"http.panic",
						"error", recovered,
						"request_id", sanitizeForLog(r.Header.Get(RequestIDHeader)),
						"stack", string(debug.Stack()),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
