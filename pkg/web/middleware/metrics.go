package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fastygo/framework/pkg/web/metrics"
)

// MetricsMiddleware records HTTP request counters, in-flight gauge, and
// duration histogram into reg. Three metrics are emitted with stable,
// Prometheus-compatible names:
//
//   - http_requests_total{method,status}        counter
//   - http_requests_in_flight                   gauge
//   - http_request_duration_seconds{method}     histogram (default buckets)
//
// Pass nil to disable: the middleware degrades to a passthrough with
// zero overhead beyond the wrapper allocation.
func MetricsMiddleware(reg *metrics.Registry) Middleware {
	if reg == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	requests := reg.Counter(
		"http_requests_total",
		"Total number of HTTP requests processed.",
		"method", "status",
	)
	inFlight := reg.Gauge(
		"http_requests_in_flight",
		"Number of HTTP requests currently being served.",
	)
	duration := reg.Histogram(
		"http_request_duration_seconds",
		"Duration of HTTP requests in seconds.",
		nil,
		"method",
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inFlight.Inc()
			defer inFlight.Dec()

			start := time.Now()
			wrapped := &statusResponseWriter{ResponseWriter: w}

			next.ServeHTTP(wrapped, r)

			status := wrapped.statusCode
			if status == 0 {
				status = http.StatusOK
			}
			method := r.Method

			requests.Inc(method, strconv.Itoa(status))
			duration.Observe(time.Since(start).Seconds(), method)
		})
	}
}
