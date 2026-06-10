package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// summaryProperties are the Email properties fetched for list rows.
var summaryProperties = []string{
	"id", "threadId", "mailboxIds", "keywords", "size",
	"receivedAt", "sentAt", "messageId", "inReplyTo", "references",
	"from", "to", "cc", "replyTo", "subject", "preview", "hasAttachment",
}

// fullProperties extends summaryProperties with body structures.
var fullProperties = append(append([]string{}, summaryProperties...),
	"bcc", "bodyValues", "textBody", "htmlBody", "attachments")

// defaultPageSize applies when ListOptions.Limit is zero.
const defaultPageSize = 50

// maxPageSize clamps excessive page sizes.
const maxPageSize = 500

// Messages implements mail.Client. It batches Email/query and Email/get
// in one round-trip using a back-reference.
func (c *Client) Messages(ctx context.Context, mailboxID string, opts mail.ListOptions) (mail.Page[mail.MessageSummary], error) {
	filter := map[string]any{"inMailbox": mailboxID}
	if opts.UnreadOnly {
		filter = map[string]any{
			"operator":   "AND",
			"conditions": []any{filter, map[string]any{"notKeyword": mail.FlagSeen}},
		}
	}
	return c.queryMessages(ctx, "jmap: Email/query", filter, opts)
}

// queryMessages runs the shared Email/query + Email/get batch for both
// mailbox listings and search.
func (c *Client) queryMessages(ctx context.Context, op string, filter map[string]any, opts mail.ListOptions) (mail.Page[mail.MessageSummary], error) {
	var page mail.Page[mail.MessageSummary]

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	account := c.account()
	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail},
		call{
			method: "Email/query",
			args: map[string]any{
				"accountId":      account,
				"filter":         filter,
				"sort":           []map[string]any{sortComparator(opts)},
				"position":       opts.Offset,
				"limit":          limit,
				"calculateTotal": true,
			},
		},
		call{
			method: "Email/get",
			args: map[string]any{
				"accountId": account,
				"#ids": map[string]any{
					"resultOf": "c0",
					"name":     "Email/query",
					"path":     "/ids",
				},
				"properties": summaryProperties,
			},
		},
	)
	if err != nil {
		return page, err
	}

	var query wire.QueryResponse
	if err := c.result(op, resp, "c0", "Email/query", &query); err != nil {
		return page, err
	}
	var get wire.GetResponse
	if err := c.result(op, resp, "c1", "Email/get", &get); err != nil {
		return page, err
	}

	var emails []wire.Email
	if err := json.Unmarshal(get.List, &emails); err != nil {
		return page, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	// Email/get returns objects in undefined order; restore query order.
	byID := make(map[string]*wire.Email, len(emails))
	for i := range emails {
		byID[emails[i].ID] = &emails[i]
	}
	items := make([]mail.MessageSummary, 0, len(query.IDs))
	for _, id := range query.IDs {
		if em, ok := byID[id]; ok {
			items = append(items, toSummary(em))
		}
	}

	page.Items = items
	page.Offset = query.Position
	page.Total = query.Total
	if page.Total == 0 && len(items) > 0 {
		page.Total = -1
	}
	return page, nil
}

// sortComparator builds the JMAP sort comparator from ListOptions.
func sortComparator(opts mail.ListOptions) map[string]any {
	property := "receivedAt"
	switch opts.SortBy {
	case mail.SortSize:
		property = "size"
	case mail.SortFrom:
		property = "from"
	case mail.SortSubject:
		property = "subject"
	}
	return map[string]any{"property": property, "isAscending": opts.Ascending}
}

// Message implements mail.Client.
func (c *Client) Message(ctx context.Context, id string) (*mail.Message, error) {
	const op = "jmap: Email/get"

	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail}, call{
		method: "Email/get",
		args: map[string]any{
			"accountId":           c.account(),
			"ids":                 []string{id},
			"properties":          fullProperties,
			"fetchTextBodyValues": true,
			"fetchHTMLBodyValues": true,
			"maxBodyValueBytes":   c.maxBody,
		},
	})
	if err != nil {
		return nil, err
	}

	var get wire.GetResponse
	if err := c.result(op, resp, "c0", "Email/get", &get); err != nil {
		return nil, err
	}
	var emails []wire.Email
	if err := json.Unmarshal(get.List, &emails); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}
	if len(emails) == 0 {
		return nil, &mail.Error{Op: op, Code: mail.CodeNotFound, Err: fmt.Errorf("message %q", id)}
	}

	em := &emails[0]
	msg := &mail.Message{MessageSummary: toSummary(em)}
	msg.TextBody = joinBodyValues(em, em.TextBody)
	msg.HTMLBody = joinBodyValues(em, em.HTMLBody)
	msg.Attachments = make([]mail.AttachmentInfo, 0, len(em.Attachments))
	for _, part := range em.Attachments {
		msg.Attachments = append(msg.Attachments, mail.AttachmentInfo{
			PartID:      part.BlobID,
			Filename:    part.Name,
			ContentType: part.Type,
			Size:        part.Size,
			Inline:      part.Disposition == "inline" && part.CID != "",
		})
	}
	return msg, nil
}

