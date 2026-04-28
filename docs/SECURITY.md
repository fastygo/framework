# Security

> Threat model, defaults, and the framework-vs-application
> responsibility split for `github.com/fastygo/framework`. Read
> together with [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) (the
> middleware order and lifecycle this document refers to) and
> [`docs/12-FACTOR.md`](12-FACTOR.md) (the env vars that toggle the
> defences described below).
>
> If you discover a vulnerability, **do not open a public issue.**
> See [§7 Reporting](#7-reporting-vulnerabilities) for the private
> disclosure path.

---

## 1. Scope

This document covers the framework's contribution to the security
posture of an application built on top of it. The framework is a
**library**, not a runtime: it ships hardening primitives wired into
sensible defaults, but the application owns its data, its
infrastructure, and its policy decisions.

In scope:

- The default HTTP middleware chain (`pkg/web/security/*`,
  `pkg/web/middleware/*`).
- HMAC-signed cookie sessions and the OIDC client (`pkg/auth`).
- The Prometheus text-format metrics endpoint
  (`pkg/web/metrics`) and health endpoints (`pkg/web/health`).
- The secure static-file server (`pkg/web/security/staticfs.go`).
- Resource lifecycle guarantees that affect denial-of-service
  resistance (graceful shutdown, panic recovery, cache cleanup).

Out of scope:

- Authorization logic (RBAC, ABAC, row-level filters): the
  framework provides typed sessions; what they grant is a feature
  decision.
- Cryptographic key management: the framework consumes
  `SESSION_KEY`; rotation, vaulting, and HSM integration are
  operational concerns.
- Network-layer protections (TLS termination, WAF, DDoS scrubbers):
  they belong at the edge, not in the process.
- Supply-chain vetting beyond the framework's own three-pillars
  rule (no unnecessary dependencies). Applications are responsible
  for vetting their own `go.sum`.

---

## 2. Trust boundaries

```text
   ┌─────────────────────────────────────────────────────────────┐
   │ Untrusted network (clients, scanners, hostile bots)          │
   └──────────────────────────────┬──────────────────────────────┘
                                  │ TLS
                                  ▼
   ┌─────────────────────────────────────────────────────────────┐
   │ Trusted edge (reverse proxy / load balancer / WAF)           │  ← may set X-Forwarded-For
   └──────────────────────────────┬──────────────────────────────┘
                                  │ HTTP/1.1 keep-alive
                                  ▼
   ┌─────────────────────────────────────────────────────────────┐
   │ Application process                                          │
   │                                                              │
   │   ┌───────────────────────────────────────────────────┐      │
   │   │ Framework middleware chain  (defence in depth)    │      │
   │   │  Tracing → Metrics → RequestID → Recover →        │      │
   │   │  Headers → BodyLimit → MethodGuard → AntiBot →    │      │
   │   │  RateLimit → Logger                               │      │
   │   └───────────────────────────────────────────────────┘      │
   │                          │                                   │
   │                          ▼                                   │
   │   ┌───────────────────────────────────────────────────┐      │
   │   │ Feature handlers (application-owned trust)        │      │
   │   │   sessions, OIDC client, business logic, storage  │      │
   │   └───────────────────────────────────────────────────┘      │
   │                                                              │
   └─────────────────────────────────────────────────────────────┘
```

Two boundaries matter most:

1. **Edge → application.** The framework treats `RemoteAddr` as
   authoritative *unless* `APP_SECURITY_TRUST_PROXY=true`. When
   trust is opted in, `ClientIP` honours `X-Real-IP` and the
   first hop of `X-Forwarded-For`. Set this **only** when a
   trusted proxy strips client-supplied copies of those headers.
2. **Middleware → handler.** Everything past the middleware chain
   has already been validated for verb, body size, suspicious
   path, and rate. Handlers may assume `r.Body` will not exceed
   `Config.MaxBodySize` (it is wrapped in `http.MaxBytesReader`).

---

## 3. Threat model

The framework defends against the threats below by default. Each
row points at the package(s) that own the defence and at the
mitigated attack class. Threats marked **app** are the
application's responsibility (the framework provides plumbing, not
policy).

