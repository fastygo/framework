package jmap

import (
	"context"
	"encoding/json"

	"github.com/fastygo/framework/pkg/mail"
	"github.com/fastygo/framework/pkg/mail/jmap/internal/wire"
)

// Mailboxes implements mail.Client.
func (c *Client) Mailboxes(ctx context.Context) ([]mail.Mailbox, error) {
	const op = "jmap: Mailbox/get"

	resp, err := c.batch(ctx, op, []string{wire.CapCore, wire.CapMail}, call{
		method: "Mailbox/get",
		args: map[string]any{
			"accountId": c.account(),
			"ids":       nil,
		},
	})
	if err != nil {
		return nil, err
	}

	var get wire.GetResponse
	if err := c.result(op, resp, "c0", "Mailbox/get", &get); err != nil {
		return nil, err
	}

	var raw []wire.Mailbox
	if err := json.Unmarshal(get.List, &raw); err != nil {
		return nil, &mail.Error{Op: op, Code: mail.CodeProtocol, Err: err}
	}

	out := make([]mail.Mailbox, len(raw))
	for i, mb := range raw {
		out[i] = mail.Mailbox{
			ID:             mb.ID,
			ParentID:       mb.ParentID,
			Name:           mb.Name,
			Role:           mailboxRole(mb.Role),
			TotalMessages:  mb.TotalEmails,
			UnreadMessages: mb.UnreadEmails,
			SortOrder:      mb.SortOrder,
		}
	}
	return out, nil
}

// mailboxRole maps a JMAP role string onto the domain Role. Unknown roles
// degrade to RoleNone rather than failing.
func mailboxRole(role string) mail.Role {
	switch role {
	case "inbox":
		return mail.RoleInbox
	case "archive":
		return mail.RoleArchive
	case "drafts":
		return mail.RoleDrafts
	case "sent":
		return mail.RoleSent
	case "trash":
		return mail.RoleTrash
	case "junk", "spam":
		return mail.RoleJunk
	default:
		return mail.RoleNone
	}
}
