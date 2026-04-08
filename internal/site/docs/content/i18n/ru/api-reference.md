# API Reference (Go)

Этот документ — короткая, но полная карта API текущей реализации Phase 0. Он охватывает все Go файлы, которые сейчас есть в репозитории, и объясняет, как каждая часть участвует в формировании BFF-скелета.

---

## 1) Что означает BFF в этом репозитории

### Коротко
BFF (Backend for Frontend) в этом фреймворке — это тонкий серверный слой, который:

- подготавливает данные для UI (через CQRS и i18n-контент),
- рендерит страницы с помощью `templ`,
- выступает как единая HTTP-точка входа (`net/http`),
- инкапсулирует сквозные аспекты (request ID, восстановление после panic, логирование, локализация, состояние темы),
- сам по себе не содержит бизнес-логики UI-домена.

### Зачем это нужно здесь

- быстрый путь к MVP dashboard без SPA-сложностей;
- чёткое разделение слоёв: представление в `internal/site/web/*`, use-case-слой в `internal/application/*`, инфраструктурные адаптеры в `internal/infra/*`, плюс платформенный каркас в `pkg/*`;
- единая точка входа для будущего расширения в JSON/SPA/API роуты (Phase 1+).

### Текущий цепочка обработки запроса

`HTTP -> middleware chain -> feature route -> CQRS query -> i18n -> templ layout/page -> Render`

### Где находятся границы слоёв

- `pkg/app`, `pkg/core`, `pkg/web`: ядро фреймворка и platform API.
- `internal/application/*` и `internal/infra/features/*`: примеры use-case + адаптеров.
- `internal/site/web/views/*`: слой представления (templ-компоненты).

---

## 2) Контракт application и infra фич

Код фич в репозитории разделён на два контракта.

### `internal/application/<feature>`

- содержит CQRS use case-ы и оркестрацию бизнес-сценариев фичи;
- владеет типами query/command, handler-ами и локальными DTO, которые возвращаются в транспортный слой;
- не должен зависеть от `pkg/web`, `internal/site/web/views` и сгенерированных `*_templ.go`.
- пример:
  - `internal/application/welcome/handler.go`
  - экспортирует `WelcomeQuery`, `WelcomeQueryResult`, `WelcomeQueryHandler`

### `internal/infra/features/<feature>`

- содержит HTTP/templ transport adapters и регистрацию фичи;
- реализует `pkg/app.Feature` (`Routes`, `NavItems`, опционально `NavProvider`);
- принимает `*cqrs.Dispatcher` в конструкторе и диспатчит query/command на уровень application;
- может импортировать `pkg/web` и `internal/site/web/views` для транспортных задач (рендер, переключатели локали, маршрутизация).
- пример:
  - `internal/infra/features/welcome/module.go`
  - вызывает `cqrs.DispatchQuery[appwelcome.WelcomeQuery, appwelcome.WelcomeQueryResult](...)`

### Правило сборки (root composition)

- `cmd/server/main.go` владеет регистрацией:
  - сначала регистрирует application-хэндлеры в `cqrs` (`cqrs.RegisterQuery` / `RegisterCommand`),
  - затем создаёт feature-адаптеры и передаёт их в `app.New(...).WithFeature(...).Build()`.
- Код `Feature` не должен содержать прямой persistence- или presentation-логики за пределами обработки роутов.

### Чеклист новой фичи

1. Добавить файлы use-case в `internal/application/<feature>/`.
2. Добавить transport adapter в `internal/infra/features/<feature>/`, реализующий `pkg/app.Feature`.
3. Импортировать и зарегистрировать handler и adapter в `cmd/server/main.go`.
4. Добавить шаблоны и статические ассеты в `internal/site/web/*` по мере необходимости.
5. Соблюдать направление зависимостей в одну сторону: `main -> infra -> application -> core/platform`.

## 3) Карта API по файлам

Ниже API-карта для быстрого ориентирования.

---

### `cmd/server/main.go`

`package main`

#### Экспортируемый API

- `func main()`
  - загружает конфигурацию через `app.LoadConfig()`;
  - инициализирует `cqrs.Dispatcher` с поведениями:
    - `behaviors.Logging`
    - `behaviors.Validation`
  - регистрирует `welcome.WelcomeQueryHandler`;
  - собирает приложение через `app.New(...).WithFeature(...).Build()`;
  - стартует приложение с graceful shutdown через `signal.NotifyContext`.

