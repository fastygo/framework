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

// LoggerMiddleware emits structured "http.request" / "http.response" slog
// events for every request. When a Correlation has been attached to the
// request context (typically by an OTel-aware tracer middleware), the
// log lines also carry trace_id and span_id so log entries can be
// joined to distributed traces.
//
// The middleware uses slog.LogAttrs rather than the variadic form so
// allocations stay flat under load: each call site builds a small,
// fixed-size []slog.Attr without escaping to the heap.
func LoggerMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &statusResponseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}
			requestID := r.Header.Get(RequestIDHeader)
			corr := CorrelationFromContext(r.Context())

			logger := slog.Default()
			ctx := r.Context()

			reqAttrs := make([]slog.Attr, 0, 5)
			reqAttrs = append(reqAttrs,
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
			)
			reqAttrs = appendCorrelation(reqAttrs, corr)
			logger.LogAttrs(ctx, slog.LevelInfo, "http.request", reqAttrs...)

			next.ServeHTTP(wrapped, r)

			respAttrs := make([]slog.Attr, 0, 6)
			respAttrs = append(respAttrs,
				slog.String("request_id", requestID),
				slog.Int("status", wrapped.statusCode),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.Int("size", wrapped.size),
			)
			respAttrs = appendCorrelation(respAttrs, corr)
			logger.LogAttrs(ctx, slog.LevelInfo, "http.response", respAttrs...)
		})
	}
}

// appendCorrelation appends trace_id/span_id only when they are present.
// This keeps the no-tracer path free of empty-string noise that would
// otherwise pollute every log line.
func appendCorrelation(attrs []slog.Attr, c Correlation) []slog.Attr {
	if c.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", c.TraceID))
	}
	if c.SpanID != "" {
		attrs = append(attrs, slog.String("span_id", c.SpanID))
	}
	return attrs
}
