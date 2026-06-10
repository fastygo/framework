package jmap

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/fastygo/framework/pkg/mail"
)

func registerSendHandlers(f *fakeServer) {
	registerMailboxes(f)
	f.handlers["Identity/get"] = respond("Identity/get", map[string]any{
		"accountId": "a1",
		"list": []map[string]any{
			{"id": "id1", "name": "Ada", "email": "ada@example.com"},
		},
	})
	f.handlers["Email/set"] = respond("Email/set", map[string]any{
		"accountId": "a1",
		"created": map[string]any{
			"draft": map[string]any{"id": "e-new", "messageId": []string{"new@example.com"}},
		},
	})
	f.handlers["EmailSubmission/set"] = respond("EmailSubmission/set", map[string]any{
		"accountId": "a1",
		"created":   map[string]any{"submission": map[string]any{"id": "sub1"}},
	})
}

func TestSend(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	client := f.connect(t)

	result, err := client.Send(context.Background(), mail.Draft{
		From:     mail.Address{Name: "Ada", Email: "ada@example.com"},
		To:       []mail.Address{{Email: "bob@example.com"}},
		Subject:  "Hi",
		TextBody: "Hello Bob",
		HTMLBody: "<p>Hello Bob</p>",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.SentMessageID != "e-new" || result.MessageID != "new@example.com" {
		t.Errorf("result: %+v", result)
	}

	// Final batch: Email/set + EmailSubmission/set in one request.
	req := f.requests[len(f.requests)-1]
	if len(req.MethodCalls) != 2 {
		t.Fatalf("expected batched send, got %d calls", len(req.MethodCalls))
	}
	emailArgs := string(req.MethodCalls[0].Args)
	if !strings.Contains(emailArgs, `"mailboxIds":{"mb-drafts":true}`) {
		t.Errorf("draft mailbox: %s", emailArgs)
	}
	subArgs := string(req.MethodCalls[1].Args)
	if !strings.Contains(subArgs, `"emailId":"#draft"`) {
		t.Errorf("creation reference: %s", subArgs)
	}
	if !strings.Contains(subArgs, `"identityId":"id1"`) {
		t.Errorf("identity: %s", subArgs)
	}
	// onSuccess must move Drafts -> Sent.
	if !strings.Contains(subArgs, `"mailboxIds/mb-sent":true`) || !strings.Contains(subArgs, `"mailboxIds/mb-drafts":null`) {
		t.Errorf("onSuccess move: %s", subArgs)
	}
}

func TestSendWithAttachment(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	client := f.connect(t)

	_, err := client.Send(context.Background(), mail.Draft{
		From:     mail.Address{Email: "ada@example.com"},
		To:       []mail.Address{{Email: "bob@example.com"}},
		Subject:  "With file",
		TextBody: "see attachment",
		Attachments: []mail.OutgoingAttachment{{
			Filename:    "notes.txt",
			ContentType: "text/plain",
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("file-content")), nil
			},
		}},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(f.uploads) != 1 || f.uploads[0] != "file-content" {
		t.Fatalf("uploads: %v", f.uploads)
	}

	req := f.requests[len(f.requests)-1]
	var emailArgs map[string]any
	if err := json.Unmarshal(req.MethodCalls[0].Args, &emailArgs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	created := emailArgs["create"].(map[string]any)["draft"].(map[string]any)
	atts := created["attachments"].([]any)
	if len(atts) != 1 {
		t.Fatalf("attachments: %v", atts)
	}
	att := atts[0].(map[string]any)
	if att["blobId"] != "upload-1" || att["name"] != "notes.txt" {
		t.Errorf("attachment object: %v", att)
	}
}

func TestSendValidation(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	client := f.connect(t)
	ctx := context.Background()

	cases := []mail.Draft{
		{To: []mail.Address{{Email: "b@x.com"}}, TextBody: "x"},                        // no From
		{From: mail.Address{Email: "a@x.com"}, TextBody: "x"},                          // no recipients
		{From: mail.Address{Email: "a@x.com"}, To: []mail.Address{{Email: "b@x.com"}}}, // no body
		{From: mail.Address{Email: "a@x.com"}, To: []mail.Address{{Email: "b@x.com"}}, TextBody: "x",
			Attachments: []mail.OutgoingAttachment{{Filename: "f"}}}, // attachment missing fields
	}
	before := len(f.requests)
	for i, draft := range cases {
		if _, err := client.Send(ctx, draft); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
	if len(f.requests) != before {
		t.Errorf("validation failures must not reach the server")
	}
}

func TestSendNoIdentity(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	f.handlers["Identity/get"] = respond("Identity/get", map[string]any{
		"accountId": "a1",
		"list":      []map[string]any{},
	})
	client := f.connect(t)

	_, err := client.Send(context.Background(), mail.Draft{
		From:     mail.Address{Email: "ada@example.com"},
		To:       []mail.Address{{Email: "bob@example.com"}},
		TextBody: "x",
	})
	if mail.CodeOf(err) != mail.CodeAuth {
		t.Errorf("code: got %q, want auth", mail.CodeOf(err))
	}
}

func TestSendSubmissionRejected(t *testing.T) {
	f := newFakeServer(t)
	registerSendHandlers(f)
	f.handlers["EmailSubmission/set"] = respond("EmailSubmission/set", map[string]any{
		"accountId": "a1",
		"notCreated": map[string]any{
			"submission": map[string]string{"type": "forbiddenToSend", "description": "quota"},
		},
	})
	client := f.connect(t)

	_, err := client.Send(context.Background(), mail.Draft{
		From:     mail.Address{Email: "ada@example.com"},
		To:       []mail.Address{{Email: "bob@example.com"}},
		TextBody: "x",
	})
	if mail.CodeOf(err) != mail.CodeAuth {
		t.Errorf("code: got %q, want auth", mail.CodeOf(err))
	}
}
