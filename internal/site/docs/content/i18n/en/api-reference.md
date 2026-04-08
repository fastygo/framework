# API Reference (Go)

This document is a concise but complete API map of the current Phase 0 implementation. It covers every Go source file currently present in the repository and explains how each part contributes to the BFF skeleton.

---

## 1) What BFF means in this repository

### Short version
BFF (Backend for Frontend) in this framework is a thin server-side layer that:
- prepares data for the UI (via CQRS and i18n content),
- renders pages with `templ`,
- acts as a single HTTP entry point (`net/http`),
- owns cross-cutting concerns (request ID, panic recovery, logging, localization, theme state),
- does not contain domain business UI logic itself.

### Why this is useful here
- Fast path to an MVP dashboard without SPA complexity.
- Clear separation: presentation in `internal/site/web/*`, use-cases in `internal/application/*`, and infrastructure adapters in `internal/infra/*`, plus platform core in `pkg/*`.
- Unified entry point for future JSON/SPA/API endpoints (Phase 1+).

### Current request flow
`HTTP -> middleware chain -> feature route -> CQRS query -> i18n -> templ layout/page -> Render`

### Where to find layer boundaries
- `pkg/app`, `pkg/core`, `pkg/web`: framework core and platform API.
- `internal/application/*` and `internal/infra/features/*`: use-case + adapter examples.
- `internal/site/web/views/*`: presentation layer (templ components).

---

## 2) Application vs Infra feature contract

Feature code in this repository is split into two contracts:

### `internal/application/<feature>`
- Contains CQRS use cases and business orchestration for the feature.
- Owns query/command types, handlers, and local DTOs returned to the transport layer.
- Must not depend on `pkg/web`, `internal/site/web/views`, or generated `*_templ.go` files.
- Example:
  - `internal/application/welcome/handler.go`
  - exports `WelcomeQuery`, `WelcomeQueryResult`, `WelcomeQueryHandler`

### `internal/infra/features/<feature>`
- Contains HTTP/templ transport adapters and feature registration.
- Implements `pkg/app.Feature` (`Routes`, `NavItems`, optional `NavProvider`).
- Accepts a `*cqrs.Dispatcher` in constructor and dispatches application queries/commands.
- May import `pkg/web` and `internal/site/web/views` for transport concerns (rendering, localization switches, route handling).
- Example:
  - `internal/infra/features/welcome/module.go`
  - calls `cqrs.DispatchQuery[appwelcome.WelcomeQuery, appwelcome.WelcomeQueryResult](...)`

### Wiring rule (root composition)
- `cmd/server/main.go` owns registration:
  - register application handlers in `cqrs` first (`cqrs.RegisterQuery` / `RegisterCommand`);
  - create feature adapters and pass them into `app.New(...).WithFeature(...).Build()`.
- `Feature` code should not contain direct persistence or presentation logic beyond route handling.

### New feature checklist
1. Add use-case files in `internal/application/<feature>/`.
2. Add transport adapter in `internal/infra/features/<feature>/` implementing `pkg/app.Feature`.
3. Import and register handler and adapter in `cmd/server/main.go`.
4. Add templates and static assets under `internal/site/web/*` as needed.
5. Keep import direction one-way: `main -> infra -> application -> core/platform`.

## 3) File-by-file API map

Below is the API you can use as a quick reference.

---

### `cmd/server/main.go`

`package main`

#### Exported API
- `func main()`
  - Loads configuration with `app.LoadConfig()`.
  - Initializes `cqrs.Dispatcher` with behaviors:
    - `behaviors.Logging`
    - `behaviors.Validation`
  - Registers `welcome.WelcomeQueryHandler`.
  - Builds the app through `app.New(...).WithFeature(...).Build()`.
  - Starts the app with graceful shutdown context (`signal.NotifyContext`).

#### Behavior Notes
- Exits with code `1` on configuration error or fatal server stop.
- `application.Run(ctx)` manages the HTTP server lifecycle.

---

### `pkg/app/config.go`

`package app`

#### Types
- `type Config struct`
  - `AppBind string`
  - `DataSource string`
  - `StaticDir string`
  - `DefaultLocale string`
  - `AvailableLocales []string`

