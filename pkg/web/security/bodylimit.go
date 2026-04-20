package security

import (
	"net/http"

	"github.com/fastygo/framework/pkg/web/middleware"
)

// BodyLimitMiddleware rejects requests with Content-Length above
// cfg.MaxBodySize (HTTP 413) and wraps the body with
// http.MaxBytesReader so handlers reading the body never exceed the
// limit even when Content-Length is absent or lying. A zero
// MaxBodySize disables the middleware.
func BodyLimitMiddleware(cfg Config) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.MaxBodySize <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > cfg.MaxBodySize {
				http.Error(w, "request body is too large", http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodySize)
			next.ServeHTTP(w, r)
		})
	}
}
