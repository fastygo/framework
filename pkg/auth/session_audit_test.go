package auth

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type auditPayload struct {
	Event   string `json:"event"`
	Cookie  string `json:"cookie"`
	Reason  string `json:"reason,omitempty"`
	Level   string `json:"level"`
	Message string `json:"msg"`
}

// captureAudit installs a JSON slog handler at debug level for the
// duration of fn and returns parsed audit entries (only "auth.audit"
// messages, in emission order).
func captureAudit(t *testing.T, fn func()) []auditPayload {
	t.Helper()
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	fn()

	var out []auditPayload
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var entry auditPayload
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("parse audit line %q: %v", line, err)
		}
		if entry.Message == "auth.audit" {
			out = append(out, entry)
		}
	}
	return out
}

type fakePayload struct {
	UserID string `json:"u"`
}

func newSession() CookieSession[fakePayload] {
	return CookieSession[fakePayload]{
		Name:   "test_session",
		Secret: "test-secret-must-be-long-enough",
		TTL:    1 * time.Hour,
	}
}

func TestSession_AuditOnIssueSuccess(t *testing.T) {
	events := captureAudit(t, func() {
		rec := httptest.NewRecorder()
		if err := newSession().Issue(rec, fakePayload{UserID: "alice"}); err != nil {
			t.Fatalf("Issue: %v", err)
		}
	})

	if len(events) != 1 || events[0].Event != "session_issued" {
		t.Fatalf("expected one session_issued event, got %+v", events)
	}
	if events[0].Cookie != "test_session" {
		t.Errorf("cookie = %q, want test_session", events[0].Cookie)
	}
}

func TestSession_AuditOnIssueMissingSecret(t *testing.T) {
	events := captureAudit(t, func() {
		s := newSession()
		s.Secret = ""
		_ = s.Issue(httptest.NewRecorder(), fakePayload{})
	})

	if len(events) != 1 || events[0].Event != "session_issue_failed" {
		t.Fatalf("expected session_issue_failed, got %+v", events)
	}
	if events[0].Reason != "missing_secret" {
		t.Errorf("reason = %q, want missing_secret", events[0].Reason)
	}
	if events[0].Level != "WARN" {
		t.Errorf("level = %q, want WARN", events[0].Level)
	}
}

func TestSession_AuditOnTamper(t *testing.T) {
	events := captureAudit(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "test_session", Value: "garbage.value"})

		_, ok := newSession().Read(req)
		if ok {
			t.Fatal("Read should fail on garbage cookie")
		}
	})

	if len(events) != 1 || events[0].Event != "session_tampered" {
		t.Fatalf("expected session_tampered, got %+v", events)
	}
	if events[0].Level != "WARN" {
		t.Errorf("level = %q, want WARN", events[0].Level)
	}
}

func TestSession_AuditOnExpired(t *testing.T) {
	s := newSession()
	s.TTL = 1 * time.Hour

	rec := httptest.NewRecorder()
	if err := s.Issue(rec, fakePayload{UserID: "alice"}); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	cookie := rec.Result().Cookies()[0]

	// Manually craft an expired envelope by re-encoding with past Exp.
	// Easier: build a fresh CookieSession with negative TTL and capture.
	expired, err := SignedEncode(envelope[fakePayload]{
		Data: fakePayload{UserID: "bob"},
		Exp:  time.Now().Add(-time.Hour).Unix(),
	}, s.Secret)
	if err != nil {
		t.Fatalf("SignedEncode: %v", err)
	}

	events := captureAudit(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: expired})

		if _, ok := s.Read(req); ok {
			t.Fatal("Read should reject expired session")
		}
	})

	found := false
	for _, e := range events {
		if e.Event == "session_expired" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected session_expired, got %+v", events)
	}
}

func TestSession_AuditOnMissingCookie(t *testing.T) {
	events := captureAudit(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_, _ = newSession().Read(req)
	})

	if len(events) != 1 || events[0].Event != "session_missing" {
		t.Fatalf("expected session_missing, got %+v", events)
	}
}

func TestSession_AuditOnClear(t *testing.T) {
	events := captureAudit(t, func() {
		newSession().Clear(httptest.NewRecorder())
	})

	if len(events) != 1 || events[0].Event != "session_cleared" {
		t.Fatalf("expected session_cleared, got %+v", events)
	}
}
