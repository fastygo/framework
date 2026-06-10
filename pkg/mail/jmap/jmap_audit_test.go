package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
)

type auditPayload struct {
	Event   string `json:"event"`
	Message string `json:"msg"`
	Level   string `json:"level"`
}

// captureAudit installs a JSON slog handler for fn and returns the
// emitted mail.audit entries in order, alongside the raw log text.
func captureAudit(t *testing.T, fn func()) ([]auditPayload, string) {
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
		if entry.Message == "mail.audit" {
			out = append(out, entry)
		}
	}
	return out, buf.String()
}

func hasEvent(entries []auditPayload, event string) bool {
	for _, e := range entries {
		if e.Event == event {
			return true
		}
	}
	return false
}

func TestAuditSessionLoaded(t *testing.T) {
	f := newFakeServer(t)
	entries, _ := captureAudit(t, func() {
		_ = f.connect(t)
	})
	if !hasEvent(entries, "session_loaded") {
		t.Errorf("missing session_loaded: %+v", entries)
	}
}

func TestAuditAuthFailed(t *testing.T) {
	f := newFakeServer(t)
	entries, _ := captureAudit(t, func() {
		_, err := New(context.Background(), Options{
			SessionURL: f.srv.URL + "/.well-known/jmap",
			Auth:       mail.BasicAuth{Username: f.user, Password: "wrong"},
			HTTPClient: f.srv.Client(),
		})
		if err == nil {
			t.Error("expected auth failure")
		}
	})
	if !hasEvent(entries, "auth_failed") {
		t.Errorf("missing auth_failed: %+v", entries)
	}
}

func TestAuditInsecureTLS(t *testing.T) {
	entries, _ := captureAudit(t, func() {
		// Connection fails (no server), but the insecure_tls_enabled
		// event must fire before the dial.
		_, _ = New(context.Background(), Options{
			SessionURL:  "https://127.0.0.1:1/.well-known/jmap",
			Auth:        mail.BasicAuth{Username: "u", Password: "p"},
			InsecureTLS: true,
		})
	})
	if !hasEvent(entries, "insecure_tls_enabled") {
		t.Errorf("missing insecure_tls_enabled: %+v", entries)
	}
	for _, e := range entries {
		if e.Event == "insecure_tls_enabled" && e.Level != "WARN" {
			t.Errorf("insecure_tls_enabled level: got %s, want WARN", e.Level)
		}
	}
}

func TestAuditSendRedactsContent(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	client := f.connect(t)

	const secretSubject = "TOP-SECRET-SUBJECT-LINE"
	entries, raw := captureAudit(t, func() {
		_, err := client.Send(context.Background(), mail.Draft{
			From:     mail.Address{Email: "ada@example.com"},
			To:       []mail.Address{{Email: "bob@example.com"}},
			Subject:  secretSubject,
			TextBody: "body",
		})
		if err != nil {
			t.Errorf("Send: %v", err)
		}
	})
	if !hasEvent(entries, "message_sent") {
		t.Fatalf("missing message_sent: %+v", entries)
	}
	if strings.Contains(raw, secretSubject) {
		t.Errorf("audit log leaked the subject")
	}
	if strings.Contains(raw, "ada@example.com") {
		t.Errorf("audit log leaked the full sender address")
	}
}
