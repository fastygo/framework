package metrics

import (
	"net/http"
)

// Handler returns an http.Handler that serves the Prometheus text
// exposition format from r. Content-Type matches the spec
// (text/plain; version=0.0.4) so any standard scraper can ingest it.
func Handler(r *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if r == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = r.Write(w)
	})
}
