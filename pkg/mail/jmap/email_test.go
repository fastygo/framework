package jmap

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

func registerMailboxes(f *fakeServer) {
	f.handlers["Mailbox/get"] = respond("Mailbox/get", map[string]any{
		"accountId": "a1",
		"state":     "mb1",
		"list": []map[string]any{
			{"id": "mb-inbox", "name": "Inbox", "role": "inbox", "totalEmails": 12, "unreadEmails": 3, "sortOrder": 1},
			{"id": "mb-drafts", "name": "Drafts", "role": "drafts", "totalEmails": 1, "unreadEmails": 0, "sortOrder": 2},
			{"id": "mb-sent", "name": "Sent", "role": "sent", "totalEmails": 4, "unreadEmails": 0, "sortOrder": 3},
			{"id": "mb-custom", "name": "Projects", "role": "", "parentId": "mb-inbox", "sortOrder": 10},
		},
	})
}

func TestMailboxes(t *testing.T) {
	f := newFakeServer(t)
	registerMailboxes(f)
	client := f.connect(t)

	boxes, err := client.Mailboxes(context.Background())
	if err != nil {
		t.Fatalf("Mailboxes: %v", err)
	}
	if len(boxes) != 4 {
		t.Fatalf("got %d mailboxes", len(boxes))
	}
	inbox := mail.FindByRole(boxes, mail.RoleInbox)
	if inbox == nil || inbox.ID != "mb-inbox" || inbox.UnreadMessages != 3 {
		t.Errorf("inbox: %+v", inbox)
	}
	custom := boxes[3]
	if custom.Role != mail.RoleNone || custom.ParentID != "mb-inbox" {
		t.Errorf("custom folder: %+v", custom)
	}
}

func TestMessages(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/query"] = respond("Email/query", map[string]any{
		"accountId": "a1",
		"ids":       []string{"e2", "e1"},
		"position":  0,
		"total":     2,
	})
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1",
		// Deliberately reversed relative to query order.
		"list": []map[string]any{
			fixtureEmail("e1", "t1", "mb-inbox", "Older message"),
			fixtureEmail("e2", "t1", "mb-inbox", "Newer message"),
		},
	})
	client := f.connect(t)

	page, err := client.Messages(context.Background(), "mb-inbox", mail.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("page: total=%d items=%d", page.Total, len(page.Items))
	}
	// Query order must be preserved (e2 first).
	if page.Items[0].ID != "e2" || page.Items[1].ID != "e1" {
		t.Errorf("order: got %s, %s", page.Items[0].ID, page.Items[1].ID)
	}
	first := page.Items[0]
	if first.Envelope.Subject != "Newer message" || first.Envelope.From[0].Email != "ada@example.com" {
		t.Errorf("summary mapping: %+v", first.Envelope)
	}
	if !first.Flags.Has(mail.FlagSeen) {
		t.Errorf("flags: %+v", first.Flags)
	}
	if first.Envelope.Date.IsZero() {
		t.Errorf("date not parsed")
	}

	// The batch must contain a back-reference, proving one round-trip.
	req := f.requests[len(f.requests)-1]
	if len(req.MethodCalls) != 2 {
		t.Fatalf("expected 2 batched calls, got %d", len(req.MethodCalls))
	}
	if !strings.Contains(string(req.MethodCalls[1].Args), `"#ids"`) {
		t.Errorf("Email/get must use a back-reference: %s", req.MethodCalls[1].Args)
	}
}

func TestMessagesUnreadFilter(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/query"] = respond("Email/query", map[string]any{
		"accountId": "a1", "ids": []string{}, "position": 0, "total": 0,
	})
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1", "list": []map[string]any{},
	})
	client := f.connect(t)

	if _, err := client.Messages(context.Background(), "mb-inbox", mail.ListOptions{UnreadOnly: true}); err != nil {
		t.Fatalf("Messages: %v", err)
	}
	req := f.requests[len(f.requests)-1]
	args := string(req.MethodCalls[0].Args)
	if !strings.Contains(args, `"notKeyword":"$seen"`) {
		t.Errorf("unread filter missing: %s", args)
	}
}