// SetFlags implements mail.Client.
func (c *Client) SetFlags(ctx context.Context, ids []string, add, remove mail.FlagSet) error {
	const op = "jmap: Email/set (flags)"
	if len(ids) == 0 || (len(add) == 0 && len(remove) == 0) {
		return nil
	}

	patch := make(map[string]any, len(add)+len(remove))
	for flag, on := range add {
		if on {
			patch["keywords/"+flag] = true
		}
	}
	for flag, on := range remove {
		if on {
			patch["keywords/"+flag] = nil
		}
	}

	update := make(map[string]any, len(ids))
	for _, id := range ids {
		update[id] = patch
	}
	return c.emailSet(ctx, op, map[string]any{"update": update})
}

// Move implements mail.Client. JMAP move is a mailboxIds replacement.
func (c *Client) Move(ctx context.Context, ids []string, destMailboxID string) error {
	const op = "jmap: Email/set (move)"
	if len(ids) == 0 {
		return nil
	}

	update := make(map[string]any, len(ids))
	for _, id := range ids {
		update[id] = map[string]any{
			"mailboxIds": map[string]bool{destMailboxID: true},
		}
	}
	return c.emailSet(ctx, op, map[string]any{"update": update})
}

// Delete implements mail.Client. This is permanent destruction; moving to
// Trash is a Move.
func (c *Client) Delete(ctx context.Context, ids []string) error {
	const op = "jmap: Email/set (destroy)"
	if len(ids) == 0 {
		return nil
	}
	return c.emailSet(ctx, op, map[string]any{"destroy": ids})
}

// emailSet runs one Email/set call and surfaces per-object failures.
func (c *Client) emailSet(ctx context.Context, op string, args map[string]any) error {
	args["accountId"] = c.account()

	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail}, call{
		method: "Email/set",
		args:   args,
	})
	if err != nil {
		return err
	}

	var set wire.SetResponse
	if err := c.result(op, resp, "c0", "Email/set", &set); err != nil {
		return err
	}
	if err := firstSetError(op, set.NotUpdated); err != nil {
		return err
	}
	return firstSetError(op, set.NotDestroyed)
}

// toSummary maps a wire Email onto the domain summary.
func toSummary(em *wire.Email) mail.MessageSummary {
	mailboxes := make([]string, 0, len(em.MailboxIDs))
	for id, ok := range em.MailboxIDs {
		if ok {
			mailboxes = append(mailboxes, id)
		}
	}

	flags := make(mail.FlagSet, len(em.Keywords))
	for kw, ok := range em.Keywords {
		if ok {
			flags[kw] = true
		}
	}

	var msgID string
	if len(em.MessageID) > 0 {
		msgID = em.MessageID[0]
	}

	return mail.MessageSummary{
		ID:         em.ID,
		ThreadID:   em.ThreadID,
		MailboxIDs: mailboxes,
		Envelope: mail.Envelope{
			From:       toAddresses(em.From),
			To:         toAddresses(em.To),
			Cc:         toAddresses(em.Cc),
			Bcc:        toAddresses(em.Bcc),
			ReplyTo:    toAddresses(em.ReplyTo),
			Subject:    em.Subject,
			Date:       emailDate(em),
			MessageID:  msgID,
			InReplyTo:  em.InReplyTo,
			References: em.References,
		},
		Preview:        em.Preview,
		Flags:          flags,
		Size:           em.Size,
		HasAttachments: em.HasAttachment,
	}
}

// emailDate prefers the header date and falls back to receivedAt.
func emailDate(em *wire.Email) time.Time {
	for _, raw := range []string{em.SentAt, em.ReceivedAt} {
		if raw == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func toAddresses(in []wire.EmailAddress) []mail.Address {
	if len(in) == 0 {
		return nil
	}
	out := make([]mail.Address, len(in))
	for i, a := range in {
		out[i] = mail.Address{Name: a.Name, Email: a.Email}
	}
	return out
}

// joinBodyValues concatenates the decoded bodyValues for the given parts.
func joinBodyValues(em *wire.Email, parts []wire.EmailBodyPart) string {
	var out string
	for _, part := range parts {
		if val, ok := em.BodyValues[part.PartID]; ok {
			out += val.Value
		}
	}
	return out
}
