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
//   - jmap — JMAP RFC 8620/8621 (reference servers such as Stalwart)
//   - imap — IMAP + SMTP submit (A+B parity with jmap; see
//     .project/roadmap-mail-imap-smtp.md and ADR
//     .project/adr/0001-mail-transport-parity.md)
//
// Applications depend on Client (+ optional Threader/Pusher/Searcher), never
// on a specific server product. Consumer Gmail/Outlook OAuth is out of scope.
//
// Shared test fixtures live in subpackage mailtest.
//
// # Security contract
//
//   - Message bodies are plain strings. HTML bodies are untrusted input;
//     sanitization is the renderer's responsibility. The package never
//     wraps bodies in template.HTML.
//   - JMAP credentials enter through Authenticator; IMAP/SMTP use
//     username/password on the transport Options. Secrets are never
//     logged (see redact helpers).
//   - Attachment returns an io.ReadCloser. JMAP streams from the download
//     endpoint; IMAP may buffer one part in memory — enforce size policy
//     in the application (see .project/architecture-mail.md).
package mail