#### Functions
- `func LoadConfig() (Config, error)`
  - Reads env values:
    - `APP_BIND`
    - `APP_DATA_SOURCE`
    - `APP_STATIC_DIR`
    - `APP_DEFAULT_LOCALE`
    - `APP_AVAILABLE_LOCALES`
  - Returns an error if no locales are configured.
- `func getEnv(key string, fallback string) string` (unexported)
- `func parseLocales(raw string) []string` (unexported)

---

### `pkg/app/feature.go`

`package app`

#### Types
- `type NavItem struct { Label, Path, Icon string; Order int }`
- `type Feature interface`
  - `ID() string`
  - `Routes(mux *http.ServeMux)`
  - `NavItems() []NavItem`

#### Notes
- In Phase 0, `Feature` is intentionally minimal and focused on route registration and navigation metadata.

---

### `pkg/app/builder.go`

`package app`

#### Types
- `type App struct`
  - `cfg Config`
  - `mux *http.ServeMux`
  - `features []Feature`
  - `handler http.Handler`
  - `navItems []NavItem`
- `type NavProvider interface`
  - `SetNavItems([]NavItem)`
- `type AppBuilder struct`
  - `cfg Config`
  - `features []Feature`
  - `mux *http.ServeMux`

#### Methods / Functions
- `func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request)`
  - Implements `http.Handler`.
- `func (a *App) Handler() http.Handler`
- `func (a *App) NavItems() []NavItem`
- `func (a *App) Run(ctx context.Context) error`
  - Starts `http.Server` on `a.cfg.AppBind`.
  - Supports graceful shutdown with a 5s timeout.
- `func New(cfg Config) *AppBuilder`
- `func (b *AppBuilder) WithFeature(feature Feature) *AppBuilder`
- `func (b *AppBuilder) Build() *App`
  - Builds middleware chain (`RequestID`, `Recover`, `Logger`).
  - Registers feature routes.
  - Injects shared navigation via `SetNavItems` when feature implements `NavProvider`.
  - Adds static file serving at `/static/`.
- `func collectNavItems(features []Feature) []NavItem` (unexported)
  - Aggregates and stable-sorts nav items.

---

### `pkg/core/entity.go`

`package core`

#### Types / Functions
- `type Entity[ID comparable] struct { ID ID; CreatedAt time.Time; UpdatedAt time.Time }`
- `func NewEntity[ID comparable](id ID) Entity[ID]`
- `func (e *Entity[ID]) Touch()`

---

### `pkg/core/errors.go`

`package core`

#### Types / Constants
- `type ErrorCode string`
  - `ErrorCodeNotFound`, `ErrorCodeConflict`, `ErrorCodeValidation`, `ErrorCodeUnauthorized`, `ErrorCodeForbidden`, `ErrorCodeInternal`
- `type DomainError struct { Code ErrorCode; Message string; Cause error }`

#### Functions / Methods
- `func NewDomainError(code ErrorCode, message string) DomainError`
- `func WrapDomainError(code ErrorCode, message string, cause error) DomainError`
- `func (e DomainError) Error() string`
- `func (e DomainError) StatusCode() int`

---

### `pkg/core/cqrs/command.go`

`package cqrs`

#### Types
- `type Command interface{}`
- `type Query interface{}`
- `type CommandHandler[T any, R any] interface { Handle(context.Context, T) (R, error) }`
- `type QueryHandler[T any, R any] interface { Handle(context.Context, T) (R, error) }`

---

### `pkg/core/cqrs/behavior.go`

`package cqrs`

#### Types
- `type HandlerFunc func(context.Context, any) (any, error)`
- `type PipelineBehavior interface { Handle(ctx context.Context, request any, next HandlerFunc) (any, error) }`
- `type HandlerNotFoundError struct { RequestType string }`
  - `func (e HandlerNotFoundError) Error() string`

---

### `pkg/core/cqrs/dispatcher.go`

`package cqrs`

#### Types
- `type Dispatcher struct`
  - `commandHandlers map[string]HandlerFunc`
  - `queryHandlers map[string]HandlerFunc`
  - `behaviors []PipelineBehavior`