func TestMessage(t *testing.T) {
	f := newFakeServer(t)
	full := fixtureEmail("e1", "t1", "mb-inbox", "Hello")
	full["bodyValues"] = map[string]any{
		"p1": map[string]any{"value": "plain text body"},
		"p2": map[string]any{"value": "<p>html body</p>"},
	}
	full["textBody"] = []map[string]any{{"partId": "p1", "type": "text/plain"}}
	full["htmlBody"] = []map[string]any{{"partId": "p2", "type": "text/html"}}
	full["attachments"] = []map[string]any{
		{"partId": "p3", "blobId": "blob-9", "name": "report.pdf", "type": "application/pdf", "size": 12345, "disposition": "attachment"},
	}
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1",
		"list":      []map[string]any{full},
	})
	client := f.connect(t)

	msg, err := client.Message(context.Background(), "e1")
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if msg.TextBody != "plain text body" || msg.HTMLBody != "<p>html body</p>" {
		t.Errorf("bodies: text=%q html=%q", msg.TextBody, msg.HTMLBody)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0].PartID != "blob-9" || msg.Attachments[0].Filename != "report.pdf" {
		t.Errorf("attachments: %+v", msg.Attachments)
	}
}

func TestMessageNotFound(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1",
		"list":      []map[string]any{},
		"notFound":  []string{"missing"},
	})
	client := f.connect(t)

	_, err := client.Message(context.Background(), "missing")
	if mail.CodeOf(err) != mail.CodeNotFound {
		t.Errorf("code: got %q, want not_found", mail.CodeOf(err))
	}
}

func TestSetFlags(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/set"] = respond("Email/set", map[string]any{
		"accountId": "a1",
		"updated":   map[string]any{"e1": nil},
	})
	client := f.connect(t)

	err := client.SetFlags(context.Background(), []string{"e1"},
		mail.NewFlagSet(mail.FlagSeen), mail.NewFlagSet(mail.FlagFlagged))
	if err != nil {
		t.Fatalf("SetFlags: %v", err)
	}
	args := string(f.requests[len(f.requests)-1].MethodCalls[0].Args)
	if !strings.Contains(args, `"keywords/$seen":true`) {
		t.Errorf("add patch missing: %s", args)
	}
	if !strings.Contains(args, `"keywords/$flagged":null`) {
		t.Errorf("remove patch missing: %s", args)
	}

	// No-op short circuit must not hit the server.
	before := len(f.requests)
	if err := client.SetFlags(context.Background(), nil, nil, nil); err != nil {
		t.Fatalf("noop SetFlags: %v", err)
	}
	if len(f.requests) != before {
		t.Errorf("noop SetFlags performed a request")
	}
}

func TestMoveAndDelete(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/set"] = respond("Email/set", map[string]any{
		"accountId": "a1",
		"updated":   map[string]any{"e1": nil},
		"destroyed": []string{"e1"},
	})
	client := f.connect(t)

	if err := client.Move(context.Background(), []string{"e1"}, "mb-trash"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	args := string(f.requests[len(f.requests)-1].MethodCalls[0].Args)
	if !strings.Contains(args, `"mailboxIds":{"mb-trash":true}`) {
		t.Errorf("move args: %s", args)
	}

	if err := client.Delete(context.Background(), []string{"e1"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	args = string(f.requests[len(f.requests)-1].MethodCalls[0].Args)
	if !strings.Contains(args, `"destroy":["e1"]`) {
		t.Errorf("destroy args: %s", args)
	}
}

func TestSetErrorSurfaced(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/set"] = respond("Email/set", map[string]any{
		"accountId":  "a1",
		"notUpdated": map[string]any{"e1": map[string]string{"type": "notFound"}},
	})
	client := f.connect(t)

	err := client.Move(context.Background(), []string{"e1"}, "mb-x")
	if mail.CodeOf(err) != mail.CodeNotFound {
		t.Errorf("code: got %q, want not_found", mail.CodeOf(err))
	}
}

