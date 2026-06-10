package mail

import "io"

// Draft is an outgoing message under composition, handed to Client.Send.
type Draft struct {
	// From is the sender identity. Required; must match an identity
	// the account is allowed to send as.
	From Address
	// To lists the primary recipients. At least one recipient across
	// To, Cc, and Bcc is required.
	To []Address
	// Cc lists the carbon-copy recipients.
	Cc []Address
	// Bcc lists the blind-copy recipients.
	Bcc []Address
	// Subject is the message subject.
	Subject string
	// TextBody is the plain-text body. At least one of TextBody and
	// HTMLBody must be non-empty.
	TextBody string
	// HTMLBody is the optional HTML body. The caller is responsible
	// for only submitting markup it generated or sanitized.
	HTMLBody string
	// InReplyTo holds Message-ID values for the In-Reply-To header
	// when this draft replies to another message.
	InReplyTo []string
	// References holds Message-ID values for the References header.
	References []string
	// Attachments lists the parts to upload and attach.
	Attachments []OutgoingAttachment
}

// OutgoingAttachment is one attachment of a Draft. Content is streamed
// from Open at send time so large files are never held in memory.
type OutgoingAttachment struct {
	// Filename is the file name presented to recipients. Required.
	Filename string
	// ContentType is the MIME type, e.g. "image/png". Required.
	ContentType string
	// Open returns a fresh reader over the attachment content. It may
	// be called more than once (e.g. on retry); each call must return
	// an independent reader positioned at the start.
	Open func() (io.ReadCloser, error)
}

// SendResult reports the outcome of a successful Client.Send.
type SendResult struct {
	// MessageID is the RFC 5322 Message-ID assigned to the sent
	// message, without angle brackets. May be empty when the server
	// does not report it.
	MessageID string
	// SentMessageID is the transport-scoped ID of the copy saved to
	// the Sent mailbox, empty when the transport does not save one.
	SentMessageID string
}
