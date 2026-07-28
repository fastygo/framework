package imap

import (
	"context"
	"strings"

	goimap "github.com/emersion/go-imap/v2"
	"github.com/fastygo/framework/pkg/mail"
)

// Mailboxes implements mail.Client.
func (c *Client) Mailboxes(ctx context.Context) ([]mail.Mailbox, error) {
	const op = "imap: Mailboxes"
	unlock, err := c.lock(ctx, op)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return c.listMailboxesLocked(op)
}

func (c *Client) listMailboxesLocked(op string) ([]mail.Mailbox, error) {
	items, err := c.imap.List("", "*", &goimap.ListOptions{
		ReturnStatus: &goimap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
		},
		ReturnSpecialUse: true,
	}).Collect()
	if err != nil {
		return nil, wrapError(op, mail.CodeUnavailable, err)
	}

	boxes := make([]mail.Mailbox, 0, len(items))
	for _, item := range items {
		box := mail.Mailbox{
			ID:             item.Mailbox,
			Name:           item.Mailbox,
			Role:           roleFromList(item),
			TotalMessages:  -1,
			UnreadMessages: -1,
		}
		if item.Delim != 0 {
			if index := strings.LastIndex(item.Mailbox, string(item.Delim)); index >= 0 {
				box.ParentID = item.Mailbox[:index]
				box.Name = item.Mailbox[index+len(string(item.Delim)):]
			}
		}
		if item.Status != nil {
			if item.Status.NumMessages != nil {
				box.TotalMessages = int64(*item.Status.NumMessages)
			}
			if item.Status.NumUnseen != nil {
				box.UnreadMessages = int64(*item.Status.NumUnseen)
			}
		}
		boxes = append(boxes, box)
	}
	return boxes, nil
}

func roleFromList(item *goimap.ListData) mail.Role {
	for _, attr := range item.Attrs {
		switch {
		case strings.EqualFold(string(attr), string(goimap.MailboxAttrArchive)):
			return mail.RoleArchive
		case strings.EqualFold(string(attr), string(goimap.MailboxAttrDrafts)):
			return mail.RoleDrafts
		case strings.EqualFold(string(attr), string(goimap.MailboxAttrSent)):
			return mail.RoleSent
		case strings.EqualFold(string(attr), string(goimap.MailboxAttrTrash)):
			return mail.RoleTrash
		case strings.EqualFold(string(attr), string(goimap.MailboxAttrJunk)):
			return mail.RoleJunk
		}
	}
	// Conventional-name fallback covers servers that don't advertise
	// SPECIAL-USE (and the beta imapmemserver used by integration tests).
	switch {
	case strings.EqualFold(item.Mailbox, "INBOX"):
		return mail.RoleInbox
	case strings.EqualFold(item.Mailbox, "Archive"):
		return mail.RoleArchive
	case strings.EqualFold(item.Mailbox, "Drafts"):
		return mail.RoleDrafts
	case strings.EqualFold(item.Mailbox, "Sent"):
		return mail.RoleSent
	case strings.EqualFold(item.Mailbox, "Trash"):
		return mail.RoleTrash
	case strings.EqualFold(item.Mailbox, "Junk") || strings.EqualFold(item.Mailbox, "Spam"):
		return mail.RoleJunk
	default:
		return mail.RoleNone
	}
}
