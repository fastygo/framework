package mailtest

import (
	"time"

	"github.com/fastygo/framework/pkg/mail"
)

// FixedTime is a deterministic instant for transport tests.
func FixedTime() time.Time {
	return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
}

// RoleMailboxes returns a minimal personal mailbox set with standard roles.
// IDs are opaque fixtures — transports must not assume these wire formats.
func RoleMailboxes() []mail.Mailbox {
	return []mail.Mailbox{
		{ID: "mb-inbox", Name: "Inbox", Role: mail.RoleInbox, TotalMessages: 0, UnreadMessages: 0, SortOrder: 1},
		{ID: "mb-drafts", Name: "Drafts", Role: mail.RoleDrafts, TotalMessages: 0, UnreadMessages: 0, SortOrder: 2},
		{ID: "mb-sent", Name: "Sent", Role: mail.RoleSent, TotalMessages: 0, UnreadMessages: 0, SortOrder: 3},
		{ID: "mb-trash", Name: "Trash", Role: mail.RoleTrash, TotalMessages: 0, UnreadMessages: 0, SortOrder: 4},
		{ID: "mb-junk", Name: "Junk", Role: mail.RoleJunk, TotalMessages: 0, UnreadMessages: 0, SortOrder: 5},
		{ID: "mb-archive", Name: "Archive", Role: mail.RoleArchive, TotalMessages: 0, UnreadMessages: 0, SortOrder: 6},
	}
}

// SeenFlagSet is a FlagSet with FlagSeen set.
func SeenFlagSet() mail.FlagSet {
	return mail.NewFlagSet(mail.FlagSeen)
}

// DraftFlagSet is a FlagSet with FlagDraft set.
func DraftFlagSet() mail.FlagSet {
	return mail.NewFlagSet(mail.FlagDraft)
}

// SampleSummary returns a MessageSummary suitable for list-row tests.
func SampleSummary(mailboxID string) mail.MessageSummary {
	now := FixedTime()
	return mail.MessageSummary{
		ID:         "msg-1",
		ThreadID:   "thr-1",
		MailboxIDs: []string{mailboxID},
		Flags:      mail.NewFlagSet(),
		Size:       128,
		Envelope: mail.Envelope{
			From:    []mail.Address{{Name: "Ada", Email: "ada@example.com"}},
			To:      []mail.Address{{Email: "bob@example.com"}},
			Subject: "Hello",
			Date:    now,
		},
		Preview: "Hello from Ada",
	}
}

// SampleDraft returns a minimal outgoing draft for Send tests.
func SampleDraft() mail.Draft {
	return mail.Draft{
		From:     mail.Address{Name: "Ada", Email: "ada@example.com"},
		To:       []mail.Address{{Email: "bob@example.com"}},
		Subject:  "Hello",
		TextBody: "Hello from Ada\n",
	}
}