#### Methods / Functions
- `func NewDispatcher(behaviors ...PipelineBehavior) *Dispatcher`
- `func (d *Dispatcher) RegisterCommandHandler(requestType string, handler HandlerFunc)`
- `func (d *Dispatcher) RegisterQueryHandler(requestType string, handler HandlerFunc)`
- `func (d *Dispatcher) Dispatch(ctx context.Context, request any) (any, error)`
  - Resolves request handler from command/query maps and applies behavior pipeline.
  - Returns `core.WrapDomainError(... HandlerNotFoundError...)` when missing.
- `func (d *Dispatcher) execute(ctx context.Context, _ string, handler HandlerFunc, request any) (any, error)` (unexported)
- `func wrapBehavior(behavior PipelineBehavior, next HandlerFunc) HandlerFunc` (unexported)

---

### `pkg/core/cqrs/handler.go`

`package cqrs`

#### Functions
- `func requestKey(request any) string` (unexported)
- `func RegisterCommand[T any, R any](dispatcher *Dispatcher, handler CommandHandler[T, R])`
- `func RegisterQuery[T any, R any](dispatcher *Dispatcher, handler QueryHandler[T, R])`
- `func DispatchCommand[T any, R any](ctx context.Context, dispatcher *Dispatcher, command T) (R, error)`
- `func DispatchQuery[T any, R any](ctx context.Context, dispatcher *Dispatcher, query T) (R, error)`
- `func dispatchTyped(ctx context.Context, dispatcher *Dispatcher, request any, _ string) (any, error)` (unexported)
- `func wrapHandler[T any](fn func(context.Context, T) (any, error)) HandlerFunc` (unexported)

---

### `pkg/core/cqrs/behaviors/logging.go`

`package behaviors`

#### Types
- `type Logging struct { Logger *slog.Logger }`

#### Methods
- `func (l Logging) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error)`
  - Logs CQRS request start/completion/errors.
  - Uses `slog.Default()` if `Logger` is nil.

---

### `pkg/core/cqrs/behaviors/validation.go`

`package behaviors`

#### Types
- `type Validation struct{}`

#### Methods
- `func (Validation) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error)`
  - If request has `Validate() error`, calls it before `next`.
  - On validation failure, returns `core.WrapDomainError(ErrorCodeValidation, ...)`.

---

### `pkg/web/page.go`

`package web`

#### Types
- `type ThemeToggleData struct`
- `type PageData struct`
- `type LanguageToggle struct`

Used for passing view-state into templates.

---

### `pkg/web/render.go`

`package web`

#### Functions
- `func Render(ctx context.Context, w http.ResponseWriter, component templ.Component) error`
  - Renders templ component into a response buffer.
  - Sets `Content-Type: text/html`.
- `func WriteJSON(w http.ResponseWriter, status int, payload any) error`
  - Serializes payload as JSON with the specified status.

---

### `pkg/web/error_handler.go`

`package web`

#### Functions
- `func HandleError(w http.ResponseWriter, err error)`
  - Maps `core.DomainError` to an HTTP status.
  - Logs through `slog.Error`.
  - Sends a generic response via `http.Error`.

---

### `pkg/web/middleware/chain.go`

`package middleware`

#### Types
- `type Middleware func(http.Handler) http.Handler`
- `type Chain []Middleware`

#### Methods
- `func (c Chain) Then(h http.Handler) http.Handler`
  - Applies middleware in reverse order.

---

### `pkg/web/middleware/request_id.go`

`package middleware`

#### Types / constants
- `const RequestIDHeader = "X-Request-ID"`

#### Functions
- `func ContextWithRequestID(ctx context.Context, requestID string) context.Context`
- `func RequestIDFromContext(ctx context.Context) string`
- `func RequestIDMiddleware() Middleware`
  - If the header is missing, generates `uuid.NewString()` and writes it to both context and response.

---

### `pkg/web/middleware/recover.go`

`package middleware`

#### Functions
- `func RecoverMiddleware() Middleware`
  - Recovers panics and returns HTTP 500.

---

### `pkg/web/middleware/logger.go`

`package middleware`

#### Types
- `type statusResponseWriter struct`
  - `statusCode int`
  - `size int`
- `func (rw *statusResponseWriter) WriteHeader(statusCode int)`
- `func (rw *statusResponseWriter) Write(b []byte) (int, error)`

#### Methods
- `func LoggerMiddleware() Middleware`
  - Logs `http.request` and `http.response` with `request_id`, `status`, `duration_ms`, and `size`.

