package security

import (
	"net/http"

	"github.com/fastygo/framework/pkg/web/middleware"
)

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
