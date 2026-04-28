# Architecture

> Source of truth for `github.com/fastygo/framework`. This document
> describes what the framework is, what it deliberately is **not**,
> how the packages compose at runtime, and the contracts every
> consumer can rely on.
>
> Status: applies to the `improved` branch (Phase 1 + Phase 2 + Phase
> 2.5 closed). Update this file together with the code in the same
> commit when an architectural contract changes.

---

## 1. Goals and non-goals

### The three pillars

The whole framework is judged against three constraints. Every PR
must hold them simultaneously, otherwise it is not merged.

1. **No unnecessary code.** A line earns its place by removing more
   complexity than it adds. Prefer a 30-line stdlib implementation
   over a third-party API that hides 30k lines.
2. **No unnecessary requests.** No hidden DB calls, no surprise HTTP
   round-trips, no goroutine, cache or stack leaks. Every long-lived
   resource has a `Stop`/`Cleanup`/`Close` path that is actually
   wired into `App.Run`.
3. **No unnecessary external dependencies.** Direct dependencies fit
   on one line: `github.com/a-h/templ`, `github.com/yuin/goldmark`
   (markdown only, opt-in), and `go.uber.org/goleak` (test-only).

### Goals

- Be a **library**, not a runtime. Applications import `pkg/...` and
  stay in control of `main`. There is no CLI, no service registry,
  no plugin loader.
- Provide **safe defaults** for the dangerous parts of HTTP services:
  graceful shutdown, panic recovery, security headers, body limits,
  rate limiting, request timeouts.
- Provide **interfaces** for observability so applications can pick
  any backend (Prometheus, OTel) without dragging the dependency
  graph into the framework.
- Stay **boring** about Go: stdlib `net/http`, `log/slog`,
  `crypto/rand`. No code generation beyond `templ`.

### Non-goals

- A general-purpose dependency injection container.
- A Rails/Laravel-style "scaffold and run" experience.
- An ORM. Storage is the application's problem; the framework only
  defines `pkg/core/Entity` as a tiny typed marker for IDs and
  timestamps.
- A distributed runtime. Clustering, leader election, queues belong
  in dedicated libraries (planned: `github.com/fastygo/queue`).

---

## 2. Repository layout

```text
fastygo/framework/
├── pkg/                    # Public, importable library (this is the framework)
│   ├── app/                # AppBuilder, App, WorkerService, Config, Feature contracts
│   ├── auth/               # OIDC client + signed cookie sessions (HMAC, no JWT lib)
│   ├── cache/              # Generic sharded TTL cache (no external deps)
│   ├── content-markdown/   # Markdown rendering via goldmark (opt-in import)
│   ├── core/               # Entity[ID], DomainError, ErrorCode, CQRS dispatcher
│   ├── observability/      # Tracer / SpanContext interfaces (no OTel imports)
│   └── web/                # HTTP layer
│       ├── health/         # Aggregator, /healthz, /readyz
│       ├── i18n/           # Translation store
│       ├── instant/        # Fixed prebuilt page snapshot store
│       ├── locale/         # Accept-Language negotiation
│       ├── metrics/        # Prometheus text-format registry (no Prometheus dep)
│       ├── middleware/     # RequestID, Recover, Logger, Tracing, Metrics, Correlation
│       ├── security/       # Headers, BodyLimit, MethodGuard, AntiBot, RateLimit, SecureFileServer
│       └── view/           # Layout, theme/language toggles (templ-friendly DTOs)
├── examples/               # Each subfolder is its own Go module (landing, web, blog, docs, dashboard)
├── docs/                   # Long-form documentation (this file lives here)
├── scripts/godoc-audit/    # AST inspector enforcing godoc completeness as a regression guard
└── go.work                 # Workspace tying examples to the local pkg/ during development
```

Examples are deliberately separate Go modules. They cannot leak
their own dependencies (templates, DBs, fixtures) into the
framework's `go.sum`. ADR 0001 explains the split in detail.

---

## 3. Layered model

The framework has four layers. Each layer may import the layers
**above** it but never the layers below. Violating this order is
caught at code-review time and would produce an import cycle
anyway.

