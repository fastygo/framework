package mail

import (
	"context"
	"io"
	"net/http"
)

// Client is the transport-agnostic mail client contract. Implementations:
// jmap.Client (Stalwart and other JMAP servers); an IMAP+SMTP transport is
// planned (docs/adr/0004-mail-package.md).
//
// All methods are safe for concurrent use. Message and mailbox IDs are
// opaque transport-scoped strings; callers must not parse them.
//
// Optional capabilities (threading, push, search) are separate interfaces
// discovered via type assertion:
//
//	if t, ok := client.(mail.Threader); ok { ... }
type Client interface {
	// Mailboxes returns every mailbox in the account.
	Mailboxes(ctx context.Context) ([]Mailbox, error)

	// Messages returns one page of message summaries from a mailbox.
	Messages(ctx context.Context, mailboxID string, opts ListOptions) (Page[MessageSummary], error)

	// Message fetches one full message including decoded bodies and
	// attachment metadata (but not attachment content).
	Message(ctx context.Context, id string) (*Message, error)

	// Attachment streams the content of one attachment part. The caller
	// must close the reader. Content is never buffered by the client.
	Attachment(ctx context.Context, messageID, partID string) (io.ReadCloser, AttachmentInfo, error)

	// SetFlags adds and removes keywords on the given messages.
	SetFlags(ctx context.Context, ids []string, add, remove FlagSet) error

	// Move moves the given messages into the destination mailbox.
	Move(ctx context.Context, ids []string, destMailboxID string) error

	// Delete permanently destroys the given messages. Moving to Trash
	// is a Move to the RoleTrash mailbox, not a Delete.
	Delete(ctx context.Context, ids []string) error

	// Send submits a draft for delivery and, when the transport
	// supports it, saves a copy to the Sent mailbox.
	Send(ctx context.Context, draft Draft) (*SendResult, error)

	// Capabilities reports static server capabilities discovered at
	// connection time.
	Capabilities() Capabilities

	// Close releases the connection and any background resources.
	// The client must not be used afterwards.
	Close() error
}

// Threader is implemented by transports that group messages into
// conversations (JMAP Thread/get, or client-side threading for IMAP).
type Threader interface {
	// Thread returns every message in a conversation, oldest first.
	Thread(ctx context.Context, threadID string) ([]MessageSummary, error)
}

// ChangeEvent signals that mailbox state changed on the server. It
// intentionally carries no message content; consumers re-query.
type ChangeEvent struct {
	// MailboxID is the affected mailbox when known, empty for
	// account-wide changes.
	MailboxID string
}

// Pusher is implemented by transports that deliver server-side change
// notifications (JMAP EventSource, IMAP IDLE).
type Pusher interface {
	// Watch starts a notification stream. The returned channel closes
	// when ctx is cancelled or the stream fails permanently. The
	// goroutine behind the channel is owned by ctx — cancelling it is
	// sufficient cleanup.
	Watch(ctx context.Context) (<-chan ChangeEvent, error)
}

// Searcher is implemented by transports that support server-side search.
type Searcher interface {
	// Search returns one page of messages matching the query.
	Search(ctx context.Context, query SearchQuery, opts ListOptions) (Page[MessageSummary], error)
}

// Authenticator decorates outgoing HTTP requests with credentials. It is
// the only place credentials touch this package: implementations hold the
// secret, the package never stores or logs it.
type Authenticator interface {
	// Authenticate adds credentials to req (e.g. an Authorization
	// header). Called for every request; implementations backed by
	// expiring tokens should return fresh credentials each call.
	Authenticate(req *http.Request) error
}

// BasicAuth authenticates with HTTP Basic credentials (RFC 7617). Suitable
// for Stalwart app passwords and dev setups over TLS.
type BasicAuth struct {
	// Username is the account login, usually the email address.
	Username string
	// Password is the account or app password. Treat as PII.
	Password string
}

// Authenticate implements Authenticator.
func (b BasicAuth) Authenticate(req *http.Request) error {
	req.SetBasicAuth(b.Username, b.Password)
	return nil
}

// TokenAuth authenticates with a bearer token obtained by the caller
// (e.g. via pkg/auth OIDC). Token returns a currently valid token; it is
// invoked per request so refresh stays with the token source.
type TokenAuth struct {
	// Token returns a valid bearer token or an error. Required.
	Token func() (string, error)
}

// Authenticate implements Authenticator.
func (t TokenAuth) Authenticate(req *http.Request) error {
	tok, err := t.Token()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	return nil
}
