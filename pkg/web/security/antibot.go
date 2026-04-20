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

// AntiBotMiddleware blocks requests whose User-Agent is empty (when
// cfg.BlockEmptyUA) or matches one of a small set of well-known
// scanner signatures (sqlmap, nikto, nessus, dirbuster, masscan,
// nmap, openvas). Blocked requests get HTTP 403 without further
// processing.
//
// This is intentionally a deterrent, not a defence: a determined
// attacker can override their UA. Pair with rate limiting and WAF
// rules at the edge.
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
