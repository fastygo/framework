package security

import (
	"net/http"
	"path"
	"strings"

	"github.com/fastygo/framework/pkg/web/middleware"
)

var blockedPatterns = []string{
	".env",
	".git",
	"wp-admin",
	"wp-login",
}

func MethodGuardMiddleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodTrace || r.Method == http.MethodConnect {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if containsSuspiciousPath(r.URL.Path) {
				http.NotFound(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func containsSuspiciousPath(requestPath string) bool {
	cleanPath := path.Clean(requestPath)
	if strings.Contains(cleanPath, "..") {
		return true
	}
	if strings.Contains(cleanPath, "\x00") {
		return true
	}
	for _, pattern := range blockedPatterns {
		if strings.Contains(strings.ToLower(cleanPath), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
