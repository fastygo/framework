# Руководство по документационному примеру FastyGO

Этот документ описывает `examples/docs` из текущего состояния репозитория и отражает реальную структуру этого примера.

## Что это за пример

`examples/docs` — серверная документационная страница на `net/http` + `templ`, где содержимое берётся из markdown-файлов с последующим рендером в HTML.

В отличие от общего каркаса framework, в этом примере используется только docs-фича и локали для интерфейса.

## Архитектура

### 1) Точка входа приложения — `cmd/server/main.go`

`main.go` делает три вещи:

1. Загружает конфигурацию через `app.LoadConfig()`.
2. Инициализирует библиотеку контента `pkg/content-markdown`:
   - читает файлы из `embed.FS` (`examples/docs/content/embed.go`),
   - использует `docsfeature.Pages` как карту доступных slug,
   - создаёт локализованную библиотеку `library`.
3. Строит приложение через `app.New(cfg)`:
   - включает `security.LoadConfig()`
   - включает `WithLocales(...)` с Query-strategy (`lang`) и cookie-памятью
   - регистрирует фичу `docsfeature.New(library)`
   - `Build()` и `Run(ctx)`.

### 2) Контентный слой — `pkg/content-markdown`

Пакет `pkg/content-markdown` делает следующее:

- проходит по описанию `PageMeta` (`Slug`, `Title`),
- читает markdown из `embed.FS`,
- рендерит его через `goldmark` в HTML,
- кэширует результат в `Library`.

Кэшированные страницы затем читаются вызовом `Page(loc, slug)`.

### 3) Docs feature — `internal/site/docs/feature.go`

`docsfeature` отвечает за маршрутизацию страниц документации:

- хранит `docs.Pages` (список публичных slug: `quickstart`, `developer-guide`, `api-reference`),
- реализует `Routes` для:
  - `GET /`
  - `GET /quickstart`
  - `GET /developer-guide`
  - `GET /api-reference`,
- формирует заголовки/`NavItem` для Shell,
- рендерит страницу через `views.DocsPage(...)`.

### 4) Представление — `internal/site/views/*`

- `layout.templ` — общий каркас (Shell, header, shell actions)
- `page.templ` — рендерит HTML документа (`@t.Raw(data.HTMLContent)`)
- `index.templ` — стартовая страница docs
- `partials/language_toggle.templ` — переключатель языка

Контейнер для Markdown — `<article class="prose max-w-none">`, поэтому визуально все страницы получают «reader-like» типографику.

### 5) Локали — `internal/site/i18n`

- `common.json` на `en` и `ru` содержит текст для shell и shared UI.
- Контентная локаль и i18n JSON подключаются через `go:embed` (`internal/site/i18n/embed.go`) и `app.LocalesConfig`.

## Как работает запрос

1. Запрос приходит в `AppBuilder` и проходит цепочку middleware (`request id`, `recover`, `logger`).
2. `docsfeature` выбирает нужный slug по пути.
3. `contentMarkdown.Library.Page(locale, slug)` возвращает заранее отрендеренный HTML.
4. `templ` собирает страницу и отдаёт `html` в ответ.

## Что важно про Reader mode

Причина, почему для некоторых страниц (чаще всего `developer-guide`, `api-reference`) режим чтения предлагается активнее:

- браузерный reader ориентирован на «читаемый текст»,
- страницы с большим объёмом текста легче классифицируются как article,
- у pages с большим количеством блоков кода, где мало развернутых абзацев, поведение может отличаться.

Это не ошибка рендера, а эвристика браузера.

## Добавление новой docs-страницы

1. Добавьте markdown в `examples/docs/content/i18n/<locale>/<slug>.md`.
2. Добавьте `PageMeta` в `examples/docs/internal/site/docs/feature.go`.
3. При необходимости добавьте заголовок/переводы в `internal/site/i18n`.
4. Перезапустите сервер (`make dev` или `go run ./cmd/server`).

## Карта файлов (для этого примера)

- `cmd/server/main.go` — bootstrap приложения и запуск.
- `internal/site/docs/feature.go` — docs feature и маршруты.
- `internal/site/views/{layout.templ,page.templ,index.templ}` — шаблоны.
- `internal/site/views/partials/language_toggle.templ` — переключатель локали.
- `internal/site/i18n/{en,ru}/common.json` — локальные строки.
- `internal/site/i18n/embed.go` — загрузка локалей и helper-ы.
- `examples/docs/content/*/*.md` — markdown источники.
- `examples/docs/content/embed.go` — `go:embed` для markdown.

## Переменные окружения

- `APP_BIND` (по умолчанию: `127.0.0.1:8081`)
- `APP_STATIC_DIR` (по умолчанию: `web/static`)
- `APP_DEFAULT_LOCALE` (по умолчанию: `en`)
- `APP_AVAILABLE_LOCALES` (по умолчанию: `en,ru`)
- `APP_DATA_SOURCE` (по умолчанию: `fixture`)

## Troubleshooting

- **404 после добавления новой страницы**: проверьте `docs.Pages` и наличие `slug` в markdown.
- **Reader mode не появляется**: добавьте больше пояснительного текста между блоками кода или проверьте на странице меньше технических блоков.
- **Ошибки сборки шаблонов**: запустите `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`.
