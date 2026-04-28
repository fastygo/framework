# Examples Guide

The `examples/` directory contains small, copyable applications. Each
example is an independent Go module with its own `go.mod`, `cmd/server`,
views, static assets, and CSS build pipeline.

Use these examples as starting points, not as framework internals. A real
application should be able to copy one example out of this repository,
remove local `replace` directives, and keep building against tagged module
versions.

## Layer Model

The examples use the framework plus four UI-oriented layers:

| Layer | Module | Owns | Examples |
|---|---|---|---|
| Framework | `github.com/fastygo/framework` | HTTP composition, feature registration, render helpers, locale/security/cache/auth primitives | `pkg/app`, `pkg/web`, `pkg/auth`, `pkg/cache` |
| UI8Kit | `github.com/fastygo/ui8kit` | Low-level UI primitives, shell layout, CSS/JS/font assets | `layout.Shell`, `ui.Title`, `ui.Text`, `ui8kit.js`, `theme.js` |
| Elements | `github.com/fastygo/elements` | Reusable interactive widgets | navigation, dialogs, toggles, account menu |
| Blocks | `github.com/fastygo/blocks` | Reusable page sections and page-level compositions | marketing hero, blog article/list, docs pages, dashboard screens |
| App | `examples/<name>` | Routes, data mapping, i18n content, brand CSS, app-specific behavior | dashboard contacts CRUD, docs markdown registry |

The dependency direction should stay one-way:

```text
App / example -> Framework
App / example -> Blocks / Elements / UI8Kit
Blocks -> Elements / UI8Kit
Elements -> UI8Kit
```

The framework must stay UI-agnostic. UI8Kit, Elements, Blocks, and brand
CSS are application concerns.

## Current Examples

### `examples/landing`

A minimal one-page marketing app. It is useful when you want the smallest
possible composition root and do not need i18n, auth, markdown, or CQRS.

It consumes:

- `framework/pkg/app` for app startup and feature registration.
- UI8Kit assets through `bun run vendor:assets`.
- `Blocks/marketing` for the landing page sections.

Use this starter for simple campaign pages or a first production smoke
test of the stack.

### `examples/web`

A public product website with i18n and an optional authenticated cabinet.

It consumes:

- `framework/pkg/app`, `pkg/web/locale`, and CQRS helpers for feature
  composition.
- `Blocks/marketing` for hero and feature sections.
- `Elements/navigation` and `Elements/toggles` for header actions.
- UI8Kit shell and static assets.

Use this starter for product websites that may later grow an account area.

### `examples/blog`

A markdown-driven blog with a custom app shell. It demonstrates how to
build your own layout while keeping reusable behavior in Elements.

It consumes:

- `pkg/content-markdown` for pre-rendered markdown content.
- `Blocks/editorial` for post lists and articles.
- `Elements/navigation`, `Elements/dialogs`, and `Elements/toggles` for
  the mobile sheet, navigation, close button, and theme/language toggles.
- A custom app-owned shell instead of importing `layout.Shell`.

Use this starter when the layout itself is part of the brand or product
experience.

### `examples/docs`

A localized documentation site backed by embedded markdown.

It consumes:

- `pkg/content-markdown` for the markdown registry.
- `Blocks/docs` for the docs index and article pages.
- `Elements/navigation` and `Elements/toggles` for header navigation and
  locale switching.
- UI8Kit `layout.Shell` because the shell is not the focus of this app.

Use this starter for product docs, handbooks, changelogs, and internal
knowledge bases.

### `examples/dashboard`

An authenticated admin-style demo with a sidebar shell and contacts CRUD.
It is intentionally small, but it shows how protected app screens can be
split into reusable Blocks and Elements.

It consumes:

- `pkg/auth` and dashboard-local middleware for protected routes.
- `Blocks/dashboard` for overview stats, contacts, and login panel.
- `Elements/account` for the account/sign-out/language menu.
- UI8Kit `layout.Shell` for the sidebar and mobile shell.
- App-owned `dashboard-components.css` for Authfly-inspired brand styling.

Use this starter for admin panels, internal tools, or CRM-like apps.

