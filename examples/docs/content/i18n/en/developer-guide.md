# FastyGO Framework Developer Guide (Phase 0)

This guide describes the current implemented architecture in English, focused on onboarding a developer quickly.

## What this repository is

This project is a **Phase 0 dashboard skeleton** (zero domain logic), including:

- UI8Kit-powered dashboard shell (`Shell`, sidebar, mobile sheet)
- SSR via `a-h/templ`
- Theme toggle and locale switcher
- CQRS request flow
- Typed feature composition
- Embedded JSON fixtures for i18n (`en`, `ru`)

The goal of Phase 0 is to provide a fast, stable baseline you can copy for new projects.

---

## Implemented Architecture

### 1) Composition Root

`cmd/server/main.go` is the composition root:

- loads configuration from environment variables
- creates a CQRS dispatcher
- registers pipeline behaviors:
  - validation behavior
  - logging behavior
- registers handlers/features
- builds the app and starts HTTP server with graceful shutdown

This keeps bootstrapping and wiring in one place.

### 2) App Kernel

`pkg/app` provides the host application abstraction:

- `config.go`
  - loads runtime settings (`APP_BIND`, `APP_STATIC_DIR`, `APP_DEFAULT_LOCALE`, `APP_AVAILABLE_LOCALES`, `APP_DATA_SOURCE`)
  - validates locale list
- `feature.go`
  - `Feature` interface:
    - `ID() string`
    - `Routes(*http.ServeMux)`
    - `NavItems() []NavItem`
  - `NavItem` model: `Label`, `Path`, `Icon`, `Order`
- `builder.go`
  - `AppBuilder` API:
    - `New(cfg).WithFeature(feature).Build().Run(ctx)`
  - collects nav items from all features and sorts them
  - builds middleware chain + static file handler (`/static/...`)

### 3) Web Platform Layer

`pkg/web` contains shared HTTP/templ utilities:

- `middleware/chain.go` - middleware chain composition utility
- `middleware/request_id.go` - assigns request id and passes via context/header
- `middleware/recover.go` - panic recovery
- `middleware/logger.go` - structured request logging and response meta (`status`, `duration_ms`, `size`)
- `render.go` - `templ` render helper to HTTP response
- `error_handler.go` - maps `DomainError` to HTTP response
- `page.go` - shared page DTOs used by views

### 4) CQRS Core (lightweight)

Implemented in `pkg/core/cqrs`:

- `command.go` - marker interfaces and handler interfaces:
  - `Command`, `Query`, `CommandHandler`, `QueryHandler`
- `behavior.go` - `PipelineBehavior` interface
- `dispatcher.go` - registers and dispatches handlers
- `behaviors/validation.go` - optional `Validate() error` hook
- `behaviors/logging.go` - logs request lifecycle with duration

Important behavior:
- `cqrs.RegisterQuery(...)` registers typed query handlers
- `cqrs.DispatchQuery(ctx, dispatcher, query)` executes pipeline and handler

This is intentionally minimal but fully wired and used in runtime.

### 5) View Layer (templ)

`internal/site/web/views/` contains SSR templates:

- `layout.templ`
  - wraps all pages into `ui8layout.Shell`
  - provides header actions and nav
- `welcome.templ`
  - greeting page with `ui.Title`, `ui.Text`, `ui.Button`
- `nav.templ`
  - inserts JS (`/static/js/app-shell.js`) into shell head
- `models.go`
  - strongly typed data models for templates
- `partials/language_toggle.templ`
  - language switch control data attributes for browser script


## Locale strategy and language switcher

Examples now centralize locale negotiation in the app layer so features do not pass
`defaultLocale`, `available`, or `negotiator` explicitly.

### 1) Configure i18n in `cmd/server/main.go`

Use `app.WithLocales(...)` once in the composition root:

