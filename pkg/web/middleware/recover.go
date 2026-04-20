package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

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
					slog.Error(
						"http.panic",
						"error", recovered,
						"request_id", r.Header.Get(RequestIDHeader),
						"stack", string(debug.Stack()),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
