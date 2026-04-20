# Changelog

All notable changes to `github.com/fastygo/framework` are documented
here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and the project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **Phase 3.3 — coverage gate in CI.** New cross-platform Go program
  `scripts/coverage-gate` parses a `coverage.out` profile (no shell,
  no `go tool cover` dependency) and fails the build when any tracked
  package drops below its declared per-package threshold. The
  threshold table lives next to the enforcement code as a single
  source of truth: security-critical packages (`pkg/auth`,
  `pkg/web/security`) at 90 %, core domain (`pkg/core`,
  `pkg/core/cqrs`, `pkg/core/cqrs/behaviors`) at 80 %, composition
  root (`pkg/app`) at 85 %, infrastructure with full suites (`cache`,
  `health`, `view`) at 80–90 %. Wired into `make ci`,
  `scripts/preflight.sh`, and `.github/workflows/ci.yml` (which also
  uploads the profile as a workflow artefact for inspection).
  Untracked packages are reported but never block the gate, so adding
  a new `pkg/foo` is a non-breaking event until thresholds are added.
- **Phase 3.1+ — `pkg/web/security` and `pkg/app` coverage push.**
  Added focused regression suites that take the two largest gaps to
  the targets set by Phase 3.3:
  `pkg/web/security` 61 → **98.0 %** (`config_test.go` covers the full
  `LoadConfig` env overlay incl. malformed and non-positive numerics
  falling back to defaults; `extra_test.go` adds body-limit
  pass-through and under-limit, rate-limiter constructor clamping,
  `Cleanup` stale + fresh, nil-limiter and unknown-IP middleware,
  every suspicious-path category for `MethodGuard` plus a documented
  "normalised traversal not flagged" regression, secure file server
  with non-immutable extensions, default `maxAge` fallback, 304 on
  `If-None-Match`, traversal/missing/directory/root rejection,
  `cacheControlValue` table, `hasDotSegment` table, and a known-issue
  snapshot test for the existing "no-extension treated as immutable"
  classification so any future fix breaks the gate intentionally).
  `pkg/app` 79 → **93.3 %** (`builder_extra_test.go` covers the
  zero-coverage accessors `Mux`, `WithLogger(nil)`, `WithStaticPrefix`
  variants, `WithFeature(nil)`, `AddBackgroundTask`,
  `WithMetricsRegistry`, `WithMetricsEndpoint` lazy-create,
  `WithTracer`, `AddHealthChecker`; the `Build` accessors
  `Config`/`Workers`/`Handler`/`NavItems` with a defensive-copy
  assertion; `collectNavItems` sort + empty path; `App.Run`
  init-error abort, `Closer` reverse-order invocation, and the
  `ListenAndServe` error path; `ServeHTTP` delegation; `closeFeatures`
  non-closer skip; `WorkerService.Add` rejecting invalid tasks +
  defaulting Name + Stop with nil ctx + Stop deadline; `safeRun`
  panic recovery). Together these close the last "bigger than 5 % gap"
  pockets in the framework and make the coverage gate sustainable.
- **Phase 3.1 — core test hardening.** Added focused unit suites for
  the previously low-coverage core packages, lifting per-package
  statement coverage to: `pkg/auth` 30 → **90.6 %** (`session_test.go`,
  `signed_test.go`, `oidc_test.go` with a stdlib-only mock OP +
  RS256/JWKS), `pkg/core` 0 → **100 %** (`errors_test.go`,
  `entity_test.go`), `pkg/core/cqrs` 0 → **96.0 %**
  (`dispatcher_test.go`, `handler_test.go`, `pipeline_test.go`),
  `pkg/core/cqrs/behaviors` 0 → **100 %** (`logging_test.go`,
  `validation_test.go`). The OIDC suite covers the full happy path
  (discovery caching, code exchange, ID-token verification) plus the
  reject paths that matter for security: tampered signature, issuer
  mismatch, audience mismatch, expired token, unknown `kid`. No new
  external dependencies were introduced for any of these tests.
