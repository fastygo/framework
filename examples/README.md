# examples/

Each subdirectory is an **independent Go module** that depends on
[`github.com/fastygo/framework`](../README.md) and (usually)
[`github.com/fastygo/ui8kit`](https://github.com/fastygo/ui8kit). They are
designed to be cloned out into their own GitHub repositories the moment
the team behind them is ready.

## Available starters

| Directory | One-liner | Best fit |
|---|---|---|
| [`landing/`](./landing/) | One templ page, no i18n, one feature | Static marketing sites, conference pages |
| [`web/`](./web/) | i18n marketing site + optional OIDC cabinet | Public-facing product websites |
| [`blog/`](./blog/) | Markdown posts pre-rendered at startup | Personal/team blogs, changelog feeds |
| [`docs/`](./docs/) | Localized markdown documentation site | Product docs, internal handbooks |
| [`dashboard/`](./dashboard/) | Sidebar shell, auth middleware, contacts CRUD | Internal tools, CRMs, admin panels |

## How they share assets

- `web/static/css/*.css` and `web/static/js/ui8kit.js` are **synced from
  UI8Kit** via `../../scripts/sync-ui8kit-css.sh`. Each example's
  `package.json` exposes that as `npm run sync:ui8kit`.
- The Outfit font files live in `pkg/fonts/` and are copied into each
  example's `web/static/fonts/` by the same script.
- Tailwind 4 builds CSS from `web/static/css/input.css` to
  `web/static/css/app.css` (gitignored).

## How they consume the framework

Every example's `go.mod` looks like:

```go
module github.com/fastygo/framework/examples/<name>

go 1.25.0

require (
    github.com/a-h/templ v0.3.1001
    github.com/fastygo/framework v0.0.0-00010101000000-000000000000
    github.com/fastygo/ui8kit v0.2.5
)

replace github.com/fastygo/framework => ../..
```

The `replace` directive resolves the local framework module during
monorepo development. When you copy an example out into its own
repository, **delete the `replace` line** and bump the `require` to a
tagged framework release.

## Adding a new example

1. Create `examples/<name>/` with at least:
   - `cmd/server/main.go` (composition root)
   - `internal/site/...` for templates and features
   - `web/static/css/input.css` for Tailwind
   - `go.mod` with the `replace` directive above
   - `package.json` exposing `sync:ui8kit`, `dev:css`, `build:css`, and `build`
   - `Makefile` exposing `dev`, `build`, `sync-ui8kit`, `css`, `generate`
   - `README.md` describing the goal and quick start
2. Add `./examples/<name>` to the top-level `go.work`.
3. Add the example to `.github/workflows/ci.yml` under the `build-examples` matrix.
