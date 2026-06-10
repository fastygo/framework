package mail

import "time"

// SortField selects the property message listings are ordered by.
type SortField string

// Sort fields supported by every transport.
const (
	// SortDate orders by message date (header date or receivedAt).
	SortDate SortField = "date"
	// SortSize orders by message size in bytes.
	SortSize SortField = "size"
	// SortFrom orders by the first From address.
	SortFrom SortField = "from"
	// SortSubject orders by subject.
	SortSubject SortField = "subject"
)

// ListOptions controls pagination and ordering of message listings. The
// zero value means: first page, default page size, newest first.
type ListOptions struct {
	// Offset is the absolute position to start from (0-based).
	Offset int64
	// Limit caps the number of returned items. Zero applies the
	// transport default (50); transports also clamp excessive values.
	Limit int
	// SortBy selects the sort property. Empty means SortDate.
	SortBy SortField
	// Ascending flips the sort direction. Default false (newest /
	// largest first).
	Ascending bool
	// UnreadOnly restricts results to messages without FlagSeen.
	UnreadOnly bool
}

// SearchQuery describes a full-mailbox search for transports implementing
// Searcher. Empty fields are ignored; set fields combine with AND.
type SearchQuery struct {
	// Text matches anywhere in the message (subject, bodies, names),
	// using the server's text-search semantics.
	Text string
	// From matches the From header.
	From string
	// To matches the To/Cc headers.
	To string
	// Subject matches the subject header.
	Subject string
	// MailboxID restricts the search to one mailbox. Empty searches
	// the whole account.
	MailboxID string
	// HasAttachment, when true, restricts to messages with attachments.
	HasAttachment bool
	// After restricts to messages dated after this time (exclusive).
	// Zero means unbounded.
	After time.Time
	// Before restricts to messages dated before this time (exclusive).
	// Zero means unbounded.
	Before time.Time
}