| # | Threat                                          | Owner     | Defence                                                                                                  |
|---|--------------------------------------------------|-----------|----------------------------------------------------------------------------------------------------------|
| T1 | Slowloris / slow-write exhaustion                | framework | `Config.HTTP{Read,ReadHeader,Write,Idle}Timeout` + `MaxHeaderBytes` (12-Factor III; ADR 0002)             |
| T2 | Goroutine leaks at shutdown                       | framework | `WorkerService.Stop` waits on `sync.WaitGroup`; CI guarded by `goleak` (ADR 0002)                        |
| T3 | Panic in handler crashes process                  | framework | `RecoverMiddleware` returns 500 + `http.panic` slog event with stack                                     |
| T4 | Panic in background worker crashes process        | framework | `WorkerService.safeRun` recovers + logs stack; ticker keeps firing                                       |
| T5 | Unbounded request body (memory pressure)          | framework | `BodyLimitMiddleware` rejects 413 above `MaxBodySize`; wraps body in `http.MaxBytesReader`               |
| T6 | Unbounded cache growth (memory leak)              | framework | `pkg/cache` sharded TTL + `Len`/`Stats` + optional `MaxEntries` + `app.CleanupTask` eviction             |
| T7 | Unbounded rate-limit map growth                   | framework | `RateLimiter.Cleanup` background task drops idle visitors every minute (5-min staleness)                 |
| T7a| Unbounded Instant page snapshots                  | framework | `pkg/web/instant` validates fixed page count and byte budgets at startup                                 |
| T8 | Brute force / credential stuffing on a single IP  | framework | Sharded token-bucket `RateLimitMiddleware`; defaults 50 r/s, burst 100                                   |
| T9 | Reconnaissance scanners (sqlmap, nikto, …)        | framework | `AntiBotMiddleware` blocks empty UA + 7 known scanner signatures                                         |
| T10| Path traversal / dotfile reads on static assets   | framework | `SecureFileServer` rejects `..`, dot-segments, returns 404/403 before stdlib FS sees the path            |
| T11| Method-based bypasses (TRACE, CONNECT)            | framework | `MethodGuardMiddleware` returns 405                                                                      |
| T12| Reconnaissance probes against `.env`, `wp-admin`  | framework | `MethodGuardMiddleware` returns 404 for known patterns                                                   |
| T13| Clickjacking                                       | framework | `X-Frame-Options: DENY` (default); CSP `frame-ancestors` per app                                         |
| T14| MIME sniffing                                     | framework | `X-Content-Type-Options: nosniff` (always)                                                               |
| T15| Mixed-content / TLS strip                         | framework | `Strict-Transport-Security: max-age=63072000; includeSubDomains; preload` when `Config.HSTS=true`        |
| T16| Information leakage via Referer                   | framework | `Referrer-Policy: strict-origin-when-cross-origin`                                                       |
| T17| Browser feature abuse (cam/mic/geo)               | framework | `Permissions-Policy: geolocation=(), microphone=(), camera=()` by default                                |
| T18| Session forgery (HMAC bypass)                     | framework | `pkg/auth.CookieSession` HMAC-SHA256, constant-time compare, `session_tampered` audit on failure         |
| T19| Session replay past expiry                        | framework | Envelope carries `Exp`; expired cookies rejected; `session_expired` audit                                |
| T20| Cookie theft via XSS                              | shared    | Framework: `HTTPOnly` field on `CookieSession`. App: emit a CSP that blocks inline scripts (T22)         |
| T21| Cookie theft via TLS strip                        | shared    | Framework: `Secure` field on `CookieSession`. Ops: terminate full HTTPS, set `APP_SECURITY_HSTS=true`    |
| T22| Cross-site scripting (XSS)                        | **app**   | Framework: `templ` auto-escapes HTML; `Config.CSP` slot. App: pick a strict CSP, use nonces in templates |
| T23| Cross-site request forgery (CSRF)                 | **app**   | Framework: `SameSite` field on `CookieSession`. App: token-pattern middleware (planned for v0.1.1)       |
| T24| SQL/NoSQL injection                               | **app**   | Use parameterised queries; framework never touches the DB                                                |
| T25| Server-side request forgery (SSRF)                | **app**   | If a feature performs outbound HTTP, validate the target host                                            |
| T26| Authorization bypass (IDOR, privilege escalation) | **app**   | Framework supplies typed sessions; access checks belong in handlers                                      |
| T27| Secrets in logs                                   | shared    | Framework: never logs `SESSION_KEY`, OIDC client secret, cookie value. App: same discipline in handlers  |
| T28| Sensitive data in metrics labels                  | **app**   | Framework: stable label cardinality on built-in metrics. App: never use raw user input as a label        |
| T29| Scrape DoS on `/metrics`                          | framework | `Registry.Write` takes one snapshot under RLock; concurrent observations are lock-free                   |
| T30| Open redirect                                     | **app**   | If a feature implements redirects, validate the target URL is in an allow-list                           |

T20–T22 are joint: the framework gives you the knobs, but only the
application decides how strict the policy should be.

---

## 4. Defaults at a glance

The defaults below are what `app.New(cfg)` produces with
`security.DefaultConfig()`. Override per-deployment via
`APP_SECURITY_*` env vars (see `docs/12-FACTOR.md`).

