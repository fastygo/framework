# API Reference (Go)

This document reflects the actual API surface of the `examples/docs` application in this repository.

## 1) BFF-style role in this example

`examples/docs` behaves as a server-rendered BFF for documentation pages:

- markdown content is converted to HTML on the server,
- full SSR responses are returned,
- middleware/localization/theme/error concerns are handled in app/wire layers,
- presentation remains thin and deterministic.

## 2) Root composition: `cmd/server/main.go`

`main` performs:

- `cfg := app.LoadConfig()`
- `content.NewLibrary(...)` from `pkg/content-markdown`
- `application := app.New(cfg)`
- `WithSecurity(security.LoadConfig())`
- `WithLocales(app.LocalesConfig{... QueryStrategy ... Cookie ...})`
- `WithFeature(docsfeature.New(library))`
- `Build()` and `Run(ctx)`

This wires all docs routes through one HTTP entrypoint.

## 3) `pkg/content-markdown` API

File: `pkg/content-markdown/content.go`

- `type PageMeta struct { Slug, Title string }`
- `type LibraryOptions struct`
  - `FS fs.ReadFileFS`
  - `Pages []PageMeta`
  - `Locales []string`
  - `DefaultLocale string`
  - `DefaultTitle string`
- `type PageRender struct { Slug, Title, HTML string }`
- `type Library struct { pages map[string]map[string]PageRender }`
- `func NewLibrary(opts LibraryOptions) (*Library, error)`
- `func (l *Library) Page(locale, slug string) (PageRender, bool)`
- `func readPages(opts LibraryOptions, out map[string]map[string]PageRender) error`
- `func localize(locale, fallback string) string`

`Page` returns pre-rendered HTML content.

## 4) Docs feature API: `internal/site/docs/feature.go`

- `var Pages = []content.PageMeta{{Slug: "quickstart", ...}, ...}`
- `type Feature struct { library *content.Library }`
- `func New(library *content.Library) *Feature`
- `func (f *Feature) ID() string`
- `func (f *Feature) NavItems() []web.NavItem`
- `func (f *Feature) Routes(mux *http.ServeMux)`
- `func (f *Feature) renderPage(...)`

`Routes` maps `/`, `/quickstart`, `/developer-guide`, `/api-reference`.

## 5) View-layer API

### `internal/site/views/models.go`

- `type DocsPageData struct { Title string; HTMLContent string }`
- `type DocsIndexData struct { Title string }`

### `internal/site/views/layout.templ`

- `templ Layout(data LayoutData)` — wraps content in shared `ui8layout.Shell`.

### `internal/site/views/page.templ`

- `templ DocsPage(data DocsPageData)` — renders markdown HTML in `<article class="prose max-w-none">`.

### `internal/site/views/index.templ`

- `templ DocsIndex(data DocsIndexData)` — landing page for docs.

### `internal/site/views/partials/language_toggle.templ`

- renders the locale switch component.

## 6) Localization API: `internal/site/i18n`

- `internal/site/i18n/embed.go` and `internal/site/i18n/{en,ru}/common.json`
- exported helpers:
  - `func Locales() []string`
  - `func Decode[T any](locale string, section string) (T, error)`
  - `func Load(locale string) (Localized, error)`

## 7) Framework packages used by this example

The docs app composes core packages from this repository:

- `pkg/app`
  - `app.LoadConfig`, `app.New`, `app.LocalesConfig`, `app.WithLocales`, `app.WithFeature`, `app.App.Run`
  - `pkg/app.Feature`, `NavItem`, `AppBuilder`, middleware chain.
- `pkg/web`
  - `web.CachedRender`, `web.HandleError`
- `pkg/web/middleware`
  - `RequestIDMiddleware`, `RecoverMiddleware`, `LoggerMiddleware`
- `pkg/security`
  - `security.LoadConfig`
- `pkg/content-markdown`
  - shared markdown renderer used by docs and other apps.

## 8) Adding a new docs page

1. Add markdown under `examples/docs/content/i18n/<locale>/<slug>.md`.
2. Add matching `PageMeta` in `internal/site/docs/feature.go`.
3. Extend locale strings in `internal/site/i18n` if needed.
4. Restart dev server (`make dev` or `go run ./cmd/server`).

## 9) Runtime request flow

- `main` -> `app.New` -> `docsfeature.New` -> `app.Build()`.
- request -> middleware stack -> docs route handler.
- `content.Library.Page(locale, slug)` -> `DocsPage` -> `templ` render.
- SSR HTML response is sent via `http.ResponseWriter`.

## Reader mode note

There is no dedicated reader implementation in this example backend.
If Reader mode appears only for some pages, it is driven by browser readability heuristics and is sensitive to content structure (long prose blocks are favored over code-heavy sections).