- **`pkg/core.DomainError.Unwrap()`.** Additive method exposing the
  wrapped `Cause` so `errors.Is` / `errors.As` / `errors.Unwrap` can
  traverse domain-error chains. Pre-existing call sites that built
  `DomainError` literals through `WrapDomainError` are unaffected;
  `NewDomainError` (no cause) returns `nil` from `Unwrap` as expected.
- **Documentation: `docs/SECURITY.md`.** Threat model and
  responsibility split between framework and application: scope,
  trust-boundary diagram, a 30-row threat table (T1–T30) mapping
  each threat to the package or actor that mitigates it, defaults
  table for `security.Config` (with a note on why `Secure` /
  `HTTPOnly` are *not* flipped silently), framework-vs-application
  responsibility matrix, a 7-section pre-production operational
  checklist (configuration, cookies, templates, static assets,
  observability, build pipeline, runtime), private vulnerability
  reporting process, and a change history that links Phase 1 / 2 /
  2.5 to the relevant ADRs and CHANGELOG entries. Closes Phase 3C.
- **Documentation: `CONTRIBUTING.md`.** Practical onboarding guide
  rooted in the actual repository: the three-pillars rule, the
  toolchain table (Go 1.25.x, golangci-lint v1.64.x, templ
  v0.3.1001), local setup with `make ci` and the portable
  `scripts/preflight.sh` (incl. `PREFLIGHT_CI_PARITY=1` for CI
  parity and `PREFLIGHT_RUN_RACE=1` for race tests), the test
  taxonomy used in the codebase (`leak_test.go`, `*_audit_test.go`,
  `example_test.go`, `*_integration_test.go`), the docs/code "same
  commit" rule, the local quality gates (`make ci`, `make lint-go`,
  `scripts/godoc-audit`), commit/PR style, the CHANGELOG protocol,
  the ADR template (with the list of changes that *require* an
  ADR), the CI workflow overview, and the release/security/licence
  pointers. Replaces the previous absence of contributor docs.
- **Documentation: `docs/ARCHITECTURE.md`.** Bird's-eye view of the
  framework: the three pillars, repository layout, layered import
  rules, the composition-root pattern, the full `Feature` contract
  (and its five optional capability interfaces), the `App.Run`
  lifecycle sequence diagram, the fixed middleware order, the
  `WorkerService` contract, the 12-factor configuration story, the
  observability "interfaces over implementations" stance, and the
  rule that this document is updated in the same commit as any
  contract change. Single source of truth referenced by quickstart,
  API reference, and ADRs.
- **Documentation: godoc audit + Example tests.** Every exported
  symbol in `pkg/` now carries a doc comment that starts with the
  symbol's name (verified by `scripts/godoc-audit`). Five public
  packages gained executable `Example*` tests that double as
  godoc-rendered usage and as regression guards for the API
  surface: `pkg/cache.Cache`, `pkg/web/metrics.Registry` (counter
  and histogram), `pkg/web/health.Aggregator` (up + down paths),
  `pkg/auth.CookieSession` (round-trip + tamper detection), and
  `pkg/app.AppBuilder` (feature wiring + health endpoints).

## [0.2.1] — 2026-04-19

Phase 2.5 cleanup pass. Pure hygiene driven by the framework's "three
pillars": no extra code, no extra requests/leaks, no extra deps. No
behaviour changes for end users beyond the import-path rename of the
markdown library.

### Changed
- **`pkg/web/middleware`**: `RequestIDMiddleware` no longer depends on
  `github.com/google/uuid`. UUID v4 generation is now inlined as
  `newRequestID()` (~30 LOC, `crypto/rand` + `encoding/hex`, zero heap
  allocations beyond the final string conversion). Behaviour is
  identical: the canonical `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx` form
  is still emitted.
- **`pkg/web/metrics.Registry.Write`**: takes the registry RLock
  exactly once and copies a `(names, metrics)` snapshot before any
  I/O. Previously it re-acquired the RLock on every loop iteration,
  which could ping-pong with concurrent observation paths under a
  slow scrape (gzipped HTTP response, network socket). Per-metric
  observation paths remain lock-free.