| Knob                           | Default                                   | Where                                |
|---------------------------------|-------------------------------------------|--------------------------------------|
| `Enabled`                       | `true`                                    | `security.DefaultConfig`             |
| `HSTS`                          | `false` (enable when fully HTTPS)         | `security.DefaultConfig`             |
| `FrameOptions`                  | `DENY`                                    | `security.DefaultConfig`             |
| `CSP`                           | empty (set per app)                       | `security.DefaultConfig`             |
| `Permissions`                   | `geolocation=(), microphone=(), camera=()`| `security.DefaultConfig`             |
| `MaxBodySize`                   | 1 MiB                                     | `security.DefaultConfig`             |
| `PageRateLimit` / `PageRateBurst` | 50 r/s / burst 100                      | `security.DefaultConfig`             |
| `TrustProxy`                    | `false` (explicit trusted-edge opt-in)    | `security.DefaultConfig`             |
| `BlockEmptyUA`                  | `true`                                    | `security.DefaultConfig`             |
| `HTTPReadHeaderTimeout`         | non-zero (filled by `applyHTTPDefaults`)  | `app.New`                            |
| `HTTPShutdownTimeout`           | non-zero (filled by `applyHTTPDefaults`)  | `app.New`                            |
| Cookie `Secure`                 | `false` (you set `true` in production)    | `auth.CookieSession`                 |
| Cookie `HTTPOnly`               | `false` (you set `true` in production)    | `auth.CookieSession`                 |
| Cookie `SameSite`               | unset (you pick `Lax`/`Strict`)           | `auth.CookieSession`                 |

Two defaults are deliberately conservative-by-omission: the
framework will not flip `Secure` / `HTTPOnly` on cookies behind your
back, because doing so silently when the deployment is not yet
HTTPS would *appear* to work and then break login. Configure both
explicitly in `main`.

---

## 5. Framework vs application responsibilities

The matrix below is the contract. Anything not listed is the
application's responsibility by default.

| Concern                                | Framework                                                     | Application                                                              |
|----------------------------------------|---------------------------------------------------------------|--------------------------------------------------------------------------|
| Transport (TLS, HSTS preload)          | Sets HSTS header when toggled                                 | Terminates TLS; submits domain to preload list                           |
| HTTP timeouts                          | Provides defaults; honours `Config.HTTP*` overrides           | Tunes for its own SLA                                                    |
| Body size, method guard, anti-bot      | Default chain (opt-out via `WithSecurity({Enabled:false})`)   | Adjusts limits in `Config`                                               |
| Rate limiting                          | Per-IP token bucket, sharded; cleanup task                    | Decides per-route limits if needed (extra middleware on its own routes)  |
| Secure static files                    | Path traversal + dotfile defence; ETag; immutable-asset cache | Chooses `StaticDir` and bundling pipeline                                |
| Security headers                       | Headers always set; CSP slot is empty by default              | Picks a strict CSP (nonces, `default-src 'self'`)                        |
| Sessions                               | HMAC envelope, signed cookies, audit events                   | Picks `Secret`, `TTL`, `Secure`, `HTTPOnly`, `SameSite`; rotates secrets |
| Authentication (OIDC)                  | Discovery + JWKS verification + state/nonce                   | Owns login UX, account linking, post-login authorization                 |
| Authorization                          | —                                                             | Implements RBAC/ABAC inside handlers                                     |
| CSRF                                   | `SameSite` cookie attribute on sessions                       | Anti-forgery tokens for state-changing forms (planned helper in v0.1.1)  |
| XSS                                    | `templ` auto-escapes; CSP slot                                | Avoids inline scripts; uses nonces; reviews HTML helpers                 |
| Logging discipline                     | Structured `slog`; never logs secrets it sees                 | Never logs PII / tokens it touches                                       |
| Observability of attacks               | `auth.audit` events, `http.panic`, request IDs                | Wires logs/metrics/traces to a SIEM                                      |
| Secret storage                         | Reads `SESSION_KEY`, `OIDC_CLIENT_SECRET` from env            | Ships them via secret manager / K8s Secret / Vault                       |
| Dependency vetting                     | Three direct deps (`templ`, `goldmark`, `goleak` test-only)   | Vets its own indirect deps; re-runs `govulncheck` on every release       |

If a row reads "framework" you can rely on it without writing the
code yourself. If it reads "application", the framework will not
silently substitute a default — you will notice the gap during
implementation.

---

## 6. Operational checklist (pre-production)

Tick this list before exposing an application to the internet.

### Configuration

- [ ] `SESSION_KEY` is at least 32 random bytes (output of
      `openssl rand -hex 32`) and stored in a secret manager.
- [ ] `OIDC_CLIENT_SECRET` is delivered via the same channel; never
      committed to the repo.
