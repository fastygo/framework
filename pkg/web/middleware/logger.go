package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *statusResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *statusResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func LoggerMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &statusResponseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}
			requestID := r.Header.Get(RequestIDHeader)

			logger := slog.Default()
			logger.Info(
				"http.request",
				"request_id",
				requestID,
				"method",
				r.Method,
				"path",
				r.URL.Path,
			)

			next.ServeHTTP(wrapped, r)

			logger.Info(
				"http.response",
				"request_id",
				requestID,
				"status",
				wrapped.statusCode,
				"duration_ms",
				time.Since(start).Milliseconds(),
				"size",
				wrapped.size,
			)
		})
	}
}
