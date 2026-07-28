package imap_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	frameworkimap "github.com/fastygo/framework/pkg/mail/imap"
)

func TestClientWatchIDLE(t *testing.T) {
	harness := newIMAPHarness(t)
	client, err := frameworkimap.New(context.Background(), harness.options)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if !client.Capabilities().Push {
		t.Fatal("expected Capabilities.Push when IDLE is available")
	}
	var _ mail.Pusher = client

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := client.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Give IDLE a moment to start before appending.
	time.Sleep(50 * time.Millisecond)
	harness.append(t, "INBOX", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 11:00:00 +0000",
		"From: Ada <ada@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: idle notify",
		"Message-ID: <idle@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"ping",
		"",
	}, "\r\n")))

	select {
	case ev := <-events:
		if ev.MailboxID != "INBOX" {
			t.Fatalf("MailboxID: got %q", ev.MailboxID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for IDLE ChangeEvent")
	}

	cancel()
	select {
	case _, ok := <-events:
		if ok {
			// drain optional trailing event then wait for close
			select {
			case _, ok = <-events:
				if ok {
					t.Fatal("events channel still open after cancel")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("events channel did not close after cancel")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("events channel did not close after cancel")
	}
}