```go
builder := app.New(cfg).WithLocales(app.LocalesConfig{
    Default:   cfg.DefaultLocale,
    Available: cfg.AvailableLocales,
    Strategy: &locale.QueryStrategy{
        Param:    "lang",
        Aliases:  []string{"translate"},
        Available: cfg.AvailableLocales,
        Default:   cfg.DefaultLocale,
        ValueMap: map[string]string{
            "english": "en",
            "russian": "ru",
        },
    },
    Cookie: locale.CookieOptions{
        Enabled: true,
        Name:    "lang",
    },
})
```

### 2) Strategy options available

`app.WithLocales(...)` accepts:

1. `QueryStrategy` — switch by query parameter (`?lang=en`).
2. `PathPrefixStrategy` — switch by URL segment (`/en/...`, `/ru/...`).
3. `CookieOptions` (default-on per plan) — persist choice across sessions.

You can pass a custom strategy and keep cookie persistence by setting `Cookie.Enabled`.

### 3) Read active locale inside handlers

In any handler, resolve the active locale from request context:

```go
loc := locale.From(r.Context())
```

### 4) Render the toggle with context helpers

```go
language := view.BuildLanguageToggleFromContext(
    r.Context(),
    view.WithLocaleLabels(bundle.Common.Language.LocaleLabels),
)
```

### 5) Toggle template and progressive enhancement

Use `NextHref` for a normal, no-JS flow:

```templ
<a href={ t.URL(data.NextHref) }>
```

Enable SPA-style switching (fetch-and-swap) by opting into strategy mode in
`app.WithLocales`:

```go
builder := app.New(cfg).WithLocales(app.LocalesConfig{
    // ...
    SPA: true,
})
```

Then include `data-ui8kit-spa-lang="1"` in the rendered toggle and keep
`data-spa-target="main"` (default) so the language change updates the page
without a full reload while still remaining as a normal link if JS is unavailable.

Current guide examples use this behavior in `examples/web`.

Generated files (`*_templ.go`) are produced by `templ generate`.

### 6) I18N / Content layer

`internal/site/web/i18n/` contains embedded content:

- `en/common.json`
- `en/welcome.json`
- `ru/common.json`
- `ru/welcome.json`

`internal/site/web/i18n/embed.go`:
- uses `go:embed` for JSON files
- decodes locale-specific content by section
- provides:
  - `Load(locale)` for full page payload
  - `Locales()` list

Business data (Phase 0) is currently static content, not external service-based.

### 7) Frontend behavior layer

`internal/site/web/static/js/app-shell.js` handles:

- persisted theme toggle (`dark` / `light`) in `localStorage`
- locale preference persistence in `localStorage`
- redirect-on-load behavior to align browser/local stored/default locale
- click-to-switch locale handler

`internal/site/web/static/css/input.css` imports Tailwind and UI8Kit styles.

---

## Business logic implemented in this phase

Phase 0 has a single feature and one public interaction:

### Welcome Feature (`internal/infra/features/welcome`, `internal/application/welcome`)

- `WelcomeQuery` + `WelcomeQueryHandler` (`internal/application/welcome/handler.go`)
  - loads data from fixtures
  - validates locale
- `welcome.Module` (`internal/infra/features/welcome/module.go`)
  - registers route `/`
  - provides one nav item `Welcome`
  - resolves effective locale (from query string or defaults)
  - maps `locale` -> `WelcomePageData` and renders `views.Layout + views.WelcomePage`
- No external integrations, databases, or authentication logic in this phase

Current business-level capabilities:
- render welcome page from fixture content
- switch language UI + URL (`?lang=...`) + persistence
- theme persistence
- navigation rendering from feature nav metadata

---

## Request and rendering flow

1. HTTP request enters `AppBuilder` chain
2. Middleware chain runs:
   - request id
   - recover
   - logger
3. Route handler (`welcome.Module`) resolves locale and dispatches `WelcomeQuery`
4. `Dispatcher` executes behaviors and query handler
5. Fixtures are loaded for the locale
6. Data is passed to `views.Layout + views.WelcomePage`
7. `templ` renders HTML response

---

