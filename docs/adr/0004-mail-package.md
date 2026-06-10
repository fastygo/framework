# 4. `pkg/mail` — transport-agnostic email client core

Date: 2026-06-10

## Status

Accepted.

## Context

FastyGo product applications need a webmail building block: a library
that can list mailboxes, page through messages, fetch bodies and
attachments, mutate flags, and send mail — against more than one kind
of mail server. The immediate target is **Stalwart** (JMAP, RFC 8620 /
RFC 8621); classic IMAP+SMTP servers (maddy, Dovecot) follow later.

Existing open-source webmail code (the LilMail snapshot kept under
`@BuildY/.project/.webmail`) was evaluated as a porting source and
rejected: it couples protocol access to a Fiber web layer, buffers
whole attachments in memory, leaks `template.HTML` into the domain
model, and pins the design to IMAP sequence semantics. It remains a
useful *feature checklist*, not a code base. `pkg/mail` is a clean
design.

Constraints inherited from the framework's three pillars:

1. **No unnecessary code** — the package exposes one small `Client`
   contract plus optional capability interfaces, not a kitchen-sink
   mail SDK.
2. **No unnecessary requests / leaks** — attachments stream, JMAP
   method calls are batched with back-references, every goroutine
   (push) is context-owned and goleak-verified, every response body
   read is size-bounded.
3. **No unnecessary external dependencies** — JMAP is JSON over
   HTTPS. A complete client needs only `net/http`, `encoding/json`,
   and `mime`. Phase 1–3 therefore add **zero** dependencies to
   `go.mod`.

## Decision

Add `pkg/mail` (domain types + `Client` contract) and
`pkg/mail/jmap` (stdlib-only RFC 8620/8621 implementation,
Stalwart-first).

### Contract shape

- `mail.Client` is the transport-agnostic interface: mailboxes,
  message pages, single-message fetch, streamed attachments, flag /
  move / delete mutations, send, capabilities, close.
- Optional capabilities are separate interfaces discovered by type
  assertion: `Threader`, `Pusher`, `Searcher`. A transport that
  cannot provide them simply does not implement them; callers feature-
  detect instead of branching on a transport enum.
- **String IDs everywhere.** JMAP IDs are opaque strings natively.
  The future IMAP transport must map `mailbox + UIDVALIDITY + UID`
  into one opaque string and never leak sequence numbers into the
  API.
- **HTML is data, not markup.** Message bodies are plain `string`.
  Sanitization is the renderer's responsibility; the package never
  produces `template.HTML`.
- **Auth is injected.** An `Authenticator` decorates each outgoing
  `*http.Request` (Basic or Bearer). Token acquisition/refresh stays
  with the caller (pairs with `pkg/auth/oidc`); `pkg/mail` neither
  stores nor logs secrets.

### Why a from-scratch JMAP client

Third-party Go JMAP libraries exist (`mikluko/jmap`,
`pr0ton11/jmap-go`). Wrapping one was rejected because:

- it would add the first non-test third-party dependency tree to the
  framework module for ~1–2 kLOC of JSON plumbing we can own;
- the public surface we need (eight methods) is a fraction of the
  full JMAP spec surface those libraries model;
- protocol details stay in `pkg/mail/jmap/internal/wire`, so a future
  swap to a library remains possible without breaking the public API.

### Deferred: IMAP/SMTP transport

`pkg/mail/imap` + `pkg/mail/smtp` (maddy, Dovecot) require
`github.com/emersion/go-imap` and friends — real new dependencies that
need their own ADR (0005) and a leak-hardening pass on connection
pooling. The `Client` contract in this ADR is designed so that the
IMAP transport slots in without public API changes.

### Security / operations requirements (binding)

- TLS verification is on by default. A dev-only insecure mode exists
  solely as an explicit constructor option and emits a Warn-level
  audit event when enabled.
- Security-relevant events emit structured `slog` events under the
  `mail.audit` message, mirroring `pkg/auth`: `auth_failed`,
  `session_refreshed`, `insecure_tls_enabled`, `push_started`,
  `push_stopped`. Covered by `*_audit_test.go`.
- Log redaction helpers guarantee addresses, subjects, and tokens do
  not reach logs from inside the package; log fields carry counts and
  IDs, not content.
- Bounded reads: every JMAP response body is wrapped in an
  `io.LimitReader`-style guard; attachment downloads stream through
  `io.ReadCloser` and are never buffered.
- Concurrency: a `jmap.Client` is safe for concurrent use; all state
  mutation (session refresh) is mutex-guarded; the only goroutine
  (EventSource push) is owned by the caller's context and verified by
  goleak.

### Threat model summary

| Asset | Threat | Mitigation |
|---|---|---|
| Mailbox credentials / bearer tokens | leakage via logs or API | injected `Authenticator`, never stored, redaction helpers, audit tests |
| Message content | leakage via logs | log redaction; only IDs/counts logged |
| HTML bodies | XSS in consuming UI | bodies are `string`; sanitization contract documented on `Message` |
| Server responses | memory exhaustion | size-bounded response reads, streamed blobs |
| TLS downgrade | MITM in dev-mode misuse | insecure mode is explicit + audited, off by default |

## Consequences

### Positive

- Framework `go.mod` is unchanged: zero new dependencies through
  Phase 3.
- One contract serves Stalwart today and maddy/Dovecot later; webmail
  apps (separate repositories) depend only on `mail.Client`.
- Protocol wire types are `internal`, so RFC churn or a library swap
  cannot break consumers.

### Negative / accepted costs

- We own JMAP protocol maintenance (~1.5 kLOC + fixtures). Accepted:
  the subset used is stable (RFC 8620/8621 are final).
- No SRV-record autodiscovery in v1 (explicit session URL or
  `.well-known/jmap` on a known host only). Accepted: webmail apps
  configure their server explicitly.
- IMAP users wait for Phase 4. Accepted: Stalwart is the priority
  backend.

## Verification

- Fixture-driven protocol tests against recorded Stalwart JSON via
  `httptest.Server` — no network in CI.
- `leak_test.go` with goleak in both packages.
- Audit-log assertions (`*_audit_test.go`) for every security event.
- Manual smoke recipe (not in CI): run Stalwart in Docker on
  localhost, create one account, and exercise the client end-to-end:

  ```bash
  docker run -d --name stalwart -p 8080:8080 -p 443:443 \
    -v stalwart-data:/opt/stalwart stalwartlabs/stalwart:latest
  # create an account in the web admin, then point a build-tagged
  # integration test (go test -tags=stalwart_integration) at
  # http://localhost:8080/.well-known/jmap
  ```
