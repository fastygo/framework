package imap_test

import (
	"context"
	"net"
	"testing"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	frameworkimap "github.com/fastygo/framework/pkg/mail/imap"
)

const (
	testUsername = "test@example.com"
	testPassword = "test-password"
)

type imapHarness struct {
	address string
	options frameworkimap.Options
}

func newIMAPHarness(t *testing.T) *imapHarness {
	t.Helper()
	memory := imapmemserver.New()
	user := imapmemserver.NewUser(testUsername, testPassword)
	createMailbox(t, user, "INBOX", nil)
	createMailbox(t, user, "Drafts", []goimap.MailboxAttr{goimap.MailboxAttrDrafts})
	createMailbox(t, user, "Sent", []goimap.MailboxAttr{goimap.MailboxAttrSent})
	createMailbox(t, user, "Trash", []goimap.MailboxAttr{goimap.MailboxAttrTrash})
	memory.AddUser(user)

	server := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memory.NewSession(), nil, nil
		},
		Caps: goimap.CapSet{
			goimap.CapIMAP4rev1: {},
			goimap.CapIMAP4rev2: {},
			goimap.CapIdle:      {},
		},
		InsecureAuth: true,
	})
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("listen IMAP: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
		if err := <-serveDone; err != nil {
			t.Errorf("serve IMAP: %v", err)
		}
	})

	harness := &imapHarness{address: listener.Addr().String()}
	harness.options = frameworkimap.Options{
		IMAPHost: "localhost",
		Username: testUsername,
		Password: testPassword,
		DialIMAP: func(ctx context.Context) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "tcp", harness.address)
		},
	}
	return harness
}

func createMailbox(t *testing.T, user *imapmemserver.User, name string, specialUse []goimap.MailboxAttr) {
	t.Helper()
	var options *goimap.CreateOptions
	if len(specialUse) > 0 {
		options = &goimap.CreateOptions{SpecialUse: specialUse}
	}
	if err := user.Create(name, options); err != nil {
		t.Fatalf("create mailbox %s: %v", name, err)
	}
}

func (h *imapHarness) append(t *testing.T, mailbox string, raw []byte) goimap.UID {
	t.Helper()
	conn, err := net.Dial("tcp", h.address)
	if err != nil {
		t.Fatalf("dial raw IMAP: %v", err)
	}
	client := imapclient.New(conn, nil)
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("raw IMAP login: %v", err)
	}
	command := client.Append(mailbox, int64(len(raw)), nil)
	if _, err := command.Write(raw); err != nil {
		t.Fatalf("append write: %v", err)
	}
	if err := command.Close(); err != nil {
		t.Fatalf("append close: %v", err)
	}
	data, err := command.Wait()
	if err != nil {
		t.Fatalf("append wait: %v", err)
	}
	return data.UID
}
