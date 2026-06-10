// Package mail defines a transport-agnostic email client contract and the
// domain types shared by all mail transports.
//
// The package is the core for building webmail applications on top of the
// framework. It deliberately contains no HTTP routing, no session storage,
// no caching policy, and no UI concerns — those belong to the consuming
// application. What it does own:
//
//   - domain types: Address, Envelope, Message, MessageSummary, Mailbox,
//     Draft, FlagSet, Capabilities;
//   - the Client interface every transport implements, plus the optional
//     capability interfaces Threader, Pusher, and Searcher discovered via
//     type assertion;
//   - typed errors (Error with an ErrorCode) so applications can branch on
//     failure classes without string matching;
//   - log redaction helpers that keep addresses, subjects, and credentials
//     out of structured logs.
//
// # Transports
//
// Subpackage jmap implements the contract for JMAP servers (RFC 8620 and
// RFC 8621), with Stalwart as the reference backend. An IMAP+SMTP transport
// is planned as a separate subpackage behind the same interface (see
// docs/adr/0004-mail-package.md).
//
// # Security contract
//
//   - Message bodies are plain strings. HTML bodies are untrusted input;
//     sanitization is the renderer's responsibility. The package never
//     wraps bodies in template.HTML.
//   - Credentials are injected per request through an Authenticator and are
//     never stored or logged by the package.
//   - Attachments stream through io.ReadCloser and are never buffered
//     whole in memory.
//
// See docs/adr/0004-mail-package.md for the full threat model.
package mail