```text
                          ┌────────────────────────────────────┐
   Application owns this  │ cmd/server/main.go                 │   composition root
                          │ internal/site/<feature>/...        │   features (Feature impl)
                          └────────────────────────────────────┘
                                          │ uses
                                          ▼
                          ┌────────────────────────────────────┐
   Framework public API   │ pkg/app   (AppBuilder, lifecycle)  │   layer 4: integration
                          └────────────────────────────────────┘
                                          │
                                          ▼
                          ┌────────────────────────────────────┐
                          │ pkg/web/* (handlers, middleware,   │   layer 3: HTTP
                          │           security, health,        │
                          │           metrics, i18n, view)     │
                          │ pkg/auth  (sessions, OIDC client)  │
                          │ pkg/content-markdown (opt-in)      │
                          └────────────────────────────────────┘
                                          │
                                          ▼
                          ┌────────────────────────────────────┐
                          │ pkg/core, pkg/cache,               │   layer 2: domain primitives
                          │ pkg/observability                  │
                          └────────────────────────────────────┘
                                          │
                                          ▼
                          ┌────────────────────────────────────┐
                          │ Go stdlib (net/http, log/slog,     │   layer 1: stdlib
                          │ crypto/rand, encoding/...)         │
                          └────────────────────────────────────┘
```

Practical consequences:

- `pkg/core` and `pkg/cache` never import anything HTTP-shaped. You
  can use them in a CLI tool with zero web baggage.
- `pkg/web/instant` is HTTP-adjacent but deliberately static: fixed
  prebuilt page snapshots, construction-time budgets, no TTL and no
  background cleanup loop.
- `pkg/observability` is interface-only. Importing it does not pull
  `go.opentelemetry.io/*` into your `go.sum`. The future
  `github.com/fastygo/otel` adapter will provide the real
  implementation as a separate module.
- `pkg/app` is the only place that knows the order of middleware,
  the order of lifecycle hooks, and the worker shutdown contract.
  Reading `pkg/app/builder.go` is enough to understand the runtime.

---

## 4. The composition root

Applications are written in the **composition root** style: `main`
constructs every dependency, hands it to the builder, and starts the
result. Nothing is global, nothing reaches into a registry behind
your back.

A minimal `cmd/server/main.go` looks like this:

```go
func main() {
    cfg, err := app.LoadConfig()
    if err != nil { /* fail fast */ }

    db := openDatabase(cfg)            // application owns persistence
    sessions := auth.CookieSession[appSession]{
        Name: "sid", Secret: cfg.SessionKey, TTL: 8 * time.Hour,
        Secure: true, SameSite: http.SameSiteLaxMode, HTTPOnly: true,
    }

    a := app.New(cfg).
        WithFeature(blog.New(db)).
        WithFeature(profile.New(sessions, db)).
        WithHealthEndpoints(cfg.HealthLivePath, cfg.HealthReadyPath).
        WithMetricsRegistry(metrics.NewRegistry()).
        WithMetricsEndpoint(cfg.MetricsPath).
        Build()

    ctx, stop := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer stop()

    if err := a.Run(ctx); err != nil { /* log + exit */ }
}
```

Every example under `examples/` follows this exact shape.

---

## 5. The `Feature` contract

A feature is a compile-time plugin: the composition root constructs
it, the builder wires its routes, and from that point on it behaves
like any other `http.Handler`-producing object.

```go
type Feature interface {
    ID() string                        // stable identifier (used in logs, health, metrics)
    Routes(mux *http.ServeMux)         // register handlers on the shared mux
    NavItems() []NavItem               // contribute to global navigation
}
```

Optional capabilities are expressed as **separate interfaces** the
feature may implement on top of `Feature`. The builder type-asserts
them at `Build` and `Run` time:

| Optional interface       | When fired                                         | Purpose                                                     |
|--------------------------|----------------------------------------------------|-------------------------------------------------------------|
| `Initializer`            | Once, before HTTP listen, in registration order    | Warm caches, validate config, pre-load templates            |
| `Closer`                 | Once, on shutdown                                  | Close DB pools, flush logs                                  |
| `NavProvider`            | Once, at `Build`                                   | Receive the merged nav list (for layout shells)             |
| `HealthChecker`          | On every `/readyz` request                         | Contribute a probe under the feature's `ID()` as label      |
| `BackgroundProvider`     | At `Build`                                         | Register background tasks with the supervised WorkerService |

The split is intentional: a feature that only renders a page never
implements `BackgroundProvider`, so the builder never even imports
ticker code on its behalf. This keeps small features small.

