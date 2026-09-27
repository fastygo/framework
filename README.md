# fastygo/framework

[![ci](https://github.com/fastygo/framework/actions/workflows/ci.yml/badge.svg)](https://github.com/fastygo/framework/actions/workflows/ci.yml)

A small, opinionated Go framework for the HTTP process: composition, sessions,
middleware, and graceful shutdown. Fonts and mail are separate modules in this
repository. Templ rendering, markdown, and view data live in
[`github.com/fastygo/modules`](https://github.com/fastygo/modules) and are not
required by the root `go.mod`.

**UI stack for new apps:** [`github.com/fastygo/templ`](https://github.com/fastygo/templ)
(primitives + composites) on top of this framework. Reference shell:
[`github.com/fastygo/blank`](https://github.com/fastygo/blank) (`framework` + `templ` only).
The `examples/*` modules below still use the **legacy** UI8Kit/Elements/Blocks stack — see
[`docs/EXAMPLES.md`](./docs/EXAMPLES.md).

This repository contains:

- **The framework module** (`./`) — only the code under `pkg/` is part of
  the public API. The framework module never depends on application code,
  on any specific UI kit, or on a specific i18n bundle.
- **Five example sites** (`./examples/*`) — each one is an independent Go
  module with its own `go.mod`, `cmd/server`, templates, and CSS pipeline.
  They are designed to be cloned out into their own repositories as soon
  as you outgrow this monorepo.

## Why is the framework split from the sites?

Imagine four developers who all `git clone` this repository to start their
own projects:

| Developer | Wants to build | What they actually need |
|---|---|---|
| 1 | Blog + product showcase | `pkg/app`, `pkg/web`, `pkg/cache`, `fastygo/templ`, content library |
| 2 | CRM + chat + internal docs | `pkg/app`, `pkg/web`, `pkg/auth`, eventually WebSockets |
| 3 | Marketplace with seller/buyer/admin cabinets | `pkg/app` × N composition roots, role-based middleware |
| 4 | Social network with feed + messaging | `pkg/app`, `pkg/web`, real-time, no UI kit |

If everyone clones one monolith they each have to delete the demo
welcome/docs modules, rewrite `cmd/server/main.go`, and inevitably drift
away from upstream. By making the framework a **pure library module** they
all `require github.com/fastygo/framework v0.x.y` and only pull in what
they import.

## Repository layout

```
.
├── pkg/                          # root module: process, HTTP, sessions
│   ├── app/                      # AppBuilder, Handler, Run, workers
│   ├── auth/                     # cookie sessions + OpenID Connect client
│   ├── cache/                    # sharded TTL cache
│   ├── core/                     # CQRS dispatcher, errors, behaviors
│   ├── observe/                  # tracing interface, no SDK
│   └── web/                      # middleware, security, health, locale, JSON
├── pkg/fonts/                    # separate module: Outfit files
├── pkg/mail/                     # separate module: IMAP and JMAP
├── scripts/
└── go.work                       # local workspace only
```

## What is in `pkg/` (and what is not)

| Package | Purpose | Notes |
|---|---|---|
| `pkg/app` | `AppBuilder`, `Feature`, optional interfaces (`Initializer`, `Closer`, `HealthChecker`, `BackgroundProvider`), config, worker service | Foundation of every app |
| `pkg/auth` | HMAC-signed cookie sessions, OpenID Connect client | Use it for SSO and demo login flows |
| `pkg/cache` | Sharded TTL cache | Used by `github.com/fastygo/modules/render` |
| `pkg/core` | Domain errors, base entities | Tiny, no third-party deps |
| `pkg/core/cqrs` | Dispatcher with pipeline behaviors | Optional — features may use it or not |
| `pkg/observe` | `Tracer` interface and no-op | No tracing SDK |
| `pkg/web` | JSON, error handler | Does not import templ |
| `pkg/web/i18n` | Generic embedded JSON locale store | Used by every example with i18n |
| `pkg/web/locale` | Request locale negotiator (query, cookie, Accept-Language) | Pure helper |
| `pkg/web/middleware` | request-id, logger, panic recovery | Wired in by `AppBuilder` |
| `pkg/web/security` | secure headers, body limit, antibot, ratelimit, secure file server | Configurable, opt-out friendly |
| `pkg/fonts` | Outfit font files | Own module |
| `pkg/mail` | IMAP and JMAP client | Own module |

The framework **does not** ship templates, JSON locale bundles, demo
features, or a default UI kit. Those concerns live in `examples/*` (which
import the framework as a regular Go module).

## Examples

> **New projects:** start from [`github.com/fastygo/blank`](https://github.com/fastygo/blank)
> (`framework` + `templ`). The modules under `examples/` are **legacy starters** that still
> depend on canceled `ui8kit` / `elements` / `blocks` — useful for maintenance and migration
> reference, not the target stack. See [`docs/EXAMPLES.md`](./docs/EXAMPLES.md).

Each legacy example is an **independent Go module**. Pick the one that resembles
the project you want to build and clone its directory into a new
repository. The first thing to delete from the copied example is the
`replace github.com/fastygo/framework => ../..` directive in `go.mod` —
that line only exists so the example resolves the local framework module
during monorepo development.

| Example | Routes | Highlights |
|---|---|---|
| `examples/landing` | `/` | Single page, no i18n, no CQRS — the absolute minimum |
| `examples/web` | `/`, `/cab/`, `/auth/...` | i18n (en/ru), optional OIDC cabinet |
| `examples/blog` | `/`, `/posts/{slug}` | Markdown posts pre-rendered at startup |
| `examples/docs` | `/`, `/{slug}` | Localized documentation site |
| `examples/dashboard` | `/`, `/contacts`, `/auth/...` | Sidebar shell + auth middleware + CRUD scaffold |

See each example's `README.md` for the local quick start and
[`docs/EXAMPLES.md`](./docs/EXAMPLES.md) for the active Templ stack and legacy example notes.

## Local development with `go.work`

The repo ships with a `go.work` file so that running `go build` from the
framework or any example automatically resolves the local copy of every
sibling module. There is nothing to install — Go picks up `go.work`
automatically.

```bash
bun install
go test ./...                        # framework tests
make examples                        # build every example (assets + CSS + Go)
(cd examples/web && make dev)        # full dev loop for one example
```

CI runs `make ci` (= `go test ./...` + the no-root-imports check) on the
framework module and `go build ./...` on every example.

## Pre-requisites

- Go `1.25.0` or newer
- Bun `1.3+` (for example CSS + JS asset builds). Bun is not a Go dependency.
- Templ rendering comes from [`github.com/fastygo/modules/render`](https://github.com/fastygo/modules). An application that generates templates pins CLI and library together: `go get -tool github.com/a-h/templ/cmd/templ@v0.3.1001`, then `go tool templ`.

## Releasing the framework

The framework is a normal Go module. To release a new version:

1. Bump the framework only (don't touch `examples/`).
2. Tag the commit with a SemVer tag, e.g. `v0.6.0`.
3. Examples stay on `replace` directives during monorepo development.
   When extracted to their own repositories they bump the
   `require github.com/fastygo/framework vX.Y.Z` line instead.

## Project boundaries

- The framework module is **never** allowed to import packages outside
  `pkg/`. The check is enforced by `scripts/check-no-root-imports.go` and
  runs in CI.
- Examples are **allowed** to depend on the framework and on any third-party
  library they need. Legacy examples still pin `ui8kit` / `elements` / `blocks`;
  new apps should use `github.com/fastygo/templ` instead (see `docs/EXAMPLES.md`).
  Examples live behind their own `go.mod` precisely so they can evolve independently.

## Versioning

See [`CHANGELOG.md`](./CHANGELOG.md) for the per-release summary and
[`RELEASE.md`](./RELEASE.md) for the maintainer checklist. Architecture
decisions are recorded under [`docs/adr/`](./docs/adr/).

Highlights:

- **v0.1.0** — graceful worker shutdown, configurable HTTP server
  timeouts (`APP_HTTP_*`), bounded TTL cache via `app.CleanupTask`,
  `goleak` + `golangci-lint` + `go vet` in CI.
- **v0.2.0** — observability without the SDK tax: `pkg/web/health`,
  `pkg/web/metrics` (manual Prometheus expfmt), interface-only
  tracer (now `pkg/observe`), structured `auth.audit` events. Zero
  new external dependencies. See
  [`docs/OBSERVABILITY.md`](./docs/OBSERVABILITY.md) for the operator
  guide and [`docs/12-FACTOR.md`](./docs/12-FACTOR.md) for the full
  env-var matrix.

## License

MIT.
