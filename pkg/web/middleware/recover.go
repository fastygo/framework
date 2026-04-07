package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

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