- **`pkg/web/content` → `pkg/content-markdown`**: package renamed and
  moved to `pkg/content-markdown` (identifier `contentmarkdown`) to
  prepare extraction into the standalone repository
  `github.com/fastygo/content-markdown`. Examples (`blog`, `docs`)
  use the import alias `content` to keep call sites unchanged.

### Removed
- `github.com/google/uuid` from `go.mod`. Direct framework
  dependencies are now `github.com/a-h/templ`, `github.com/yuin/goldmark`
  and `go.uber.org/goleak` (test-only).

### Tests
- New `TestNewRequestID_FormatAndUniqueness` (1024-id sample, regex +
  collision check) and middleware integration tests for the
  generate/echo paths.
- New `TestRegistry_WriteDoesNotBlockObservations` proves a 5ms-per-flush
  scrape does not stall a concurrent `Counter.Inc` (regression guard
  for the per-iteration RLock removal).
- `pkg/web/middleware/correlation_test.go`: 4 tests collapsed into 2
  table-driven tests, no coverage loss.

## [0.2.0] — 2026-04-19

Phase 2: observability without the SDK tax. Health, logs, metrics,
and traces all wired in with **zero new external dependencies** —
the framework's `go.mod` still ships exactly one indirect dep
(`go.uber.org/goleak`, tests only).

### Added

- `pkg/web/health` — `Aggregator`, `Checker`, `LiveHandler`,
  `ReadyHandler`. Parallel checks with per-check timeout (default 2s).
- `pkg/web/metrics` — from-scratch Prometheus text exposition
  format v0.0.4 with `Counter`, `Gauge`, `Histogram` and full label
  support. Lock-free observation paths via `sync.Map` + atomics.
- `pkg/web/middleware/metrics.go` — `MetricsMiddleware(reg)` records
  `http_requests_total{method,status}`, `http_requests_in_flight`,
  `http_request_duration_seconds{method}`.
- `pkg/observability` — interface-only tracing contract
  (`Tracer`, `Span`, `SpanContext`) and a `NoopTracer` default.
  No `go.opentelemetry.io/*` imports anywhere in the framework.
- `pkg/web/middleware/tracing.go` — `TracingMiddleware(t)` opens a
  span, attaches `(TraceID, SpanID)` to the request context as
  `middleware.Correlation`, and `LoggerMiddleware` decorates every
  log line accordingly.
- `pkg/web/middleware/correlation.go` — `WithCorrelation` /
  `CorrelationFromContext` for tracer adapters.
- `AppBuilder` opt-in methods:
  `WithHealthEndpoints(live, ready)`, `AddHealthChecker(c)`,
  `WithMetricsRegistry(r)`, `WithMetricsEndpoint(path)`,
  `WithTracer(t)`. Health and metrics endpoints mount *outside*
  the security chain so probes/scrapes bypass rate limit & anti-bot.
- `pkg/auth.CookieSession` emits structured `auth.audit` slog
  events (`session_issued`, `session_decoded`, `session_missing`,
  `session_tampered`, `session_expired`, `session_cleared`,
  `session_issue_failed`). Tamper-detection is logged at WARN so
  SIEM rules can alert on `level >= WARN` only.
- `Config` env-var-driven fields: `LOG_LEVEL`, `LOG_FORMAT`,
  `HEALTH_LIVE_PATH`, `HEALTH_READY_PATH`, `METRICS_PATH`,
  `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`.
- `goleak` integration in `pkg/web/health`, `pkg/web/metrics`,
  `pkg/web/middleware`, `pkg/observability`, `pkg/auth`.
- Documentation: `docs/OBSERVABILITY.md` (operator guide for all
  four pillars), `docs/12-FACTOR.md` (env-var matrix per Heroku
  factor), ADR 0003.

### Changed

- `LoggerMiddleware` migrated to `slog.LogAttrs` so the hot path
  no longer pays the variadic-allocation tax. Output now omits
  empty `trace_id`/`span_id` fields on the no-tracer path.