- [ ] `APP_SECURITY_HSTS=true` once the entire site is HTTPS.
- [ ] `APP_SECURITY_TRUST_PROXY` matches the deployment topology.
      The default is `false`; set it to `true` only when a trusted
      reverse proxy strips
      client-supplied `X-Forwarded-For` / `X-Real-IP`.
- [ ] `APP_SECURITY_CSP` is set to a strict policy. As a starting
      point: `default-src 'self'; script-src 'self' 'nonce-{nonce}'`.
- [ ] `APP_SECURITY_MAX_BODY_BYTES` reflects your largest legitimate
      upload (and not more).
- [ ] `APP_SECURITY_RATE_PER_IP` / `APP_SECURITY_RATE_BURST` align
      with expected human traffic; monitor 429s after launch.
- [ ] `HTTP_*_TIMEOUT` values match your slowest legitimate request.
      Defaults are conservative; do not set anything to zero.

### Cookies

- [ ] Every `auth.CookieSession` literal in `main.go` sets
      `Secure: true`, `HTTPOnly: true`, and an explicit `SameSite`
      value (`http.SameSiteLaxMode` for typical SSR apps).
- [ ] `TTL` matches your idle-session policy; do not leave it
      unbounded.
- [ ] Secret rotation is documented for the on-call team. Rotation
      invalidates every active session — by design.

### Templates and HTML

- [ ] All HTML is rendered via `templ`. Any `templ.Raw` or
      `template.HTML` call is reviewed for trusted input.
- [ ] Inline `<script>` blocks are nonce-bound (or removed) so the
      CSP can be `'strict-dynamic'`-friendly.

### Static assets

- [ ] `StaticDir` does not contain dotfiles or source files. The
      `SecureFileServer` blocks them, but defence in depth means
      not putting them there in the first place.
- [ ] Build pipeline emits hashed filenames; the immutable-asset
      cache (`max-age, immutable`) is then safe.

### Observability

- [ ] `auth.audit` events ship to a SIEM. Alert rules fire on
      `session_tampered` (Warn) and a sustained rate of
      `session_issue_failed` (Warn).
- [ ] `http.panic` events fire a paging alert.
- [ ] `/metrics` is reachable only from the scraper's network
      (segment via firewall or service mesh). The endpoint itself
      contains no secrets, but cardinality and timing data can leak
      operational topology.
- [ ] Health probes (`/healthz`, `/readyz`) are wired to the
      orchestrator; readiness includes the DB and any required
      upstream.

### Build pipeline

- [ ] `make ci` and `make lint-go` are required checks on the PR.
- [ ] `golangci-lint` is pinned to v1.64.x (matches `.golangci.yml`).
- [ ] `govulncheck ./...` runs on every release tag.
- [ ] Container base image is tracked; rebuild policy is documented.

### Runtime

- [ ] The process runs as a non-root user.
- [ ] The container is read-only filesystem (the framework writes
      no files at runtime).
- [ ] CPU/memory requests reflect the steady-state RSS. Monitor
      `pkg/cache` `Stats()` for unexpected cardinality, and keep
      Instant page snapshots within explicit byte budgets.
- [ ] Graceful shutdown is plumbed: orchestrator sends SIGTERM and
      waits at least `HTTP_SHUTDOWN_TIMEOUT` before SIGKILL.

---

## 7. Reporting vulnerabilities

**Do not file public GitHub issues for security bugs.**

Instead, contact the maintainers privately via the email on the
project's GitHub profile. Please include:

- Affected version (commit SHA or tag).
- A minimal reproduction (curl invocation, snippet, or test case).
- Your assessment of impact and any suggested fix.

We aim to acknowledge within 72 hours, ship a coordinated fix, and
credit reporters in the release notes (opt-out available). If a CVE
is warranted, we will request one and announce via GitHub Security
Advisories.

A reporter who follows this process in good faith will not face
legal action from the project even if the test inadvertently
touches a third-party deployment.

---

## 8. Change history

Material changes to the security posture are logged here in
addition to `CHANGELOG.md`. Older changes are recorded in ADRs.

- **Phase 1 (`v0.1.0`)** — Goroutine leak hardening, panic recovery
  in workers, configurable HTTP timeouts, cache cleanup task. See
  [ADR 0002](adr/0002-phase1-leak-hardening.md).
- **Phase 2 (`v0.2.0`)** — Health/metrics endpoints with their own
  slim middleware chain (no scrape DoS surface), structured
  `auth.audit` events, slog correlation. See
  [ADR 0003](adr/0003-phase2-observability.md).
- **Phase 2.5** — `RequestIDMiddleware` no longer depends on
  `google/uuid`; `Registry.Write` snapshot pattern (T29) removes
  scrape vs. observation contention. See `CHANGELOG.md` `[0.2.1]`.
