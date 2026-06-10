package mail

import "time"

// Address is a single RFC 5322 mailbox: an optional display name plus the
// address itself ("Ada Lovelace" <ada@example.com>).
type Address struct {
	// Name is the optional display name. May be empty.
	Name string
	// Email is the address in user@domain form. Required.
	Email string
}

// String renders the address for display: "Name <email>" when a display
// name is present, otherwise just the bare address.
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

// Envelope carries the routing and identification headers of a message,
// shared by both the summary and full forms.
type Envelope struct {
	// From lists the message authors. Usually one entry.
	From []Address
	// To lists the primary recipients.
	To []Address
	// Cc lists the carbon-copy recipients.
	Cc []Address
	// Bcc lists the blind-copy recipients. Populated only for messages
	// in Sent/Drafts where the server exposes it.
	Bcc []Address
	// ReplyTo lists the addresses replies should be directed to.
	ReplyTo []Address
	// Subject is the decoded message subject.
	Subject string
	// Date is the message date (header date, falling back to the
	// server's received timestamp).
	Date time.Time
	// MessageID is the RFC 5322 Message-ID header value without angle
	// brackets. May be empty.
	MessageID string
	// InReplyTo holds the Message-ID values from the In-Reply-To header.
	InReplyTo []string
	// References holds the Message-ID values from the References header.
	References []string
}

// MessageSummary is the lightweight list-row form of a message: enough to
// render a mailbox listing without fetching bodies.
type MessageSummary struct {
	// ID is the transport-scoped opaque message identifier.
	ID string
	// ThreadID groups messages into a conversation when the transport
	// supports threading. Empty otherwise.
	ThreadID string
	// MailboxIDs lists the mailboxes containing this message. JMAP
	// messages can live in several mailboxes at once; IMAP messages
	// have exactly one.
	MailboxIDs []string
	// Envelope holds the routing headers.
	Envelope Envelope
	// Preview is a short plain-text excerpt of the body, suitable for
	// list rows. May be empty.
	Preview string
	// Flags is the set of keywords on the message.
	Flags FlagSet
	// Size is the message size in bytes as reported by the server.
	Size int64
	// HasAttachments reports whether the message carries attachments.
	HasAttachments bool
}

// Message is the full form: summary data plus decoded bodies and
// attachment metadata.
type Message struct {
	MessageSummary

	// TextBody is the decoded plain-text body. Empty when the message
	// has no text part.
	TextBody string
	// HTMLBody is the decoded HTML body. SECURITY: this is untrusted
	// input straight from the wire. It MUST be sanitized (or rendered
	// inside a sandboxed iframe) by the consumer before reaching a
	// browser. The package intentionally exposes it as a plain string.
	HTMLBody string
	// Attachments describes the downloadable parts. Content is fetched
	// separately via Client.Attachment and streamed.
	Attachments []AttachmentInfo
}

// AttachmentInfo describes one attachment part without its content.
type AttachmentInfo struct {
	// PartID identifies the part within its message for download.
	PartID string
	// Filename is the suggested file name. May be empty.
	Filename string
	// ContentType is the MIME type, e.g. "application/pdf".
	ContentType string
	// Size is the part size in bytes as reported by the server.
	Size int64
	// Inline reports whether the part is referenced from the HTML body
	// (Content-Disposition: inline with a Content-ID).
	Inline bool
}

// Page is one page of results plus paging metadata.
type Page[T any] struct {
	// Items holds the page contents in server-provided order.
	Items []T
	// Total is the total number of matching items when the server
	// reports it; -1 when unknown.
	Total int64
	// Offset is the absolute position of the first item.
	Offset int64
}