---

## 6. Lifecycle of `App.Run`

`App.Run(ctx)` is the single entry point that starts the server and
blocks. Its sequence is documented here so you can reason about what
happens between "process started" and "process exited":

```text
caller                     App.Run                     server / workers
  │                           │                              │
  │  app.New(cfg)             │                              │
  │  .WithFeature(...)        │                              │
  │  .Build()                 │                              │
  │                           │                              │
  │  Run(ctx)  ──────────────►│                              │
  │                           │ for each Initializer.Init()  │
  │                           │   (sequential, fail-fast)    │
  │                           │                              │
  │                           │ workers.Start(runCtx)  ─────►│  N goroutines, supervised
  │                           │                              │
  │                           │ http.Server.ListenAndServe ─►│  accept loop
  │                           │                              │
  │                           │ select { ctx.Done           ─┘
  │                           │        | listen err }
  │                           │                              │
  │                           │ server.Shutdown(shutdownCtx)─►│  drains in-flight requests
  │                           │ workers.Stop(shutdownCtx) ───►│  WaitGroup drains tasks
  │                           │ for each Closer.Close()      │
  │                           │                              │
  │  ◄─────  return err       │                              │
```

Three guarantees come out of this sequence:

1. **No request outlives the process.** `server.Shutdown` waits for
   in-flight handlers up to `HTTPShutdownTimeout`. After that, the
   socket is closed.
2. **No goroutine outlives `App.Run`.** `WorkerService.Stop` blocks
   on a `sync.WaitGroup` until every supervised task returns or
   the shutdown context expires. ADR 0002 explains why this is the
   single biggest production failure mode in Go and why it is fixed
   in the framework, not left to applications. The contract is
   enforced in CI by `go.uber.org/goleak` running in `TestMain` of
   every package that touches goroutines.
3. **Init failure never starts the server.** A returning error from
   any `Initializer.Init` short-circuits the lifecycle before the
   listen socket is created.

---

## 7. The middleware chain

`App.Build()` constructs the request chain in a fixed order. The
order is not configurable on purpose: it is the framework's
strongest opinion. The full chain (when security is enabled and
both metrics and tracing are wired) is:

```text
incoming request
        │
        ▼
[ Tracing       ]   creates SpanContext, propagates W3C trace-context
[ Metrics       ]   records http_requests_total / http_request_duration_seconds
[ RequestID     ]   reads or generates RFC 4122 UUID v4 (no external deps)
[ Recover       ]   converts panics to 500 + structured slog "http.panic"
[ Headers       ]   sets HSTS, X-CTO, X-Frame-Options, Referrer-Policy, ...
[ BodyLimit     ]   wraps r.Body with http.MaxBytesReader (HTTP 413 on overflow)
[ MethodGuard   ]   rejects unknown verbs and traversal-shaped paths
[ AntiBot       ]   blocks empty UA and known scanner signatures
[ RateLimit*    ]   sharded token-bucket per client IP (when enabled)
[ Logger        ]   structured slog "http.request" / "http.response" with correlation
        │
        ▼
[ http.ServeMux ]   routes to the feature handler
```

`*` RateLimit is added only when `security.Config.Enabled` is true
and the limiter was successfully constructed (it brings its own
cleanup background task).

`/healthz`, `/readyz`, and `/metrics` are deliberately served by a
**slim chain** (`RequestID + Recover + Logger` only). Probes never
go through AntiBot, body limits, or rate limits — Kubernetes is not
a malicious actor. The metrics endpoint also bypasses the metrics
recorder so scrape traffic does not pollute application RPS
counters.

---

## 8. Background work: the `WorkerService` contract

Anything that needs to run on a timer for the lifetime of the app
goes through `WorkerService`. Features add tasks via
`BackgroundProvider`; the framework adds two of its own
(`ratelimit-cleanup`, the optional `cache-cleanup`).

The contract:

- One supervised goroutine per task.
- `Start(ctx)` is idempotent. `Add(task)` after start is ignored, so
  the task set is frozen at runtime.
- A panic in `Run` is recovered, logged with a stack trace, and the
  ticker keeps firing. One bad task cannot crash the process.
- `Stop(ctx)` blocks on a `sync.WaitGroup` until every task returns
  or `ctx` expires. A timed-out `Stop` does not consume the only wait:
  callers may invoke `Stop` again after cancelling the run context.
  The framework calls it from `App.Run` with a budget of
  `HTTPShutdownTimeout`.