- `pkg/web/metrics.Registry.Write` (renamed from `WriteTo`) returns
  `error` rather than `(int64, error)` so it does not partially
  satisfy `io.WriterTo` and trip `go vet -stdmethods`.

### Security

- HMAC tamper-detection in `pkg/auth.CookieSession` now produces a
  structured WARN audit event instead of silently returning
  `(zero, false)`. Brute-force detection on signed cookies becomes
  feasible without parsing log message text.

### Deferred

The following items remain on the roadmap for v0.2.1+:

- A real OTel adapter under `github.com/fastygo/otel`.
- Prometheus `summary` and `exemplar` metric types.
- Native histograms (Prometheus 2.40+).
- Configurable span name templates (today: raw URL path).

## [0.1.0] — 2026-04-19

First versioned release after the framework/examples split. Focused on
closing three concrete leak risks identified in the v0.0.x audit. All
changes are additive — every `examples/*` module continues to compile
without source edits.

### Added

- `pkg/app.WorkerService.Stop(ctx) error` — blocks until every
  supervised goroutine returns or `ctx` is done. `App.Run` calls it
  after `server.Shutdown` so background workers no longer outlive the
  HTTP server.
- `pkg/app.CleanupTask[V]` helper that adapts a `*cache.Cache[V]` into
  a `BackgroundTask` so consumers can keep cache memory bounded
  without importing both packages by hand.
- Six configurable `http.Server` timeouts on `Config`
  (`HTTPReadTimeout`, `HTTPReadHeaderTimeout`, `HTTPWriteTimeout`,
  `HTTPIdleTimeout`, `HTTPMaxHeaderBytes`, `HTTPShutdownTimeout`)
  backed by `APP_HTTP_*` environment variables.
- `pkg/app.HTTPServerOptions` and
  `AppBuilder.WithHTTPServerOptions(opts)` for programmatic timeout
  overrides on top of ENV/defaults.
- `go.uber.org/goleak` integration in `TestMain` for `pkg/app`,
  `pkg/cache`, `pkg/web/security`, `pkg/web/content`.
- `.golangci.yml` enabling errcheck, govet, staticcheck, gosec,
  bodyclose, contextcheck, noctx, errorlint, gocritic.
- CI now runs `go vet ./...` explicitly so Go 1.25 `waitgroup` and
  `hostport` analyzers participate, plus `golangci-lint` via the
  official action pinned to v1.64.
- ADR 0001 (framework / examples split, retrospective) and ADR 0002
  (Phase 1 leak hardening) under `docs/adr/`.

### Changed

- `WorkerService` recovers panics inside `BackgroundTask.Run`, logs
  them with the task name and a full stack trace, and keeps firing
  subsequent ticks instead of crashing the process.
- `pkg/app.New(cfg)` backfills zero-value HTTP timeout fields with the
  same defaults `LoadConfig` produces, so hand-built `Config` literals
  cannot start an unprotected `http.Server`.
- `pkg/cache.New` godoc now explicitly warns about unbounded growth
  for write-only keys and points readers to `app.CleanupTask`.
- `Makefile` adds `lint-go` and `vet` targets; `make ci` runs `go vet`
  in addition to tests and the no-root-imports check.

### Security

- `http.Server` timeouts are no longer hard-coded. Defaults still
  match the prior literals (`ReadTimeout 5s`, `ReadHeaderTimeout 2s`,
  `WriteTimeout 10s`, `IdleTimeout 120s`, `MaxHeaderBytes 1<<20`,
  `ShutdownTimeout 5s`), but operators can tighten them per
  deployment to mitigate Slowloris and slow-write attacks.

### Deferred

The following remain on the roadmap for v0.1.1+:

- CSRF middleware
- CSP nonce helper for inline scripts
- `pkg/auth` test coverage (session round-trip, OIDC mock provider)
- `/healthz`, `/readyz`, `/metrics` endpoints
- OpenTelemetry middleware

## [0.0.x]

Unversioned development under the monorepo layout. Replaced by the
framework / examples split documented in ADR 0001.
