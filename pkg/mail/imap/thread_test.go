package imap_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	frameworkimap "github.com/fastygo/framework/pkg/mail/imap"
)

func TestClientThread(t *testing.T) {
	harness := newIMAPHarness(t)
	harness.append(t, "INBOX", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 09:00:00 +0000",
		"From: Ada <ada@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: thread root",
		"Message-ID: <root-thread@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"root",
		"",
	}, "\r\n")))
	harness.append(t, "INBOX", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 10:00:00 +0000",
		"From: Bob <bob@example.com>",
		"To: Ada <ada@example.com>",
		"Subject: Re: thread root",
		"Message-ID: <reply1-thread@example.com>",
		"In-Reply-To: <root-thread@example.com>",
		"References: <root-thread@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"reply",
		"",
	}, "\r\n")))
	harness.append(t, "INBOX", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 11:00:00 +0000",
		"From: Ada <ada@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: other",
		"Message-ID: <other-thread@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"noise",
		"",
	}, "\r\n")))

	client, err := frameworkimap.New(context.Background(), harness.options)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if !client.Capabilities().Threads {
		t.Fatal("expected Capabilities.Threads")
	}
	var _ mail.Threader = client

	page, err := client.Messages(context.Background(), "INBOX", mail.ListOptions{Ascending: true})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	var threadID string
	for _, item := range page.Items {
		if item.Envelope.MessageID == "root-thread@example.com" {
			threadID = item.ThreadID
			break
		}
	}
	if threadID == "" {
		t.Fatalf("root ThreadID missing; items=%+v", page.Items)
	}

	thread, err := client.Thread(context.Background(), threadID)
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	if len(thread) != 2 {
		t.Fatalf("want 2 messages in thread, got %d (%+v)", len(thread), thread)
	}
	if thread[0].Envelope.MessageID != "root-thread@example.com" {
		t.Fatalf("oldest first: got %q", thread[0].Envelope.MessageID)
	}
	if thread[1].Envelope.MessageID != "reply1-thread@example.com" {
		t.Fatalf("reply: got %q", thread[1].Envelope.MessageID)
	}
}
