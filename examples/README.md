# examples/

> **Legacy / maintenance only.** These starters still depend on canceled
> `github.com/fastygo/ui8kit`, `elements`, and `blocks`. **New apps should use
> [`github.com/fastygo/blank`](https://github.com/fastygo/blank)** (`framework` + `templ`).
> See [`../docs/EXAMPLES.md`](../docs/EXAMPLES.md) for the active stack and porting notes.

Each subdirectory is an **independent Go module** that depends on
[`github.com/fastygo/framework`](../README.md) and (for UI examples)
the legacy [`github.com/fastygo/ui8kit`](https://github.com/fastygo/ui8kit) stack.
They remain in the monorepo for CI coverage and migration reference until ported or retired.

## Recommended starting point

| App | Module | Stack |
|---|---|---|
| **Blank** (external) | `github.com/fastygo/blank` | `framework` + `templ` — shell, i18n, theme, mobile sheet |

Copy Blank when starting a product app, CMS admin shell, or internal tool.
Use `examples/instant/` only when you need zero-asset prebuilt HTML (no UI kit at all).

## Local workspace

Use the top-level workspace for cross-stack development:

```bash
cd e:/_@Go/.WorkSpace-Framework
go work sync
```

Legacy UI examples keep local replaces for Framework and UI8Kit:

```text
replace github.com/fastygo/framework => ../..
replace github.com/fastygo/ui8kit => ../../../@UI8Kit
```

Add local `elements` or `blocks` replaces only when an example imports those modules:

```text
require github.com/fastygo/blocks v0.0.0-00010101000000-000000000000

replace github.com/fastygo/blocks => ../../../Blocks
replace github.com/fastygo/elements => ../../../Elements
```

Before publishing or cutting a distributable example, replace pseudo-zero requirements with tagged module versions — or port to Templ and drop UI8Kit. `instant/` is the exception: no asset pipeline and no UI kit.

## ui8px policy

Legacy examples keep their own `.ui8px/` policy tree. Call `ui8px` through `npx`:

```bash
npx ui8px@latest lint ./...
npx ui8px@latest validate aria ./...
```

## Available legacy starters

| Directory | One-liner | Best fit |
|---|---|---|
| [`landing/`](./landing/) | One templ page, no i18n | Static marketing (legacy UI8Kit) |
| [`web/`](./web/) | i18n marketing + optional OIDC | Public product sites (legacy) |
| [`blog/`](./blog/) | Markdown posts at startup | Blogs (legacy Blocks/Elements) |
| [`docs/`](./docs/) | Localized markdown docs | Handbooks (legacy) |
| [`dashboard/`](./dashboard/) | Sidebar shell, auth, CRUD | Admin panels (legacy) |
| [`instant/`](./instant/) | One prebuilt HTML page, no assets | Messenger WebViews, instant articles |
| [`pwa/`](./pwa/) | PWA shell, manifest, service worker | Offline prototypes (legacy UI8Kit) |

## Legacy asset pipeline

- `web/static/css/*.css`, `theme.js`, `ui8kit.js` are vendored by the UI8Kit CLI:
  `go run github.com/fastygo/ui8kit/scripts/cmd/sync-assets web/static`.
- Each example's `package.json` exposes `bun run vendor:assets`.
- Tailwind 4 builds from `web/static/css/input.css` to `web/static/css/app.css`.

## How they consume the framework

Legacy UI example `go.mod` files typically look like:

```go
module github.com/fastygo/framework/examples/<name>

go 1.25.0

require (
    github.com/a-h/templ v0.3.1001
    github.com/fastygo/framework v0.0.0-00010101000000-000000000000
    github.com/fastygo/ui8kit v0.2.5
)

replace github.com/fastygo/framework => ../..
replace github.com/fastygo/ui8kit => ../../../@UI8Kit
```

When copying out of the monorepo, delete local replaces and bump requirements — or migrate to `github.com/fastygo/templ` per `docs/EXAMPLES.md`.

## Adding a new example

**Prefer adding Templ-based samples outside this tree** (e.g. in `github.com/fastygo/blank` or product repos).

If you must add under `examples/` for CI:

1. Create `examples/<name>/` with `cmd/server`, `internal/`, `go.mod`, `README.md`.
2. Document whether it is legacy (UI8Kit) or kit-free (`instant` style).
3. Add to top-level `go.work` and `.github/workflows/ci.yml` matrix.
