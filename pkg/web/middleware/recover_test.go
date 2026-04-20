package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeForLog(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"clean", "abc-123", "abc-123"},
		{"newline", "abc\nINJECTED", "abc_INJECTED"},
		{"crlf", "abc\r\nINJECTED", "abc__INJECTED"},
		{"tab", "a\tb", "a_b"},
		{"control", "a\x01b\x7fc", "a_b_c"},
		{"unicode", "тест", "тест"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeForLog(tc.in)
			if got != tc.want {
				t.Errorf("sanitizeForLog(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRecoverMiddleware_LogsAndReturns500(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	handler := RecoverMiddleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Crafted X-Request-ID containing CRLF — must not appear in the log
	// untouched (see sanitizeForLog).
	req.Header.Set(RequestIDHeader, "rid-1\r\nfake.audit msg=injected")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want 500", rr.Code)
	}
	logged := buf.String()
	if !strings.Contains(logged, "http.panic") {
		t.Fatalf("expected http.panic log line, got %q", logged)
	}
	if strings.Contains(logged, "\r\nfake.audit") {
		t.Fatalf("CRLF was not sanitised in request_id, log: %q", logged)
	}
	if !strings.Contains(logged, "rid-1__fake.audit") {
		t.Fatalf("expected sanitised request_id, log: %q", logged)
	}
}
