# Examples Guide

## Active stack (new apps)

New FastyGo apps use **Framework + Templ**, not UI8Kit/Elements/Blocks.

| Layer | Module | Owns |
|---|---|---|
| Framework | `github.com/fastygo/framework` | HTTP composition, features, render helpers, locale/security/cache/auth |
| Templ registry | `github.com/fastygo/templ` | `utils/`, `ui/` primitives, `components/` composites |
| App | your module | `internal/ui/` shell, `internal/views/` pages, fixtures, brand CSS |

**Dependency direction:**

```text
App -> internal/ui + internal/views -> github.com/fastygo/templ (utils, ui, components)
App -> github.com/fastygo/framework
```

**Reference implementation:** [`github.com/fastygo/blank`](https://github.com/fastygo/blank) —
minimal shell (top nav, mobile sheet, dark theme, EN/RU locales) on `framework` + `templ` only.

**Product apps** (AppCMS, AppCRM, AppSuite, BuildY) follow the same Templ stack via Platform BFF;
see `@Platform/.cursor/rules/product-stack-current-slice.mdc`.

### Starting a new app

1. Copy or depend on `github.com/fastygo/blank` (or scaffold with the same `go.mod` shape).
2. Vendor bricks from `github.com/fastygo/templ` (`ui/`, `components/`, `utils/`).
3. Keep `cmd/server/main.go` as the composition root (`pkg/app`, feature registration).
4. Put shell chrome in `internal/ui/layout/`; page bodies in `internal/views/`.
5. Optional client ARIA: committed `@ui8kit/aria` dialog bundle in `web/static/js/ui8kit.js` (Blank pattern) — not a `ui8kit` Go import.
6. Run `templ generate`, `ui8px lint`, and `go test ./...`.

---

## Legacy examples (`examples/` — archived stack)

> **Status: legacy / maintenance only.** The modules under `examples/` still depend on
> `github.com/fastygo/ui8kit`, `github.com/fastygo/elements`, and `github.com/fastygo/blocks`.
> These Go modules and the `go-ui8kit/en` profile are **canceled** for new work.
> CI keeps building them until they are retired or ported to `fastygo/templ`.
> **Do not copy these into new products** — use Blank + Templ instead.

Each legacy example is an independent Go module with its own `go.mod`, `cmd/server`,
views, static assets, and CSS build pipeline.

### Legacy layer model

| Layer | Module | Status |
|---|---|---|
| Framework | `github.com/fastygo/framework` | Active |
| UI8Kit | `github.com/fastygo/ui8kit` | Canceled |
| Elements | `github.com/fastygo/elements` | Canceled |
| Blocks | `github.com/fastygo/blocks` | Canceled |
| App | `examples/<name>` | Legacy starters |

```text
App / example -> Framework
App / example -> Blocks / Elements / UI8Kit   # legacy only
```

The framework stays UI-agnostic. UI kit choice is an application concern.

### Legacy starters

#### `examples/landing`

Minimal one-page marketing app. UI8Kit assets + `Blocks/marketing`.

#### `examples/web`

Public product website with i18n and optional OIDC cabinet. UI8Kit shell + Blocks/Elements.

#### `examples/blog`

Markdown-driven blog with custom shell. Blocks/editorial + Elements navigation/dialogs.

#### `examples/docs`

Localized documentation site. Blocks/docs + UI8Kit `layout.Shell`.

#### `examples/dashboard`

Authenticated admin demo with sidebar and contacts CRUD. Blocks/dashboard + Elements/account.

#### `examples/instant`

Zero-asset instant article — **no UI kit**. Uses `pkg/web/instant` only. Still a valid pattern for ultra-fast entry pages.

#### `examples/pwa`

Installable PWA shell. UI8Kit static assets + Elements/toggles.

### Legacy static assets

Legacy examples vendor UI8Kit CSS/JS via:

```bash
bun run vendor:assets   # go run github.com/fastygo/ui8kit/scripts/cmd/sync-assets web/static
bun run build:css
```

From the Framework root, `bun run build:all` syncs UI8Kit assets for every legacy example.

### Porting legacy → Templ

When retiring a legacy example:

1. Replace `ui8kit` / `elements` / `blocks` imports with `github.com/fastygo/templ` bricks.
2. Move shell markup into `internal/ui/layout/` (see Blank).
3. Keep framework feature composition unchanged (`pkg/app`, `pkg/web`, auth, i18n).
4. Replace `vendor:assets` UI8Kit sync with app-owned Tailwind build + optional `@ui8kit/aria` JS bundle.
5. Add import guards: no `github.com/fastygo/ui8kit` in `go.mod` or templates.

### Legacy local `replace` wiring

```go
require (
    github.com/fastygo/blocks v0.0.0-00010101000000-000000000000
    github.com/fastygo/elements v0.0.0-00010101000000-000000000000
    github.com/fastygo/framework v0.0.0-00010101000000-000000000000
    github.com/fastygo/ui8kit v0.2.5
)

replace github.com/fastygo/framework => ../..
replace github.com/fastygo/blocks => ../../../Blocks
replace github.com/fastygo/elements => ../../../Elements
replace github.com/fastygo/ui8kit => ../../../@UI8Kit
```

When an app leaves the monorepo, delete local replaces and depend on tagged versions — or migrate to Templ and drop UI8Kit entirely.

### Validation (Templ and legacy)

```bash
templ generate ./...
go test ./...
go build ./...
npx ui8px@latest lint ./...
npx ui8px@latest validate aria ./...
```
