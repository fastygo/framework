package imap_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	frameworkimap "github.com/fastygo/framework/pkg/mail/imap"
)

func TestClientSearch(t *testing.T) {
	harness := newIMAPHarness(t)
	harness.append(t, "INBOX", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 09:00:00 +0000",
		"From: Ada <ada@example.com>",
		"To: Bob <bob@example.com>",
		"Subject: unique-search-token alpha",
		"Message-ID: <search-a@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"body one",
		"",
	}, "\r\n")))
	harness.append(t, "Sent", []byte(strings.Join([]string{
		"Date: Tue, 28 Jul 2026 10:00:00 +0000",
		"From: Ada <ada@example.com>",
		"To: Carol <carol@example.com>",
		"Subject: unique-search-token beta",
		"Message-ID: <search-b@example.com>",
		"MIME-Version: 1.0",
		"Content-Type: multipart/mixed; boundary=b",
		"",
		"--b",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"body two",
		"--b",
		"Content-Type: application/octet-stream",
		"Content-Disposition: attachment; filename=note.txt",
		"Content-Transfer-Encoding: base64",
		"",
		"YQ==",
		"--b--",
		"",
	}, "\r\n")))

	client, err := frameworkimap.New(context.Background(), harness.options)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if !client.Capabilities().Search {
		t.Fatal("expected Capabilities.Search")
	}
	var _ mail.Searcher = client

	page, err := client.Search(context.Background(), mail.SearchQuery{
		Subject: "unique-search-token",
	}, mail.ListOptions{})
	if err != nil {
		t.Fatalf("Search subject: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("subject search: total=%d items=%d", page.Total, len(page.Items))
	}
	// Newest first by default.
	if !strings.Contains(page.Items[0].Envelope.Subject, "beta") {
		t.Fatalf("expected newest first, got %q", page.Items[0].Envelope.Subject)
	}

	inboxOnly, err := client.Search(context.Background(), mail.SearchQuery{
		Subject:   "unique-search-token",
		MailboxID: "INBOX",
	}, mail.ListOptions{})
	if err != nil {
		t.Fatalf("Search mailbox: %v", err)
	}
	if inboxOnly.Total != 1 || len(inboxOnly.Items) != 1 {
		t.Fatalf("mailbox filter: %+v", inboxOnly)
	}

	fromAda, err := client.Search(context.Background(), mail.SearchQuery{
		From: "ada@example.com",
		Text: "unique-search-token",
	}, mail.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search from+text: %v", err)
	}
	if fromAda.Total < 1 {
		t.Fatalf("from+text expected hits, got %+v", fromAda)
	}

	withAttach, err := client.Search(context.Background(), mail.SearchQuery{
		Subject:       "unique-search-token",
		HasAttachment: true,
	}, mail.ListOptions{})
	if err != nil {
		t.Fatalf("Search hasAttachment: %v", err)
	}
	if withAttach.Total != 1 {
		t.Fatalf("hasAttachment: total=%d", withAttach.Total)
	}

	after := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	empty, err := client.Search(context.Background(), mail.SearchQuery{
		Subject: "unique-search-token",
		After:   after,
	}, mail.ListOptions{})
	if err != nil {
		t.Fatalf("Search after: %v", err)
	}
	if empty.Total != 0 {
		t.Fatalf("after filter should exclude fixtures, got %d", empty.Total)
	}
}
