package mail

// Role identifies the special-use function of a mailbox, mapped from JMAP
// roles (RFC 8621 §2) or IMAP SPECIAL-USE attributes (RFC 6154).
type Role string

// Special-use mailbox roles.
const (
	// RoleNone marks a regular user folder without a special use.
	RoleNone Role = ""
	// RoleInbox is the mailbox new mail is delivered to.
	RoleInbox Role = "inbox"
	// RoleArchive holds messages archived out of the inbox.
	RoleArchive Role = "archive"
	// RoleDrafts holds unsent drafts.
	RoleDrafts Role = "drafts"
	// RoleSent holds copies of sent messages.
	RoleSent Role = "sent"
	// RoleTrash holds messages pending permanent deletion.
	RoleTrash Role = "trash"
	// RoleJunk holds messages classified as spam.
	RoleJunk Role = "junk"
)

// Mailbox is a single folder in the account.
type Mailbox struct {
	// ID is the transport-scoped opaque mailbox identifier.
	ID string
	// ParentID is the ID of the parent mailbox, empty for top level.
	ParentID string
	// Name is the display name of this mailbox (not the full path).
	Name string
	// Role is the special-use role, RoleNone for regular folders.
	Role Role
	// TotalMessages is the number of messages in the mailbox, -1 when
	// the server does not report it.
	TotalMessages int64
	// UnreadMessages is the number of unseen messages, -1 when the
	// server does not report it.
	UnreadMessages int64
	// SortOrder is the server-suggested display position; lower sorts
	// first. Zero when the server does not provide one.
	SortOrder int
}

// FindByRole returns the first mailbox with the given role, or nil.
func FindByRole(boxes []Mailbox, role Role) *Mailbox {
	for i := range boxes {
		if boxes[i].Role == role {
			return &boxes[i]
		}
	}
	return nil
}
