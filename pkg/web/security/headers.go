package security

import (
	"net/http"

	"github.com/fastygo/framework/pkg/web/middleware"
)

func HeadersMiddleware(cfg Config) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", cfg.FrameOptions)
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", cfg.Permissions)
			if cfg.CSP != "" {
				w.Header().Set("Content-Security-Policy", cfg.CSP)
			}
			if cfg.HSTS {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}
