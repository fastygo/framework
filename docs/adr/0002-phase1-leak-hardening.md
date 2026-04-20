# ADR 0002 — Phase 1 leak hardening

- Status: Accepted
- Date: 2026-04-19
- Deciders: framework maintainers
- Release: v0.1.0

## Context

A roadmap audit (`.project/roadmap-framework.md`) and code review
identified three classes of latent bugs in `pkg/...` that pass tests
but fail in production:

1. **Orphan goroutines on shutdown**: `WorkerService.Start(ctx)` did
   not track its goroutines. After `App.Run` returned, background
   tickers continued running until the process exited. A panic inside
   `BackgroundTask.Run` killed the entire process.
2. **Unbounded cache growth**: `pkg/cache` exposed `Cleanup()` but the
   framework did not schedule it. Consumers using `cache.New[T]` with
   write-only keys leaked memory indefinitely.
3. **Hard-coded `http.Server` timeouts**: `App.Run` baked
   `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `MaxHeaderBytes`, and
   the 5s shutdown budget into source code. Operators could not
   harden the server against Slowloris (12-Factor III violation).

Goroutine leaks are the dominant production failure mode for Go
services in 2025–2026 (OWASP Go top issues, Go 1.25 release notes
adding `waitgroup` and `hostport` vet analyzers, multiple post-mortems
including Datadog's Go 1.24 memory regression). Catching them in CI is
non-negotiable before shipping `v0.1.0`.

## Decision

Six PRs landed in branch `improved` under additive-API constraints (no
breaking changes — all five `examples/*` modules continue to compile
unchanged):

1. **Worker shutdown** — `WorkerService` gained `sync.WaitGroup`,
   per-task `recover()` with stack-trace logging, and a new
   `Stop(ctx) error` method. `App.Run` calls `workers.Stop` after
   `server.Shutdown` in both shutdown branches.
2. **Cache cleanup wiring** — new `app.CleanupTask[V](name, interval, *cache.Cache[V])`
   helper bridges `pkg/cache` and `BackgroundTask` without forcing
   `pkg/cache` to import `pkg/app`. `cache.New` godoc warns about the
   leak risk and points to the helper.
3. **HTTP timeouts** — `Config` gained six new fields (`HTTPReadTimeout`,
   `HTTPReadHeaderTimeout`, `HTTPWriteTimeout`, `HTTPIdleTimeout`,
   `HTTPMaxHeaderBytes`, `HTTPShutdownTimeout`) backed by
   `APP_HTTP_*` environment variables. New
   `AppBuilder.WithHTTPServerOptions` allows programmatic overrides.
   `New()` backfills zero-value HTTP fields so `Config` literals stay
   safe.
4. **goleak in tests** — `go.uber.org/goleak v1.3.0` in `TestMain` for
   `pkg/app`, `pkg/cache`, `pkg/web/security`, `pkg/web/content`.
   Pure packages without goroutines are skipped.
5. **golangci-lint + go vet** — `.golangci.yml` enables errcheck,
   govet (enable-all minus fieldalignment), staticcheck, gosec,
   bodyclose, contextcheck, noctx, errorlint, gocritic. CI runs
   `go vet ./...` explicitly so Go 1.25 `waitgroup` and `hostport`
   analyzers participate.
6. **ADR + CHANGELOG + tag** — this ADR plus 0001, `CHANGELOG.md`,
   `RELEASE.md`, and the `v0.1.0` git tag.

## Consequences

Positive:

- A panicking background task is logged with a stack trace and the
  ticker keeps firing — single-task bug no longer crashes the process.
- `App.Run` no longer leaks goroutines. CI enforces this via goleak.
- Cache memory is bounded as long as consumers register
  `app.CleanupTask`; the godoc warning makes this discoverable.
- `http.Server` is hardened by default; operators tune timeouts via
  ENV without redeploying.
- New concurrency bugs caught at PR time by `go vet -waitgroup` and
  `go vet -hostport` (Go 1.25 built-ins) plus golangci-lint.
- Public API surface unchanged — five examples build untouched.

Negative / open:

- `goleak` is now a transitive dependency (marked `// indirect` in
  `go.mod` since it is only used from `_test.go`). Acceptable.
- `golangci-lint` requires v1.64+; older local installs may report
  config errors until upgraded. CI pins the version.
- CSRF middleware, CSP nonce helper, and `pkg/auth` test coverage
  remain deferred to v0.1.1 (explicitly out of scope per the
  Minimal-scope decision).

## Out of scope (deferred)

- `pkg/auth` test coverage (session round-trip, OIDC mock) → v0.1.1
- CSRF middleware → v0.1.1
- CSP nonce generator → v0.1.1
- OpenTelemetry hooks, `/healthz` endpoint → Phase 2

## References

- `.project/PLAN-phase1-minimal-leaks.md` — implementation plan
- `.project/roadmap-framework.md` — full roadmap
- `CHANGELOG.md` `[0.1.0]` entry
- Go 1.25 release notes — new `waitgroup`, `hostport` vet analyzers
- OWASP Golang Security Best Practices (2025)
