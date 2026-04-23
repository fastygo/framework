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
- `cmd/docs/main.go` – docs composition and run on `8081`
- `pkg/core` – domain primitives (`Entity`, `DomainError`), CQRS
- `pkg/app` – app builder/config/feature contracts
- `pkg/web` – middleware, render and error helpers
- `internal/application/welcome` – use-case/query-handler layer
- `internal/infra/features/welcome` – HTTP/templ adapter
- `internal/application/docs` – docs queries and pre-render cache
- `internal/site/docs/content` – docs markdown sources
- `internal/site/docs/web/views` – docs templates and layout
- `internal/site/docs/web/static` – docs CSS and docs shell script
- `internal/infra/features/docs` – docs routes and rendering module
- `internal/site/web/views` – templ views and partials
- `internal/site/web/i18n` – embedded JSON i18n content
- `internal/site/web/static/css` – Tailwind + UI8Kit CSS pipeline
- `internal/site/web/static/js/app-shell.js` – theme/locale behavior
- `go run github.com/fastygo/ui8kit/scripts/cmd/sync-assets web/static` – vendors UI8Kit CSS, fonts, theme.js, and ui8kit.js into an example app
- `docs/QUICKSTART.md` – quick run instructions

---

## Environment variables

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `internal/site/web/static`)
- Docs site:
  - `APP_BIND` (default: `127.0.0.1:8081` in docs entrypoint)
  - `APP_STATIC_DIR` (default: `internal/site/docs/web/static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)

### Docs site run/build commands

- From the framework repo root: `bun run build:css:docs` (Tailwind for `examples/docs`)
- `make dev-docs` – run docs site in development mode on `8081`
- `make build-docs` – build `./bin/docs`
- `go run ./cmd/docs` – run docs site in place
- `go run ./cmd/server` and `go run ./cmd/docs` for both apps in parallel

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
  - `bun install`
  - `go mod download`
- Vendor UI8Kit assets:
  - `bun run vendor:assets`
- Dev run:
  - `bun run build:css`
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
  - `bun run dev:css`
- Docs CSS watch (from `examples/docs`): `bun run dev:css`

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