### `examples/instant`

A zero-asset instant article example. It serves one prebuilt HTML document
with inline CSS from a fixed `pkg/web/instant.Store`.

It consumes:

- `pkg/web/instant` for immutable page snapshots with explicit page and byte
  budgets.
- `pkg/app` only for 12-factor HTTP/server configuration.
- No UI8Kit assets, JavaScript, images, fonts, static directory, or runtime
  rendering.

Use this starter for messenger links, instant articles, emergency pages, or
other entry points where first paint must be as small and predictable as
possible.

### `examples/pwa`

A TODO-style installable PWA app shell with onboarding, subscription/payment
mock screens, a web manifest, root-scoped service worker, static cache, and
offline navigation fallback.

It consumes:

- `pkg/app`, `pkg/web/locale`, `pkg/cache`, and `web.CachedRender` for routed
  feature composition, localized pages, and bounded HTML caching.
- `Elements/toggles` for the theme switcher.
- UI8Kit static assets, `theme.js`, `ui8kit.js`, and the Tailwind CSS build.
- App-owned `pwa-components.css` for mobile app-shell styling.

Use this starter for mobile PWAs, offline-first prototypes, installable app
shells, and browser-behavior demos.

## Static Assets And CSS

Each example has a `package.json` with the same basic scripts:

```bash
bun run vendor:assets
bun run build:css
bun run build
```

`vendor:assets` runs:

```bash
go run github.com/fastygo/ui8kit/scripts/cmd/sync-assets web/static
```

That copies UI8Kit CSS, fonts, `theme.js`, and `ui8kit.js` into the
example's `web/static` directory.

From the Framework root, use the aggregate build:

```bash
bun run build:all
```

This syncs UI8Kit assets for every example and rebuilds every example CSS
bundle. Use it after changing UI8Kit assets, Blocks, Elements, or shared
example CSS policy.

## How To Build The Next App

1. Pick the closest example.
   - `landing` for a single page.
   - `web` for a public product site.
   - `blog` for markdown content and custom layout.
   - `docs` for localized documentation.
   - `dashboard` for authenticated app screens.
   - `instant` for zero-asset prebuilt HTML.
   - `pwa` for installable/offline app shells.

2. Copy the example into a new module or app directory.

3. Keep `cmd/server/main.go` as the composition root.
   This is where you load config, create repositories/services, register
   features, and call `app.New(...).WithFeature(...).Build()`.

4. Keep business data in the app.
   Blocks should receive props. They should not know about repositories,
   request objects, auth sessions, or app-specific i18n bundles.

5. Reuse Blocks for page structure.
   If a section is reusable across apps, promote it to `Blocks`. If it is
   only brand copy or one-off layout, leave it in the app.

6. Reuse Elements for widgets.
   Buttons that open dialogs, navigation groups, language toggles, account
   menus, and close buttons belong in `Elements` when they are reusable
   behavior rather than page content.

7. Keep brand styling in the app.
   App CSS can define semantic classes such as `dashboard-card`,
   `blog-page`, or `landing-hero`. Shared modules should expose class
   override structs so each app can keep its own visual language.

8. Validate the result.

```bash
templ generate ./...
go test ./...
go build ./...
npx ui8px@latest lint ./...
npx ui8px@latest validate aria ./...
```

For the whole examples workspace:

```bash
cd examples
npx ui8px@latest lint
npx ui8px@latest validate aria
```

## When To Create Blocks Or Elements

Create a Block when the thing is a page section or full page composition:

- hero sections
- feature grids
- post lists
- docs article wrappers
- dashboard overview cards
- CRUD page shells

Create an Element when the thing is an interactive or reusable widget:

- mobile menu trigger
- dialog close button
- header navigation
- theme toggle
- language toggle
- account/sign-out menu

Keep it in the app when it is only data mapping, route handling, brand
copy, feature-specific forms, or app-owned styling.

## Module Wiring During Local Development

When an example imports sibling modules, its `go.mod` uses local replaces:

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

When the app leaves the monorepo, delete local replaces and depend on
tagged versions instead.