#### Примечания к поведению

- Возвращает код `1` при ошибке конфигурации или фатальной остановке сервера.
- `application.Run(ctx)` управляет жизненным циклом HTTP-сервера.

---

### `pkg/app/config.go`

`package app`

#### Типы

- `type Config struct`
  - `AppBind string`
  - `DataSource string`
  - `StaticDir string`
  - `DefaultLocale string`
  - `AvailableLocales []string`

#### Функции

- `func LoadConfig() (Config, error)`
  - читает env-переменные:
    - `APP_BIND`
    - `APP_DATA_SOURCE`
    - `APP_STATIC_DIR`
    - `APP_DEFAULT_LOCALE`
    - `APP_AVAILABLE_LOCALES`
  - возвращает ошибку, если локали не настроены.
- `func getEnv(key string, fallback string) string` (unexported)
- `func parseLocales(raw string) []string` (unexported)

---

### `pkg/app/feature.go`

`package app`

#### Типы

- `type NavItem struct { Label, Path, Icon string; Order int }`
- `type Feature interface`
  - `ID() string`
  - `Routes(mux *http.ServeMux)`
  - `NavItems() []NavItem`

#### Примечания

- В Phase 0 `Feature` намеренно минимален и сфокусирован на регистрации маршрутов и метаданных навигации.

---

### `pkg/app/builder.go`

`package app`

#### Типы

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

#### Методы и функции

- `func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request)`
  - Реализует `http.Handler`.
- `func (a *App) Handler() http.Handler`
- `func (a *App) NavItems() []NavItem`
- `func (a *App) Run(ctx context.Context) error`
  - Поднимает `http.Server` на `a.cfg.AppBind`.
  - Поддерживает graceful shutdown с таймаутом 5 сек.
- `func New(cfg Config) *AppBuilder`
- `func (b *AppBuilder) WithFeature(feature Feature) *AppBuilder`
- `func (b *AppBuilder) Build() *App`
  - собирает middleware chain (`RequestID`, `Recover`, `Logger`);
  - регистрирует feature routes;
  - inject-ит общую навигацию через `SetNavItems`, когда фича реализует `NavProvider`;
  - добавляет отдачу статических файлов на `/static/`.
- `func collectNavItems(features []Feature) []NavItem` (unexported)
  - агрегирует и стабильно сортирует элементы меню.

---

### `pkg/core/entity.go`

`package core`

#### Типы и функции

- `type Entity[ID comparable] struct { ID ID; CreatedAt time.Time; UpdatedAt time.Time }`
- `func NewEntity[ID comparable](id ID) Entity[ID]`
- `func (e *Entity[ID]) Touch()`

---

### `pkg/core/errors.go`

`package core`

#### Типы и константы

- `type ErrorCode string`
  - `ErrorCodeNotFound`, `ErrorCodeConflict`, `ErrorCodeValidation`, `ErrorCodeUnauthorized`, `ErrorCodeForbidden`, `ErrorCodeInternal`
- `type DomainError struct { Code ErrorCode; Message string; Cause error }`

#### Функции / методы

- `func NewDomainError(code ErrorCode, message string) DomainError`
- `func WrapDomainError(code ErrorCode, message string, cause error) DomainError`
- `func (e DomainError) Error() string`
- `func (e DomainError) StatusCode() int`

---

### `pkg/core/cqrs/command.go`

`package cqrs`

#### Типы

- `type Command interface{}`
- `type Query interface{}`
- `type CommandHandler[T any, R any] interface { Handle(context.Context, T) (R, error) }`
- `type QueryHandler[T any, R any] interface { Handle(context.Context, T) (R, error) }`

---

### `pkg/core/cqrs/behavior.go`

`package cqrs`

#### Типы

- `type HandlerFunc func(context.Context, any) (any, error)`
- `type PipelineBehavior interface { Handle(ctx context.Context, request any, next HandlerFunc) (any, error) }`
- `type HandlerNotFoundError struct { RequestType string }`
  - `func (e HandlerNotFoundError) Error() string`

---

### `pkg/core/cqrs/dispatcher.go`

`package cqrs`

#### Типы

