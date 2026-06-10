package wire

import "encoding/json"

// Mailbox is the JMAP Mailbox object (RFC 8621 §2).
type Mailbox struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParentID      string `json:"parentId"`
	Role          string `json:"role"`
	SortOrder     int    `json:"sortOrder"`
	TotalEmails   int64  `json:"totalEmails"`
	UnreadEmails  int64  `json:"unreadEmails"`
	TotalThreads  int64  `json:"totalThreads"`
	UnreadThreads int64  `json:"unreadThreads"`
}

// EmailAddress is the JMAP EmailAddress object (RFC 8621 §4.1.2.3).
type EmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// EmailBodyPart is the JMAP EmailBodyPart object (RFC 8621 §4.1.4).
type EmailBodyPart struct {
	PartID      string `json:"partId"`
	BlobID      string `json:"blobId"`
	Size        int64  `json:"size"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Charset     string `json:"charset"`
	Disposition string `json:"disposition"`
	CID         string `json:"cid"`
}

// EmailBodyValue is one entry of the bodyValues map (RFC 8621 §4.1.4).
type EmailBodyValue struct {
	Value             string `json:"value"`
	IsEncodingProblem bool   `json:"isEncodingProblem"`
	IsTruncated       bool   `json:"isTruncated"`
}

// Email is the JMAP Email object (RFC 8621 §4) restricted to the
// properties the transport requests.
type Email struct {
	ID            string                    `json:"id"`
	BlobID        string                    `json:"blobId"`
	ThreadID      string                    `json:"threadId"`
	MailboxIDs    map[string]bool           `json:"mailboxIds"`
	Keywords      map[string]bool           `json:"keywords"`
	Size          int64                     `json:"size"`
	ReceivedAt    string                    `json:"receivedAt"`
	SentAt        string                    `json:"sentAt"`
	MessageID     []string                  `json:"messageId"`
	InReplyTo     []string                  `json:"inReplyTo"`
	References    []string                  `json:"references"`
	From          []EmailAddress            `json:"from"`
	To            []EmailAddress            `json:"to"`
	Cc            []EmailAddress            `json:"cc"`
	Bcc           []EmailAddress            `json:"bcc"`
	ReplyTo       []EmailAddress            `json:"replyTo"`
	Subject       string                    `json:"subject"`
	Preview       string                    `json:"preview"`
	HasAttachment bool                      `json:"hasAttachment"`
	TextBody      []EmailBodyPart           `json:"textBody"`
	HTMLBody      []EmailBodyPart           `json:"htmlBody"`
	Attachments   []EmailBodyPart           `json:"attachments"`
	BodyValues    map[string]EmailBodyValue `json:"bodyValues"`
}

// Identity is the JMAP Identity object (RFC 8621 §6).
type Identity struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetResponse is the generic Foo/get response shape.
type GetResponse struct {
	AccountID string          `json:"accountId"`
	State     string          `json:"state"`
	List      json.RawMessage `json:"list"`
	NotFound  []string        `json:"notFound"`
}

// QueryResponse is the generic Foo/query response shape.
type QueryResponse struct {
	AccountID  string   `json:"accountId"`
	QueryState string   `json:"queryState"`
	IDs        []string `json:"ids"`
	Position   int64    `json:"position"`
	Total      int64    `json:"total"`
}

// SetResponse is the generic Foo/set response shape.
type SetResponse struct {
	AccountID    string                     `json:"accountId"`
	Created      map[string]json.RawMessage `json:"created"`
	Updated      map[string]json.RawMessage `json:"updated"`
	Destroyed    []string                   `json:"destroyed"`
	NotCreated   map[string]SetError        `json:"notCreated"`
	NotUpdated   map[string]SetError        `json:"notUpdated"`
	NotDestroyed map[string]SetError        `json:"notDestroyed"`
}

// SetError is a per-object Foo/set failure (RFC 8620 §5.3).
type SetError struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// UploadResponse is the blob upload endpoint response (RFC 8620 §6.1).
type UploadResponse struct {
	AccountID string `json:"accountId"`
	BlobID    string `json:"blobId"`
	Type      string `json:"type"`
	Size      int64  `json:"size"`
}

// StateChange is the push notification object (RFC 8620 §7.1) delivered
// over EventSource.
type StateChange struct {
	Type    string                       `json:"@type"`
	Changed map[string]map[string]string `json:"changed"`
}
