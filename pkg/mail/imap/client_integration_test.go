package imap_test

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/fastygo/framework/pkg/mail"
	frameworkimap "github.com/fastygo/framework/pkg/mail/imap"
)

var fixtureMessage = []byte(strings.Join([]string{
	"Date: Tue, 28 Jul 2026 09:00:00 +0000",
	"From: Ada <ada@example.com>",
	"To: Bob <bob@example.com>",
	"Subject: IMAP integration fixture",
	"Message-ID: <fixture@example.com>",
	"MIME-Version: 1.0",
	"Content-Type: multipart/mixed; boundary=fixture-boundary",
	"",
	"--fixture-boundary",
	"Content-Type: text/plain; charset=utf-8",
	"",
	"Hello from IMAP.",
	"--fixture-boundary",
	"Content-Type: application/octet-stream",
	"Content-Disposition: attachment; filename=hello.txt",
	"Content-Transfer-Encoding: base64",
	"",
	"aGVsbG8gYXR0YWNobWVudA==",
	"--fixture-boundary--",
	"",
}, "\r\n"))

func TestClientIMAPLifecycle(t *testing.T) {
	harness := newIMAPHarness(t)
	client, err := frameworkimap.New(context.Background(), harness.options)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	caps := client.Capabilities()
	if !caps.Search || !caps.Push || caps.Threads || caps.MaxUploadSize != 0 || caps.MaxMessageSize != 0 {
		t.Fatalf("unexpected F3 capabilities: %+v", caps)
	}
	boxes, err := client.Mailboxes(context.Background())
	if err != nil {
		t.Fatalf("Mailboxes: %v", err)
	}
	for _, role := range []mail.Role{mail.RoleInbox, mail.RoleDrafts, mail.RoleSent, mail.RoleTrash} {
		if mail.FindByRole(boxes, role) == nil {
			t.Errorf("missing mailbox role %q", role)
		}
	}

	harness.append(t, "INBOX", fixtureMessage)
	page, err := client.Messages(context.Background(), "INBOX", mail.ListOptions{})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("Messages page: %+v", page)
	}
	summary := page.Items[0]
	if summary.Envelope.Subject != "IMAP integration fixture" || !summary.HasAttachments {
		t.Fatalf("summary: %+v", summary)
	}

	message, err := client.Message(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if !strings.Contains(message.TextBody, "Hello from IMAP.") {
		t.Errorf("TextBody: %q", message.TextBody)
	}
	if len(message.Attachments) != 1 || message.Attachments[0].PartID != "2" {
		t.Fatalf("attachments: %+v", message.Attachments)
	}
	attachment, info, err := client.Attachment(context.Background(), summary.ID, "2")
	if err != nil {
		t.Fatalf("Attachment: %v", err)
	}
	content, readErr := io.ReadAll(attachment)
	closeErr := attachment.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read attachment: read=%v close=%v", readErr, closeErr)
	}
	if string(content) != "hello attachment" || info.Filename != "hello.txt" {
		t.Fatalf("attachment: info=%+v content=%q", info, content)
	}

	if err := client.SetFlags(context.Background(), []string{summary.ID}, mail.NewFlagSet(mail.FlagSeen), nil); err != nil {
		t.Fatalf("SetFlags: %v", err)
	}
	unread, err := client.Messages(context.Background(), "INBOX", mail.ListOptions{UnreadOnly: true})
	if err != nil {
		t.Fatalf("Messages unread: %v", err)
	}
	if unread.Total != 0 {
		t.Fatalf("unread total: got %d, want 0", unread.Total)
	}

	if err := client.Move(context.Background(), []string{summary.ID}, "Trash"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	trash, err := client.Messages(context.Background(), "Trash", mail.ListOptions{})
	if err != nil {
		t.Fatalf("Messages Trash: %v", err)
	}
	if trash.Total != 1 || len(trash.Items) != 1 {
		t.Fatalf("Trash page: %+v", trash)
	}
	if err := client.Delete(context.Background(), []string{trash.Items[0].ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	trash, err = client.Messages(context.Background(), "Trash", mail.ListOptions{})
	if err != nil {
		t.Fatalf("Messages Trash after delete: %v", err)
	}
	if trash.Total != 0 {
		t.Fatalf("Trash total after delete: got %d, want 0", trash.Total)
	}
}

func TestClientSendSMTPAndAppendSent(t *testing.T) {
	harness := newIMAPHarness(t)
	smtpAddress, received := newSMTPHarness(t)
	options := harness.options
	options.DialSMTP = func(ctx context.Context) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "tcp", smtpAddress)
	}
	client, err := frameworkimap.New(context.Background(), options)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	result, err := client.Send(context.Background(), mail.Draft{
		From:     mail.Address{Name: "Sender", Email: testUsername},
		To:       []mail.Address{{Email: "recipient@example.com"}},
		Bcc:      []mail.Address{{Email: "hidden@example.com"}},
		Subject:  "SMTP integration fixture",
		TextBody: "plain body",
		HTMLBody: "<p>HTML body</p>",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.MessageID == "" || result.SentMessageID == "" {
		t.Fatalf("Send result: %+v", result)
	}

	select {
	case raw := <-received:
		text := string(raw)
		if !strings.Contains(text, "SMTP integration fixture") {
			t.Errorf("SMTP data omitted subject")
		}
		if strings.Contains(strings.ToLower(text), "\r\nbcc:") {
			t.Errorf("SMTP data leaked Bcc header")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SMTP server did not receive message")
	}

	sent, err := client.Messages(context.Background(), "Sent", mail.ListOptions{})
	if err != nil {
		t.Fatalf("Messages Sent: %v", err)
	}
	if sent.Total != 1 || len(sent.Items) != 1 {
		t.Fatalf("Sent page: %+v", sent)
	}
	saved, err := client.Message(context.Background(), result.SentMessageID)
	if err != nil {
		t.Fatalf("saved Message: %v", err)
	}
	if saved.Envelope.Subject != "SMTP integration fixture" || saved.HTMLBody != "<p>HTML body</p>" {
		t.Fatalf("saved message: %+v", saved)
	}
}

type smtpBackend struct {
	received chan []byte
}

func (b *smtpBackend) NewSession(*smtp.Conn) (smtp.Session, error) {
	return &smtpSession{backend: b}, nil
}

type smtpSession struct {
	backend *smtpBackend
	auth    bool
}

func (s *smtpSession) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *smtpSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, username, password string) error {
		if username != testUsername || password != testPassword {
			return smtp.ErrAuthFailed
		}
		s.auth = true
		return nil
	}), nil
}

func (s *smtpSession) Mail(string, *smtp.MailOptions) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	return nil
}

func (s *smtpSession) Rcpt(string, *smtp.RcptOptions) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	return nil
}

func (s *smtpSession) Data(reader io.Reader) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	raw, err := io.ReadAll(reader)
	if err == nil {
		s.backend.received <- raw
	}
	return err
}

func (*smtpSession) Reset() {}

func (*smtpSession) Logout() error { return nil }

func newSMTPHarness(t *testing.T) (string, <-chan []byte) {
	t.Helper()
	backend := &smtpBackend{received: make(chan []byte, 1)}
	server := smtp.NewServer(backend)
	server.Domain = "localhost"
	server.AllowInsecureAuth = true
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen SMTP: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		if err := server.Close(); err != nil && !errors.Is(err, smtp.ErrServerClosed) {
			t.Errorf("close SMTP: %v", err)
		}
		if err := <-done; err != nil {
			t.Errorf("serve SMTP: %v", err)
		}
	})
	return listener.Addr().String(), backend.received
}