- `type Dispatcher struct`
  - `commandHandlers map[string]HandlerFunc`
  - `queryHandlers map[string]HandlerFunc`
  - `behaviors []PipelineBehavior`

#### Методы и функции

- `func NewDispatcher(behaviors ...PipelineBehavior) *Dispatcher`
- `func (d *Dispatcher) RegisterCommandHandler(requestType string, handler HandlerFunc)`
- `func (d *Dispatcher) RegisterQueryHandler(requestType string, handler HandlerFunc)`
- `func (d *Dispatcher) Dispatch(ctx context.Context, request any) (any, error)`
  - выбирает handler из карт command/query и прогоняет через behavior pipeline;
  - возвращает `core.WrapDomainError(... HandlerNotFoundError...)`, если обработчик не найден.
- `func (d *Dispatcher) execute(ctx context.Context, _ string, handler HandlerFunc, request any) (any, error)` (unexported)
- `func wrapBehavior(behavior PipelineBehavior, next HandlerFunc) HandlerFunc` (unexported)

---

### `pkg/core/cqrs/handler.go`

`package cqrs`

#### Функции

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

#### Типы

- `type Logging struct { Logger *slog.Logger }`

#### Методы

- `func (l Logging) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error)`
  - логирует старт, завершение и ошибки CQRS-запроса;
  - использует `slog.Default()` если `Logger` не задан.

---

### `pkg/core/cqrs/behaviors/validation.go`

`package behaviors`

#### Типы

- `type Validation struct{}`

#### Методы

- `func (Validation) Handle(ctx context.Context, request any, next cqrs.HandlerFunc) (any, error)`
  - если запрос реализует `Validate() error`, вызов выполняется перед `next`;
  - при ошибке валидации возвращается `core.WrapDomainError(ErrorCodeValidation, ...)`.

---

### `pkg/web/page.go`

`package web`

#### Типы

- `type ThemeToggleData struct`
- `type PageData struct`
- `type LanguageToggle struct`

Используются для передачи состояния представления в шаблоны.

---

### `pkg/web/render.go`

`package web`

#### Функции

- `func Render(ctx context.Context, w http.ResponseWriter, component templ.Component) error`
  - рендерит templ-компонент в буфер ответа;
  - устанавливает `Content-Type: text/html`.
- `func WriteJSON(w http.ResponseWriter, status int, payload any) error`
  - сериализует payload в JSON с указанным статусом.

---

### `pkg/web/error_handler.go`

`package web`

#### Функции

- `func HandleError(w http.ResponseWriter, err error)`
  - маппит `core.DomainError` в HTTP-статус;
  - логирует через `slog.Error`;
  - отправляет generic response через `http.Error`.

---

### `pkg/web/middleware/chain.go`

`package middleware`

#### Типы

- `type Middleware func(http.Handler) http.Handler`
- `type Chain []Middleware`

#### Методы

- `func (c Chain) Then(h http.Handler) http.Handler`
  - применяет middleware в обратном порядке.

---

### `pkg/web/middleware/request_id.go`

`package middleware`

#### Типы / константы

- `const RequestIDHeader = "X-Request-ID"`

#### Функции

- `func ContextWithRequestID(ctx context.Context, requestID string) context.Context`
- `func RequestIDFromContext(ctx context.Context) string`
- `func RequestIDMiddleware() Middleware`
  - если header отсутствует, генерирует `uuid.NewString()` и пишет его в контекст и ответ.

---

### `pkg/web/middleware/recover.go`

`package middleware`

#### Функции

- `func RecoverMiddleware() Middleware`
  - перехватывает panic и возвращает HTTP 500.

---

### `pkg/web/middleware/logger.go`

`package middleware`

#### Типы

- `type statusResponseWriter struct`
  - `statusCode int`
  - `size int`
- `func (rw *statusResponseWriter) WriteHeader(statusCode int)`
- `func (rw *statusResponseWriter) Write(b []byte) (int, error)`

#### Методы

- `func LoggerMiddleware() Middleware`
  - логирует `http.request` и `http.response` с `request_id`, `status`, `duration_ms`, `size`.

---

### `internal/site/web/i18n/embed.go`

`package i18n`

#### Типы

- `type Localized struct { Common CommonFixture; Welcome WelcomeFixture }`
- `type CommonFixture struct`, `NavFixture`, `ThemeFixture`, `LangFixture`, `WelcomeFixture`