## Runtime sequence (important for debugging)

- `GET /` with `lang` present -> uses that locale
- `GET /` without `lang` -> falls back to `APP_DEFAULT_LOCALE` (configured default)
- Locale toggle click -> updates URL via `?lang=<next>` (or removes for default locale) and re-renders page
- Theme button -> updates `html.dark` class and persists preference

---

## Directory map

- `cmd/server/main.go` – app composition and run
- `pkg/core` – domain primitives (`Entity`, `DomainError`), CQRS
- `pkg/app` – app builder/config/feature contracts
- `pkg/web` – middleware, render and error helpers
- `internal/application/welcome` – use-case/query-handler layer
- `internal/infra/features/welcome` – HTTP/templ adapter
- `internal/site/web/views` – templ views and partials
- `internal/site/web/i18n` – embedded JSON i18n content
- `internal/site/web/static/css` – Tailwind + UI8Kit CSS pipeline
- `internal/site/web/static/js/app-shell.js` – theme/locale behavior
- `scripts/sync-ui8kit-css.sh` – copies UI8Kit CSS from module cache
- `docs/QUICKSTART.md` – quick run instructions

---

## Environment variables

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `internal/site/web/static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)

---

## Adding a new feature (recommended)

1. Create `internal/application/<name>/` and `internal/infra/features/<name>/`
2. Add:
   - feature module implementing `pkg/app.Feature`
   - query/handler pair if you need CQRS style flow
3. Register feature in `cmd/server/main.go`:
   - `WithFeature(newFeature)`
4. Add templates under `internal/site/web/views/` and wire model DTOs
5. Add fixtures or dedicated data source integration

Because `AppBuilder` composes `NavItems` automatically, each feature contributes its own menu items.

---

## Commands

- Install dependencies:
  - `npm install`
  - `go mod download`
- Sync UI8Kit styles:
  - `npm run sync:ui8kit`
- Dev run:
  - `npm run build:css`
  - `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`
  - `go run ./cmd/server`
  - or `make dev` (if available)
- Build:
  - `make build` and run `./bin/framework`
- Tests:
  - `go test ./...`
- Lint + architecture checks:
  - `make lint` (runs tests and no-root import check)
  - `make ci` (same as CI pipeline command)
  - `make lint-ci` (same as `make ci`, alias)
  - Windows fallback without `make`:
    - `go test ./...`
    - `go run ./scripts/check-no-root-imports.go`
- CSS watch:
  - `npm run dev:css`

## CI and lint checks

- `scripts/check-no-root-imports.go` validates package import boundaries:
  - It parses all Go files with AST and forbids imports from:
    - `github.com/fastygo/framework/internal/features`
    - `github.com/fastygo/framework/internal/site/features`
    - `github.com/fastygo/framework/views`
    - `github.com/fastygo/framework/fixtures`
  - If a forbidden import is found, the checker exits with code `1` and prints file + line.
- `.github/workflows/no-root-imports.yml` runs this check in CI:
  - Triggered on `push` to `main` and pull requests.
  - Executes `make ci`, which resolves to:
    - `make lint-ci`
    - `make lint`
    - `go test ./...`
    - `go run ./scripts/check-no-root-imports.go`
- On environments without `make`, run the same check locally with:

```bash
go test ./...
go run ./scripts/check-no-root-imports.go
```

- Docs site checks are covered by the same root targets because the same Makefile and script set is used.

---

## Troubleshooting

- **Language button does nothing**:
  - ensure latest JS is loaded (`Ctrl+F5`)
  - ensure `?lang` query param appears in URL after click
- **Port in use (`bind: ... Only one usage ...`)**:
  - stop old process on same port and rerun
- **Need to inspect generated template output**:
  - run `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`

---

## Notes

This is a baseline implementation. It intentionally avoids adding business complexity in Phase 0 so teams can use this as a fast scaffold and extend with:
- real domains
- repositories / persistence
- auth
- validation layers
- richer eventing and background jobs

