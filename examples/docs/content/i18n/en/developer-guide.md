# FastyGO Docs Developer Guide

This document describes the current `examples/docs` application in this repository.

## What this example is

`examples/docs` is a server-rendered documentation site built with `net/http` + `templ`.

The docs app is focused: it serves markdown pages through a reusable content renderer and a small feature module.

## Architecture

### 1) Application entrypoint — `cmd/server/main.go`

`main.go` performs three steps:

1. Loads config via `app.LoadConfig()`.
2. Creates the content library from `pkg/content-markdown`:
   - reads markdown files from embedded FS (`examples/docs/content/embed.go`),
   - uses `docsfeature.Pages` for available slugs,
   - builds localized cache `library`.
3. Builds the application via `app.New(cfg)`:
   - enables `security.LoadConfig()`
   - configures `WithLocales(...)` with query strategy (`lang`) and cookie persistence
   - registers `docsfeature.New(library)`
   - `Build()` and `Run(ctx)`.

### 2) Content layer — `pkg/content-markdown`

`pkg/content-markdown`:

- iterates `PageMeta` descriptors (`Slug`, `Title`),
- reads markdown from `embed.FS`,
- renders markdown to HTML using `goldmark`,
- caches resulting `PageRender` entries.

Pages are then requested via `Page(locale, slug)`.

### 3) Docs feature — `internal/site/docs/feature.go`

`docsfeature` is the only feature registered in this example:

- keeps `docs.Pages` (`quickstart`, `developer-guide`, `api-reference`),
- registers routes:
  - `GET /`
  - `GET /quickstart`
  - `GET /developer-guide`
  - `GET /api-reference`,
- sets feature navigation items,
- renders content through `views.DocsPage(...)`.

### 4) Views — `internal/site/views/*`

- `layout.templ` — shared shell layout
- `page.templ` — renders markdown HTML (`@t.Raw(data.HTMLContent)`)
- `index.templ` — docs index page
- `partials/language_toggle.templ` — locale switch control

The article container is `<article class="prose max-w-none">`, so all docs get the same prose styling.

### 5) Locales — `internal/site/i18n`

- `common.json` for `en` and `ru` holds shell/UI labels.
- Locale data is loaded via `go:embed` (`internal/site/i18n/embed.go`) and resolved through `app.LocalesConfig`.

## Request flow

1. HTTP request enters the `AppBuilder` chain (`request id`, `recover`, `logger`).
2. `docsfeature` resolves the slug from route.
3. `contentMarkdown.Library.Page(locale, slug)` returns cached rendered HTML.
4. `templ` builds the final page and writes HTML response.

## Reader mode behavior

Why Reader mode appears more consistently for some pages:

- Browser heuristics are text-first and favor long narrative pages,
- `developer-guide` and `api-reference` are generally richer in prose,
- pages with many code blocks and short explanations (like `quickstart`) can be treated differently.

This is browser-side behavior, not a bug in the markdown renderer.

## Adding a new docs page

1. Add markdown to `examples/docs/content/i18n/<locale>/<slug>.md`.
2. Add matching `PageMeta` to `examples/docs/internal/site/docs/feature.go`.
3. Add/update locale text in `internal/site/i18n` when needed.
4. Restart dev server (`make dev` or `go run ./cmd/server`).

## File map for this example

- `cmd/server/main.go` — bootstrap and run.
- `internal/site/docs/feature.go` — docs feature + routing.
- `internal/site/views/{layout.templ,page.templ,index.templ}` — templates.
- `internal/site/views/partials/language_toggle.templ` — language switcher.
- `internal/site/i18n/{en,ru}/common.json` — shell/localized strings.
- `internal/site/i18n/embed.go` — locale loader and helpers.
- `examples/docs/content/*/*.md` — source markdown documents.
- `examples/docs/content/embed.go` — `go:embed` manifest.

## Environment

- `APP_BIND` (default: `127.0.0.1:8081`)
- `APP_STATIC_DIR` (default: `web/static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)

## Troubleshooting

- **404 after adding a page**: check slug exists in `docs.Pages` and markdown file exists.
- **Reader mode not offered**: add descriptive text between code blocks.
- **Template errors**: run `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`.