#### Функции

- `func Locales() []string`
- `func Decode[T any](locale string, section string) (T, error)`
- `func Load(locale string) (Localized, error)`
- `func normalizeLocale(locale string) string` (unexported)

#### Глобальные переменные

- `var fixtureFS embed.FS`, заполненный через `//go:embed en/*.json ru/*.json`.

---

### `internal/application/welcome/handler.go`

`package welcome`

#### Типы

- `type WelcomeQuery struct { Locale string }`
  - `func (q WelcomeQuery) Validate() error`
- `type WelcomeQueryResult struct { Layout i18n.Localized }`
- `type WelcomeQueryHandler struct{}`
  - `func (h WelcomeQueryHandler) Handle(_ context.Context, query WelcomeQuery) (WelcomeQueryResult, error)`
    - загружает i18n fixtures и возвращает локализованный payload.

---

### `internal/infra/features/welcome/module.go`

`package welcome`

#### Типы

- `type Module struct`
  - `dispatcher *cqrs.Dispatcher`
  - `navItems []app.NavItem`
  - `defaultLocale string`
  - `availableLocales []string`

#### Методы

- `func New(dispatcher *cqrs.Dispatcher, defaultLocale string, availableLocales []string) *Module`
- `func (m *Module) ID() string`
- `func (m *Module) NavItems() []app.NavItem`
- `func (m *Module) SetNavItems(items []app.NavItem)`
- `func (m *Module) Routes(mux *http.ServeMux)`
- `func (m *Module) handleWelcome(w http.ResponseWriter, r *http.Request)`
  - определяет locale из query-параметра `?lang=`.
- `func resolveLocale(r *http.Request, defaultLocale string, allowedLocales []string) string` (unexported)
  - нормализует локаль, валидирует разрешённые значения и делает fallback на дефолт.
- `func containsLocale(allowedLocales []string, locale string) bool` (unexported)

---

### `internal/site/web/views/models.go`

`package views`

#### Типы

- `type ThemeToggleData struct`
- `type LayoutData struct`
- `type WelcomePageData struct`

---

### `internal/site/web/views/partials/models.go`

`package partials`

#### Типы

- `type LanguageToggleData struct`

---

### `internal/site/web/views/layout_templ.go` (сгенерировано)

`package views`

#### Функции

- `func Layout(data LayoutData, headExtra templ.Component, body templ.Component) templ.Component`
- `func asShellNavItems(items []app.NavItem) []ui8layout.NavItem`

#### Примечания

- Сгенерировано `templ`; это runtime API шаблона.

---

### `internal/site/web/views/welcome_templ.go` (сгенерировано)

`package views`

#### Функция

- `func WelcomePage(data WelcomePageData) templ.Component`

---

### `internal/site/web/views/nav_templ.go` (сгенерировано)

`package views`

#### Функции

- `func layoutHeadExtra(headExtra templ.Component) templ.Component`

---

### `internal/site/web/views/partials/language_toggle_templ.go` (сгенерировано)

`package partials`

#### Функция

- `func LanguageToggle(data LanguageToggleData) templ.Component`

---

## 4) Как использовать эту карту API во время разработки

- Чтобы добавить новую фичу:
  1. Определите query/command и handler в `internal/application/<name>/`.
  2. Зарегистрируйте handler в `main.go` через `cqrs.RegisterQuery` / `cqrs.RegisterCommand`.
  3. Реализуйте `Feature`-адаптер в `internal/infra/features/<name>/` с `Routes` и `NavItems`.
  4. Добавьте DTO и i18n-ресурсы и шаблоны.
  5. Интегрируйте локализацию и обработку ошибок через `internal/site/web/i18n` + `web.HandleError`.

- Для диагностики и поиска проблемы:
  1. Следуйте потоку: `features -> handler -> query -> i18n -> views`.
  2. Middleware chain всегда начинается с request ID для трассируемости.
  3. Предпочитайте возврат бизнес-ошибок как `core.DomainError`.

---

## 5) Комментарии по расширению

- В Phase 1 ожидаемые следующие API-расширения:
  - health/readiness endpoints,
  - дополнительный middleware (auth, rate limiting, metrics),
  - API envelope/router,
  - абстракции persistence и caching.
- Эти расширения пока не отражены в этом файле, поскольку текущая фаза намеренно минимальна.