---

### `internal/site/web/i18n/embed.go`

`package i18n`

#### Types
- `type Localized struct { Common CommonFixture; Welcome WelcomeFixture }`
- `type CommonFixture struct`, `NavFixture`, `ThemeFixture`, `LangFixture`, `WelcomeFixture`

#### Functions
- `func Locales() []string`
- `func Decode[T any](locale string, section string) (T, error)`
- `func Load(locale string) (Localized, error)`
- `func normalizeLocale(locale string) string` (unexported)

#### Globals
- `var fixtureFS embed.FS` populated by `//go:embed en/*.json ru/*.json`.

---

### `internal/application/welcome/handler.go`

`package welcome`

#### Types
- `type WelcomeQuery struct { Locale string }`
  - `func (q WelcomeQuery) Validate() error`
- `type WelcomeQueryResult struct { Layout i18n.Localized }`
- `type WelcomeQueryHandler struct{}`
  - `func (h WelcomeQueryHandler) Handle(_ context.Context, query WelcomeQuery) (WelcomeQueryResult, error)`
    - Loads i18n fixtures and returns localized payload.

---

### `internal/infra/features/welcome/module.go`

`package welcome`

#### Types
- `type Module struct`
  - `dispatcher *cqrs.Dispatcher`
  - `navItems []app.NavItem`
  - `defaultLocale string`
  - `availableLocales []string`

#### Methods
- `func New(dispatcher *cqrs.Dispatcher, defaultLocale string, availableLocales []string) *Module`
- `func (m *Module) ID() string`
- `func (m *Module) NavItems() []app.NavItem`
- `func (m *Module) SetNavItems(items []app.NavItem)`
- `func (m *Module) Routes(mux *http.ServeMux)`
- `func (m *Module) handleWelcome(w http.ResponseWriter, r *http.Request)`
  - Resolves locale from `?lang=` query param.
- `func resolveLocale(r *http.Request, defaultLocale string, allowedLocales []string) string` (unexported)
  - Normalizes locale, validates allowed values, and falls back to default.
- `func containsLocale(allowedLocales []string, locale string) bool` (unexported)

---

### `internal/site/web/views/models.go`

`package views`

#### Types
- `type ThemeToggleData struct`
- `type LayoutData struct`
- `type WelcomePageData struct`

---

### `internal/site/web/views/partials/models.go`

`package partials`

#### Types
- `type LanguageToggleData struct`

---

### `internal/site/web/views/layout_templ.go` (generated)

`package views`

#### Functions
- `func Layout(data LayoutData, headExtra templ.Component, body templ.Component) templ.Component`
- `func asShellNavItems(items []app.NavItem) []ui8layout.NavItem`

#### Notes
- Generated by `templ`; this is the runtime template component API.

---

### `internal/site/web/views/welcome_templ.go` (generated)

`package views`

#### Function
- `func WelcomePage(data WelcomePageData) templ.Component`

---

### `internal/site/web/views/nav_templ.go` (generated)

`package views`

#### Functions
- `func layoutHeadExtra(headExtra t.Component) templ.Component`

---

### `internal/site/web/views/partials/language_toggle_templ.go` (generated)

`package partials`

#### Function
- `func LanguageToggle(data LanguageToggleData) templ.Component`

---

## 4) How to use this API map during development

- To add a new feature:
  1. Define query/command and handler in `internal/application/<name>/`.
  2. Register the handler in `main.go` using `cqrs.RegisterQuery` / `cqrs.RegisterCommand`.
  3. Implement `Feature` adapter in `internal/infra/features/<name>/` with `Routes` and `NavItems`.
  4. Add DTOs/i18n resources and templates.
  5. Integrate localization and error handling via `internal/site/web/i18n` + `web.HandleError`.

- For troubleshooting:
  1. Follow flow: `features -> handler -> query -> i18n -> views`.
  2. Middleware chain always starts with request ID for traceability.
  3. Prefer returning business errors as `core.DomainError`.

---

## 5) Extension notes

- In Phase 1, expected next API work includes:
  - health/readiness endpoints,
  - additional middleware (auth, rate limiting, metrics),
  - API envelope/router,
  - persistence and caching abstractions.
- These extensions are not yet reflected in this file because the current phase intentionally remains minimal.
