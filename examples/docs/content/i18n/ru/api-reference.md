# API Reference (Go)

Этот документ отражает API именно для `examples/docs` и связан с текущей реализацией docs-сайта.

## 1) Что здесь является BFF-слоем

`examples/docs` — это сервер, который:

- рендерит markdown-содержимое в HTML на сервере,
- возвращает полностью SSR-страницы,
- инкапсулирует infra/кросс-аспекты: middleware, локали, темы, ошибки,
- делегирует бизнес-логику на библиотечные уровни `pkg/*` и `pkg/content-markdown`.

## 2) Корневая конфигурация: `cmd/server/main.go`

`main` выполняет:

- `cfg := app.LoadConfig()`
- инициализацию `content.NewLibrary(...)` из `pkg/content-markdown`
- `application := app.New(cfg)`
- `WithSecurity(security.LoadConfig())`
- `WithLocales(app.LocalesConfig{... QueryStrategy ... Cookie ...})`
- `WithFeature(docsfeature.New(library))`
- `Build()` и `Run(ctx)`

Ключевой эффект: все страницы docs становятся доступными через единый HTTP entrypoint.

## 3) API пакета `pkg/content-markdown`

Файл: `pkg/content-markdown/content.go`

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

`Page` возвращает уже отрендеренный HTML, полученный из markdown.

## 4) API фичи docs: `internal/site/docs/feature.go`

- `var Pages = []content.PageMeta{{Slug: "quickstart", ...}, ...}`
- `type Feature struct { library *content.Library }`
- `func New(library *content.Library) *Feature`
- `func (f *Feature) ID() string`
- `func (f *Feature) NavItems() []web.NavItem`
- `func (f *Feature) Routes(mux *http.ServeMux)`
- `func (f *Feature) renderPage(...)`

`Routes` мапит роуты на `Docs`-страницы и индекс.

## 5) API для view layer

### `internal/site/views/models.go`

- `type DocsPageData struct { Title string; HTMLContent string }`
- `type DocsIndexData struct { Title string }`

### `internal/site/views/layout.templ`

- `templ Layout(data LayoutData)` — главный шаблон с `ui8layout.Shell`.

### `internal/site/views/page.templ`

- `templ DocsPage(data DocsPageData)` — рендерит HTML документа (`@t.Raw(data.HTMLContent)`) внутри `<article class="prose max-w-none">`.

### `internal/site/views/index.templ`

- `templ DocsIndex(data DocsIndexData)` — стартовая docs-страница.

### `internal/site/views/partials/language_toggle.templ`

- рендер локализующего переключателя.

## 6) API локализации: `internal/site/i18n`

- `internal/site/i18n/embed.go` и `internal/site/i18n/{en,ru}/common.json`
- Экспорт:
  - `func Locales() []string`
  - `func Decode[T any](locale string, section string) (T, error)`
  - `func Load(locale string) (Localized, error)`

## 7) Базовая платформа framework, используемая тут

Эти пакеты из репозитория участвуют в API приложения как зависимости:

- `pkg/app`
  - `app.LoadConfig`, `app.New`, `app.LocalesConfig`, `app.WithLocales`, `app.WithFeature`, `app.App.Run`
  - `pkg/app.Feature`, `NavItem`, `AppBuilder`, middleware chain.
- `pkg/web`
  - `web.CachedRender`, `web.HandleError`
- `pkg/web/middleware` (`request_id`, `recover`, `logger`)
- `pkg/content-markdown` (см. выше)
- `pkg/security`

## 8) Расширение API docs-примера

Чтобы добавить новую публичную страницу:

1. Добавьте md-файл в `examples/docs/content/i18n/<locale>/<slug>.md`.
2. Добавьте matching `PageMeta` в `internal/site/docs/feature.go`.
3. При необходимости расширьте `internal/site/i18n`.
4. Перегенерируйте шаблоны при необходимости и перезапустите сервер.

## 9) Поток вызовов (runtime)

- `main` -> `app.New` -> `docsfeature.New` -> `app.Build()`.
- `HTTP request` -> middleware stack -> `docsfeature` handler.
- `content.Library.Page(locale, slug)` -> `DocsPage` -> `templ render`.
- Ответ отдаётся через `http.ResponseWriter` в стиле SSR.

## Примечание по reader mode

В проекте нет отдельного серверного режима читателя.
Если браузерный Reader mode появляется не на всех страницах, это связано с эвристикой чтения браузера: контент лучше анализируется, когда есть связанный explanatory text, а не только блоки кода.