- Tasks **must** respect their `ctx`. A `Run` that ignores
  cancellation will block graceful shutdown for the full timeout.

Building a periodic cache eviction task is a one-liner thanks to
the `app.CleanupTask[V]` helper, which bridges `*cache.Cache[V]` to
`BackgroundTask` without introducing a circular import.

`pkg/cache` exposes `Len()` and `Stats()` so applications can alert on
unexpected cardinality. For user-controlled keys, configure a
cardinality budget via cache options and still schedule cleanup.
For Instant pages, prefer `pkg/web/instant`: it stores a known set of
immutable page bytes with explicit page and byte budgets.

---

## 9. Configuration: 12-Factor by default

`app.Config` is a flat struct. `app.LoadConfig()` populates it from
environment variables and applies defaults. Every operationally
relevant value has a corresponding env var (see `docs/12-FACTOR.md`
for the full table).

Two design rules:

- **Read once.** `LoadConfig` is called from `main`. The framework
  never re-reads `os.Getenv`. This makes tests trivial: build a
  `Config{}` literal and pass it to `app.New`.
- **Backfill defaults at construction.** `app.New` calls
  `applyHTTPDefaults` so a hand-built `Config` still receives safe
  HTTP timeouts. Operators cannot accidentally ship a Slowloris-
  vulnerable server by forgetting one field.

---

## 10. Observability: interfaces over implementations

Three things are observable out of the box, each in a way that
keeps the framework's dependency graph empty:

- **Logs.** `log/slog` everywhere, structured. Middleware emits
  `http.request` and `http.response` with `request_id`, `trace_id`,
  `span_id`. Audit-grade events use a separate logger name
  (`auth.audit`) so SIEM rules can filter cleanly.
- **Metrics.** `pkg/web/metrics` ships a hand-rolled Prometheus
  text-format (v0.0.4) registry. No `prometheus/client_golang`.
  Counters and histograms are lock-free on the observation path;
  `Registry.Write` takes a single RLock and snapshots before any
  I/O so a slow scrape cannot block writers.
- **Traces.** `pkg/observability` defines `Tracer` and
  `SpanContext` interfaces. The framework uses them to populate
  `slog` correlation fields. Importing `pkg/observability` does
  **not** import `go.opentelemetry.io/*`. A real OTel binding will
  live in the planned `github.com/fastygo/otel` module.

`docs/OBSERVABILITY.md` documents the wire format and the
correlation contract in detail.

---

## 11. Security model in two paragraphs

Defence in depth is wired into the default middleware chain, not
left to features. Headers, body limits, method guard, anti-bot,
rate limit, and panic recovery are on by default; turning them off
requires an explicit `WithSecurity(security.Config{Enabled: false})`
call, which the codebase greps for at review time.

Sessions are HMAC-signed cookies, not JWTs: no library, no
algorithm-confusion class of bugs, easy rotation by changing
`SESSION_KEY`. OIDC handles the auth handshake; everything past
login is opaque-to-server cookie state. Tampered cookies emit a
`session_tampered` audit event so a SIEM can alert on credential
abuse. `docs/SECURITY.md` (Phase 3C) will own the full threat
model.

---

## 12. Versioning, ADRs, and changes to this document

- The framework follows SemVer. Public API lives under `pkg/...`
  and never breaks within a minor version.
- Every architectural change is accompanied by an ADR in
  `docs/adr/`. Existing records: 0001 (split framework/examples),
  0002 (Phase 1 leak hardening), 0003 (Phase 2 observability).
- This document is updated **in the same commit** as the code that
  changes a contract. If a PR modifies the middleware order, the
  lifecycle, or the `Feature` interface and does not touch
  `docs/ARCHITECTURE.md`, the PR is incomplete.
- The `scripts/godoc-audit` tool keeps the in-code documentation
  honest; this document keeps the bird's-eye view honest.

---

## See also

- `docs/QUICKSTART.md` — five-minute path to a running app
- `docs/12-FACTOR.md` — environment variable reference
- `docs/OBSERVABILITY.md` — metrics/traces wire format
- `docs/API_REFERENCE.md` — per-symbol API surface
- `docs/adr/` — accepted architectural decisions
- `examples/` — five reference applications, each a separate Go module
