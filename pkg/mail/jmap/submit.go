package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// Send implements mail.Client. The flow per RFC 8621 §7: upload
// attachments as blobs, create the Email in Drafts, create an
// EmailSubmission, and use onSuccess hooks to move the message to Sent
// and flip its keywords.
func (c *Client) Send(ctx context.Context, draft mail.Draft) (*mail.SendResult, error) {
	const op = "jmap: EmailSubmission/set"

	if err := validateDraft(draft); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	identityID, err := c.findIdentity(ctx, draft.From.Email)
	if err != nil {
		return nil, err
	}

	boxes, err := c.Mailboxes(ctx)
	if err != nil {
		return nil, err
	}
	drafts := mail.FindByRole(boxes, mail.RoleDrafts)
	sent := mail.FindByRole(boxes, mail.RoleSent)
	if drafts == nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeUnsupported, Err: fmt.Errorf("account has no drafts mailbox")}
	}

	// Upload attachments first; each is streamed, never buffered.
	attachments := make([]map[string]any, 0, len(draft.Attachments))
	for _, att := range draft.Attachments {
		reader, err := att.Open()
		if err != nil {
			return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: fmt.Errorf("open attachment %q: %w", att.Filename, err)}
		}
		uploaded, upErr := c.uploadBlob(ctx, att.ContentType, reader)
		reader.Close()
		if upErr != nil {
			return nil, upErr
		}
		attachments = append(attachments, map[string]any{
			"blobId":      uploaded.BlobID,
			"type":        att.ContentType,
			"name":        att.Filename,
			"disposition": "attachment",
		})
	}

	emailObject := buildEmailObject(draft, drafts.ID, attachments)

	onSuccessUpdate := map[string]any{
		"keywords/" + mail.FlagSeen:  true,
		"keywords/" + mail.FlagDraft: nil,
	}
	if sent != nil {
		onSuccessUpdate["mailboxIds/"+drafts.ID] = nil
		onSuccessUpdate["mailboxIds/"+sent.ID] = true
	}

	account := c.account()
	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail, wire.CapSubmission},
		call{
			method: "Email/set",
			args: map[string]any{
				"accountId": account,
				"create":    map[string]any{"draft": emailObject},
			},
		},
		call{
			method: "EmailSubmission/set",
			args: map[string]any{
				"accountId": account,
				"create": map[string]any{
					"submission": map[string]any{
						"identityId": identityID,
						"emailId":    "#draft",
					},
				},
				"onSuccessUpdateEmail": map[string]any{
					"#submission": onSuccessUpdate,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	var emailSet wire.SetResponse
	if err := c.result(op, resp, "c0", "Email/set", &emailSet); err != nil {
		return nil, err
	}
	if err := firstSetError(op, emailSet.NotCreated); err != nil {
		return nil, err
	}
	var submissionSet wire.SetResponse
	if err := c.result(op, resp, "c1", "EmailSubmission/set", &submissionSet); err != nil {
		return nil, err
	}
	if err := firstSetError(op, submissionSet.NotCreated); err != nil {
		return nil, err
	}

	result := &mail.SendResult{}
	if raw, ok := emailSet.Created["draft"]; ok {
		var created struct {
			ID        string   `json:"id"`
			MessageID []string `json:"messageId"`
		}
		if json.Unmarshal(raw, &created) == nil {
			result.SentMessageID = created.ID
			if len(created.MessageID) > 0 {
				result.MessageID = created.MessageID[0]
			}
		}
	}

	auditEvent(ctx, slog.LevelInfo, "message_sent",
		slog.String("from", mail.RedactEmail(draft.From.Email)),
		slog.Int("recipients", len(draft.To)+len(draft.Cc)+len(draft.Bcc)),
		slog.Int("attachments", len(draft.Attachments)))
	return result, nil
}

// validateDraft enforces the Draft contract before any network call.
func validateDraft(draft mail.Draft) error {
	if draft.From.Email == "" {
		return fmt.Errorf("draft has no From address")
	}
	if len(draft.To)+len(draft.Cc)+len(draft.Bcc) == 0 {
		return fmt.Errorf("draft has no recipients")
	}
	if draft.TextBody == "" && draft.HTMLBody == "" {
		return fmt.Errorf("draft has no body")
	}
	for _, att := range draft.Attachments {
		if att.Filename == "" || att.ContentType == "" || att.Open == nil {
			return fmt.Errorf("attachment missing filename, content type, or Open")
		}
	}
	return nil
}

// buildEmailObject constructs the JMAP Email creation object for a draft.
func buildEmailObject(draft mail.Draft, draftsMailboxID string, attachments []map[string]any) map[string]any {
	bodyValues := map[string]any{}
	var textBody, htmlBody []map[string]any
	if draft.TextBody != "" {
		bodyValues["text"] = map[string]any{"value": draft.TextBody}
		textBody = []map[string]any{{"partId": "text", "type": "text/plain"}}
	}
	if draft.HTMLBody != "" {
		bodyValues["html"] = map[string]any{"value": draft.HTMLBody}
		htmlBody = []map[string]any{{"partId": "html", "type": "text/html"}}
	}

	email := map[string]any{
		"mailboxIds": map[string]bool{draftsMailboxID: true},
		"keywords":   map[string]bool{mail.FlagDraft: true, mail.FlagSeen: true},
		"from":       fromAddresses([]mail.Address{draft.From}),
		"subject":    draft.Subject,
		"bodyValues": bodyValues,
	}
	if len(draft.To) > 0 {
		email["to"] = fromAddresses(draft.To)
	}
	if len(draft.Cc) > 0 {
		email["cc"] = fromAddresses(draft.Cc)
	}
	if len(draft.Bcc) > 0 {
		email["bcc"] = fromAddresses(draft.Bcc)
	}
	if len(draft.InReplyTo) > 0 {
		email["inReplyTo"] = draft.InReplyTo
	}
	if len(draft.References) > 0 {
		email["references"] = draft.References
	}
	if textBody != nil {
		email["textBody"] = textBody
	}
	if htmlBody != nil {
		email["htmlBody"] = htmlBody
	}
	if len(attachments) > 0 {
		email["attachments"] = attachments
	}
	return email
}

// findIdentity resolves the sending identity matching the From address.
func (c *Client) findIdentity(ctx context.Context, fromEmail string) (string, error) {
	const op = "jmap: Identity/get"

	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail, wire.CapSubmission}, call{
		method: "Identity/get",
		args:   map[string]any{"accountId": c.account(), "ids": nil},
	})
	if err != nil {
		return "", err
	}

	var get wire.GetResponse
	if err := c.result(op, resp, "c0", "Identity/get", &get); err != nil {
		return "", err
	}
	var identities []wire.Identity
	if err := json.Unmarshal(get.List, &identities); err != nil {
		return "", &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	for _, id := range identities {
		if id.Email == fromEmail {
			return id.ID, nil
		}
	}
	if len(identities) > 0 {
		// Fall back to the first identity; servers commonly accept it
		// when the account owns the From address via an alias.
		return identities[0].ID, nil
	}
	return "", &mail.Error{Op: op, Code: mail.CodeAuth, Err: fmt.Errorf("no sending identity for %s", mail.RedactEmail(fromEmail))}
}

// fromAddresses maps domain addresses to wire shape.
func fromAddresses(in []mail.Address) []map[string]string {
	out := make([]map[string]string, len(in))
	for i, a := range in {
		entry := map[string]string{"email": a.Email}
		if a.Name != "" {
			entry["name"] = a.Name
		}
		out[i] = entry
	}
	return out
}
