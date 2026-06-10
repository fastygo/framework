package mail

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"testing"
)

func TestAddressString(t *testing.T) {
	cases := []struct {
		addr Address
		want string
	}{
		{Address{Email: "ada@example.com"}, "ada@example.com"},
		{Address{Name: "Ada", Email: "ada@example.com"}, "Ada <ada@example.com>"},
	}
	for _, c := range cases {
		if got := c.addr.String(); got != c.want {
			t.Errorf("String(): got %q, want %q", got, c.want)
		}
	}
}

func TestFlagSet(t *testing.T) {
	s := NewFlagSet(FlagSeen, FlagFlagged)
	if !s.Has(FlagSeen) || !s.Has(FlagFlagged) {
		t.Fatalf("expected seen+flagged present: %v", s)
	}
	if s.Has(FlagDraft) {
		t.Fatalf("draft must be absent")
	}
	names := s.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != FlagFlagged || names[1] != FlagSeen {
		t.Fatalf("Names(): got %v", names)
	}
}

func TestFindByRole(t *testing.T) {
	boxes := []Mailbox{
		{ID: "a", Role: RoleNone, Name: "Projects"},
		{ID: "b", Role: RoleInbox, Name: "Inbox"},
		{ID: "c", Role: RoleTrash, Name: "Trash"},
	}
	if got := FindByRole(boxes, RoleInbox); got == nil || got.ID != "b" {
		t.Fatalf("FindByRole(inbox): got %+v", got)
	}
	if got := FindByRole(boxes, RoleSent); got != nil {
		t.Fatalf("FindByRole(sent): expected nil, got %+v", got)
	}
}

func TestErrorFormatting(t *testing.T) {
	base := errors.New("boom")
	err := &Error{Op: "jmap: Email/query", Code: CodeAuth, Err: base}
	if got := err.Error(); got != "mail: jmap: Email/query: auth: boom" {
		t.Errorf("Error(): got %q", got)
	}
	if !errors.Is(err, base) {
		t.Errorf("Unwrap chain broken")
	}
	noCause := &Error{Op: "x", Code: CodeNotFound}
	if got := noCause.Error(); got != "mail: x: not_found" {
		t.Errorf("Error() without cause: got %q", got)
	}
}

func TestCodeOf(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", &Error{Op: "op", Code: CodeRateLimit})
	if got := CodeOf(wrapped); got != CodeRateLimit {
		t.Errorf("CodeOf(wrapped): got %q", got)
	}
	if got := CodeOf(errors.New("plain")); got != CodeUnavailable {
		t.Errorf("CodeOf(plain): got %q, want unavailable", got)
	}
}

func TestRedactEmail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ada@example.com", "a***@example.com"},
		{"a@b.c", "a***@b.c"},
		{"", "***"},
		{"no-at-sign", "***"},
		{"@example.com", "***"},
		{"trailing@", "***"},
	}
	for _, c := range cases {
		if got := RedactEmail(c.in); got != c.want {
			t.Errorf("RedactEmail(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRedactSubject(t *testing.T) {
	if got := RedactSubject(""); got != "(empty)" {
		t.Errorf("empty subject: got %q", got)
	}
	if got := RedactSubject("Quarterly results"); got != "(redacted)" {
		t.Errorf("subject: got %q", got)
	}
}

func TestBasicAuth(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://mail.test/", nil)
	if err := (BasicAuth{Username: "ada@example.com", Password: "pw"}).Authenticate(req); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	user, pass, ok := req.BasicAuth()
	if !ok || user != "ada@example.com" || pass != "pw" {
		t.Fatalf("BasicAuth header: %q %q %v", user, pass, ok)
	}
}

func TestTokenAuth(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://mail.test/", nil)
	auth := TokenAuth{Token: func() (string, error) { return "tok-123", nil }}
	if err := auth.Authenticate(req); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer tok-123" {
		t.Errorf("Authorization: got %q", got)
	}

	failing := TokenAuth{Token: func() (string, error) { return "", errors.New("expired") }}
	if err := failing.Authenticate(req); err == nil {
		t.Fatalf("expected token error")
	}
}
