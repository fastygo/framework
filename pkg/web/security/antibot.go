package security

import (
	"net/http"
	"strings"

	"github.com/fastygo/framework/pkg/web/middleware"
)

var scannerSignatures = []string{
	"sqlmap",
	"nikto",
	"nessus",
	"dirbuster",
	"masscan",
	"nmap",
	"openvas",
}

func AntiBotMiddleware(cfg Config) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
			if cfg.BlockEmptyUA && userAgent == "" {
				http.Error(w, "blocked request", http.StatusForbidden)
				return
			}

			for _, signature := range scannerSignatures {
				if signature != "" && strings.Contains(userAgent, signature) {
					http.Error(w, "blocked request", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
