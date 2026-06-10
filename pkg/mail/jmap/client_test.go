package jmap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

func TestNewValidatesOptions(t *testing.T) {
	if _, err := New(context.Background(), Options{Auth: mail.BasicAuth{}}); err == nil {
		t.Fatal("expected error without SessionURL")
	}
	if _, err := New(context.Background(), Options{SessionURL: "https://x"}); err == nil {
		t.Fatal("expected error without Auth")
	}
}

func TestSessionLoad(t *testing.T) {
	f := newFakeServer(t)
	client := f.connect(t)

	caps := client.Capabilities()
	if !caps.Threads || !caps.Search || !caps.Push {
		t.Errorf("capabilities: %+v", caps)
	}
	if caps.MaxUploadSize != 50000000 {
		t.Errorf("MaxUploadSize: got %d", caps.MaxUploadSize)
	}
	if client.account() != "a1" {
		t.Errorf("account: got %q", client.account())
	}
}

func TestSessionAuthFailure(t *testing.T) {
	f := newFakeServer(t)
	_, err := New(context.Background(), Options{
		SessionURL: f.srv.URL + "/.well-known/jmap",
		Auth:       mail.BasicAuth{Username: "ada@example.com", Password: "wrong"},
		HTTPClient: f.srv.Client(),
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if mail.CodeOf(err) != mail.CodeAuth {
		t.Errorf("code: got %q, want auth", mail.CodeOf(err))
	}
}

func TestSessionMissingCapability(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"capabilities":    map[string]any{wire.CapCore: map[string]any{}},
			"accounts":        map[string]any{"a1": map[string]any{}},
			"primaryAccounts": map[string]string{},
			"apiUrl":          "https://x/api",
		})
	}))
	defer srv.Close()

	_, err := New(context.Background(), Options{
		SessionURL: srv.URL,
		Auth:       mail.BasicAuth{Username: "u", Password: "p"},
		HTTPClient: srv.Client(),
	})
	if err == nil {
		t.Fatal("expected capability error")
	}
	if mail.CodeOf(err) != mail.CodeUnsupported {
		t.Errorf("code: got %q, want unsupported", mail.CodeOf(err))
	}
}

func TestSessionUnknownAccount(t *testing.T) {
	f := newFakeServer(t)
	_, err := New(context.Background(), Options{
		SessionURL: f.srv.URL + "/.well-known/jmap",
		Auth:       mail.BasicAuth{Username: f.user, Password: f.pass},
		HTTPClient: f.srv.Client(),
		AccountID:  "nope",
	})
	if err == nil {
		t.Fatal("expected account error")
	}
	if mail.CodeOf(err) != mail.CodeNotFound {
		t.Errorf("code: got %q, want not_found", mail.CodeOf(err))
	}
}

func TestMethodErrorMapping(t *testing.T) {
	f := newFakeServer(t)
	// No Mailbox/get handler registered -> harness emits unknownMethod.
	client := f.connect(t)

	_, err := client.Mailboxes(context.Background())
	if err == nil {
		t.Fatal("expected method error")
	}
	if mail.CodeOf(err) != mail.CodeUnsupported {
		t.Errorf("code: got %q, want unsupported", mail.CodeOf(err))
	}
	var me *mail.Error
	if !errors.As(err, &me) {
		t.Fatalf("expected *mail.Error, got %T", err)
	}
}

func TestDiscover(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"mail.example.com", "https://mail.example.com/.well-known/jmap", false},
		{"mail.example.com:8443", "https://mail.example.com:8443/.well-known/jmap", false},
		{"https://mail.example.com/anything?x=1", "https://mail.example.com/.well-known/jmap", false},
		{"http://insecure.example.com", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := Discover(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Discover(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Discover(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Discover(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHTTPStatusCodeMapping(t *testing.T) {
	cases := []struct {
		status int
		want   mail.ErrorCode
	}{
		{401, mail.CodeAuth},
		{403, mail.CodeAuth},
		{404, mail.CodeNotFound},
		{413, mail.CodeTooLarge},
		{429, mail.CodeRateLimit},
		{500, mail.CodeUnavailable},
		{503, mail.CodeUnavailable},
		{400, mail.CodeProtocol},
	}
	for _, c := range cases {
		if got := httpStatusCode(c.status); got != c.want {
			t.Errorf("httpStatusCode(%d): got %q, want %q", c.status, got, c.want)
		}
	}
}