func TestAttachmentDownload(t *testing.T) {
	f := newFakeServer(t)
	full := fixtureEmail("e1", "t1", "mb-inbox", "With attachment")
	full["attachments"] = []map[string]any{
		{"partId": "p3", "blobId": "blob-9", "name": "a.txt", "type": "text/plain", "size": 5, "disposition": "attachment"},
	}
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1",
		"list":      []map[string]any{full},
	})
	f.blobs["blob-9"] = "hello"
	client := f.connect(t)

	rc, info, err := client.Attachment(context.Background(), "e1", "blob-9")
	if err != nil {
		t.Fatalf("Attachment: %v", err)
	}
	defer rc.Close()
	content, _ := io.ReadAll(rc)
	if string(content) != "hello" {
		t.Errorf("content: got %q", content)
	}
	if info.Filename != "a.txt" || info.ContentType != "text/plain" {
		t.Errorf("info: %+v", info)
	}

	// Unknown part must fail with not_found before any download.
	if _, _, err := client.Attachment(context.Background(), "e1", "blob-nope"); mail.CodeOf(err) != mail.CodeNotFound {
		t.Errorf("unknown part: got %q", mail.CodeOf(err))
	}
}

func TestSearchFilter(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Email/query"] = respond("Email/query", map[string]any{
		"accountId": "a1", "ids": []string{}, "position": 0, "total": 0,
	})
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1", "list": []map[string]any{},
	})
	client := f.connect(t)

	_, err := client.Search(context.Background(), mail.SearchQuery{
		Text:          "invoice",
		MailboxID:     "mb-inbox",
		HasAttachment: true,
	}, mail.ListOptions{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	var args map[string]any
	if err := json.Unmarshal(f.requests[len(f.requests)-1].MethodCalls[0].Args, &args); err != nil {
		t.Fatalf("parse args: %v", err)
	}
	filter := args["filter"].(map[string]any)
	if filter["operator"] != "AND" {
		t.Errorf("filter: %+v", filter)
	}
	if len(filter["conditions"].([]any)) != 3 {
		t.Errorf("conditions: %+v", filter["conditions"])
	}
}

func TestThread(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Thread/get"] = respond("Thread/get", map[string]any{
		"accountId": "a1",
		"list": []map[string]any{
			{"id": "t1", "emailIds": []string{"e1", "e2"}},
		},
	})
	older := fixtureEmail("e1", "t1", "mb-inbox", "Re: hello")
	older["receivedAt"] = "2026-05-01T10:00:00Z"
	older["sentAt"] = ""
	newer := fixtureEmail("e2", "t1", "mb-inbox", "Re: Re: hello")
	newer["receivedAt"] = "2026-06-01T10:00:00Z"
	newer["sentAt"] = ""
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1",
		// Newest first from server; Thread must re-sort oldest first.
		"list": []map[string]any{newer, older},
	})
	client := f.connect(t)

	msgs, err := client.Thread(context.Background(), "t1")
	if err != nil {
		t.Fatalf("Thread: %v", err)
	}
	if len(msgs) != 2 || msgs[0].ID != "e1" || msgs[1].ID != "e2" {
		t.Errorf("thread order: %v, %v", msgs[0].ID, msgs[1].ID)
	}
}

func TestThreadNotFound(t *testing.T) {
	f := newFakeServer(t)
	f.handlers["Thread/get"] = respond("Thread/get", map[string]any{
		"accountId": "a1",
		"list":      []map[string]any{},
		"notFound":  []string{"t-missing"},
	})
	f.handlers["Email/get"] = respond("Email/get", map[string]any{
		"accountId": "a1", "list": []map[string]any{},
	})
	client := f.connect(t)

	if _, err := client.Thread(context.Background(), "t-missing"); mail.CodeOf(err) != mail.CodeNotFound {
		t.Errorf("code: got %q, want not_found", mail.CodeOf(err))
	}
}

func TestExpandURITemplate(t *testing.T) {
	got := expandURITemplate("https://x/{accountId}/{blobId}/{name}?accept={type}", map[string]string{
		"accountId": "a1",
		"blobId":    "b 2",
		"name":      "file.txt",
		"type":      "text/plain",
	})
	want := "https://x/a1/b%202/file.txt?accept=text%2Fplain"
	if got != want {
		t.Errorf("expand: got %q, want %q", got, want)
	}
}

// Guard: unused import protection for wire in this file.
var _ = wire.CapCore
