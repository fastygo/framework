# Руководство по разработке FastyGO Framework (Phase 0)

Этот документ описывает текущую реализованную архитектуру с акцентом на быстрое знакомство с кодовой базой.

## Что это за репозиторий

Проект представляет собой **каркас dashboard для Phase 0** без доменной логики, в котором уже собрана рабочая инфраструктура:

- оболочка интерфейса на UI8Kit (`Shell`, боковая панель, мобильный sheet)
- SSR через `a-h/templ`
- переключатель темы и локали
- CQRS-подход для обработки запросов
- типобезопасная композиция фич
- встроенные JSON-fixtures для i18n (`en`, `ru`)

Цель Phase 0 — дать быстрый и стабильный базовый старт, который удобно копировать в новые проекты.

---

## Реализованная архитектура

### 1) Composition Root

`cmd/server/main.go` выполняет роль composition root:

- загружает конфигурацию из переменных окружения
- создает `cqrs.Dispatcher`
- регистрирует pipeline behaviors:
  - поведение валидации
  - поведение логирования
- регистрирует обработчики и фичи
- собирает приложение и запускает HTTP-сервер с graceful shutdown

Так вся инициализация централизована в одном месте.

### 2) Ядро приложения

`pkg/app` предоставляет базовые абстракции приложения:

- `config.go`
  - читает runtime-настройки (`APP_BIND`, `APP_STATIC_DIR`, `APP_DEFAULT_LOCALE`, `APP_AVAILABLE_LOCALES`, `APP_DATA_SOURCE`)
  - валидирует список локалей
- `feature.go`
  - интерфейс `Feature`:
    - `ID() string`
    - `Routes(*http.ServeMux)`
    - `NavItems() []NavItem`
  - модель `NavItem`: `Label`, `Path`, `Icon`, `Order`
- `builder.go`
  - API `AppBuilder`:
    - `New(cfg).WithFeature(feature).Build().Run(ctx)`
  - собирает nav-items от всех фич и сортирует их
  - строит цепочку middleware и обработчик статических файлов (`/static/...`)

### 3) Веб-платформа

`pkg/web` содержит общие HTTP/templ компоненты:

- `middleware/chain.go` — utility для композиции middleware
- `middleware/request_id.go` — назначение request ID и перенос его в контекст/headers
- `middleware/recover.go` — перехват panic
- `middleware/logger.go` — структурированное логирование запросов, включая метаданные ответа (`status`, `duration_ms`, `size`)
- `render.go` — рендер `templ` в HTTP-ответ
- `error_handler.go` — маппинг `DomainError` в HTTP-ответ
- `page.go` — общие DTO страниц для views

### 4) CQRS-ядро (минимальное)

Реализовано в `pkg/core/cqrs`:

- `command.go` — marker-интерфейсы и handler-интерфейсы:
  - `Command`, `Query`, `CommandHandler`, `QueryHandler`
- `behavior.go` — `PipelineBehavior`
- `dispatcher.go` — регистрация и запуск обработчиков
- `behaviors/validation.go` — опциональный hook `Validate() error`
- `behaviors/logging.go` — логирование жизненного цикла запросов

Ключевые точки:

- `cqrs.RegisterQuery(...)` — регистрация типизированного query handler
- `cqrs.DispatchQuery(ctx, dispatcher, query)` — выполнение pipeline и handler

Реализация намеренно компактная, но полностью рабочая.

### 5) Слой представления (templ)

`internal/site/web/views/` — SSR шаблоны:

- `layout.templ`
  - оборачивает страницы в `ui8layout.Shell`
  - рендерит header actions и навигацию
- `welcome.templ`
  - приветственная страница с использованием `ui.Title`, `ui.Text`, `ui.Button`
- `nav.templ`
  - подключает `app-shell.js` в head через shell
- `models.go`
  - строго типизированные DTO для шаблонов
- `partials/language_toggle.templ`
  - атрибуты переключателя локали для браузерного скрипта



## Переключатель локали и стратегия

В текущей архитектуре выбор локали централизован в `app.WithLocales(...)`, поэтому
фичам больше не нужно знать `defaultLocale`, `available` и отдельный `negotiator`.

### 1) Настройка в `cmd/server/main.go`

Добавьте локализацию в composition root:

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

### 2) Доступные стратегии

1. `QueryStrategy` — переключение по query (`?lang=en`).
2. `PathPrefixStrategy` — переключение через сегмент URL (`/en/...`, `/ru/...`).
3. `CookieOptions` — сохранение выбора в cookie.

### 3) Получение локали в обработчике

В обработчике используйте:

```go
loc := locale.From(r.Context())
```

### 4) Рендер переключателя через контекст

```go
language := view.BuildLanguageToggleFromContext(
    r.Context(),
    view.WithLocaleLabels(bundle.Common.Language.LocaleLabels),
)
```

### 5) Шаблон переключателя и SPA

Базовый fallback (без JS):

```templ
<a href={ t.URL(data.NextHref) }>
```

Для SPA-режима включите `SPA: true` в `app.WithLocales`:

```go
builder := app.New(cfg).WithLocales(app.LocalesConfig{
    // ...
    SPA: true,
})
```

Тогда переключатель рендерит `data-ui8kit-spa-lang="1"` и обновляет страницу через
`fetch` с заменой `data-spa-target` (по умолчанию `main`), сохраняя SEO-friendly
путь через обычную ссылку.

### 6) I18N / слой контента

`internal/site/web/i18n/` хранит embedded контент:

- `en/common.json`
- `en/welcome.json`
- `ru/common.json`
- `ru/welcome.json`

`internal/site/web/i18n/embed.go`:

- использует `go:embed` для JSON файлов
- раскладывает контент по locale и секциям
- предоставляет:
  - `Load(locale)` — единый payload страницы
  - `Locales()` — список доступных локалей

Сейчас данные бизнес-уровня в Phase 0 представлены статичными fixtures.

### 7) Frontend-слой поведения

`internal/site/web/static/js/app-shell.js`:

- хранит выбранную тему (`dark` / `light`) в `localStorage`
- хранит выбор локали в `localStorage`
- корректирует редирект при загрузке под браузерную/сохранённую/дефолтную локаль
- обрабатывает клик по переключателю локали

`internal/site/web/static/css/input.css` импортирует Tailwind и UI8Kit стили.

---

## Реализованная бизнес-логика

В Phase 0 реализована одна фича и один публичный сценарий.

### Welcome Feature (`internal/infra/features/welcome`, `internal/application/welcome`)

- `WelcomeQuery` + `WelcomeQueryHandler` (`internal/application/welcome/handler.go`)
  - загружает данные из fixtures
  - валидирует локаль
- `welcome.Module` (`internal/infra/features/welcome/module.go`)
  - регистрирует маршрут `/`
  - задаёт один nav-item `Welcome`
  - определяет effective locale из query string или дефолта
  - переводит `locale` в `WelcomePageData` и рендерит `views.Layout + views.WelcomePage`
- На этом этапе нет внешних интеграций, БД или auth-логики

Текущее бизнес-поведение:

- рендер welcome-страницы по fixture-данным
- смена языка через UI и параметр `?lang=...` с persistence
- persistence выбранной темы
- сборка навигации на основе метаданных фич

---

## Порядок обработки запроса и рендера

1. HTTP-запрос поступает в цепочку `AppBuilder`
2. Запускается middleware-цепочка:
   - request id
   - recover
   - logger
3. Handler `welcome.Module` разрешает локаль и диспатчит `WelcomeQuery`
4. `Dispatcher` выполняет behaviors и query handler
5. Загружаются fixture-данные для выбранной локали
6. Данные передаются в `views.Layout + views.WelcomePage`
7. `templ` рендерит HTML-ответ

---

## Runtime-последовательность (полезно для отладки)

- `GET /` с параметром `lang` → используется эта локаль
- `GET /` без `lang` → используется `APP_DEFAULT_LOCALE`
- клик по переключателю локали → обновляет URL через `?lang=<next>` (или убирает параметр для дефолтной локали) и перерисовывает страницу
- клик по теме → обновляет класс `html.dark` и сохраняет выбор

---

## Карта директорий

- `cmd/server/main.go` — сборка и запуск приложения
- `pkg/core` — доменные примитивы (`Entity`, `DomainError`), CQRS
- `pkg/app` — builder приложения, конфиг и контракты фич
- `pkg/web` — middleware, render и обработчики ошибок
- `internal/application/welcome` — уровень use-case / query-handler
- `internal/infra/features/welcome` — HTTP/templ адаптер фичи
- `internal/site/web/views` — шаблоны и partials
- `internal/site/web/i18n` — embedded JSON i18n
- `internal/site/web/static/css` — pipeline Tailwind + UI8Kit
- `internal/site/web/static/js/app-shell.js` — поведение темы и локали
- `scripts/sync-ui8kit-css.sh` — синхронизация UI8Kit CSS из sources
- `docs/QUICKSTART.md` — быстрый старт

---

## Переменные окружения

- `APP_BIND` (по умолчанию: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (по умолчанию: `internal/site/web/static`)
- `APP_DEFAULT_LOCALE` (по умолчанию: `en`)
- `APP_AVAILABLE_LOCALES` (по умолчанию: `en,ru`)
- `APP_DATA_SOURCE` (по умолчанию: `fixture`)

---

## Добавление новой фичи (рекомендуемый способ)

1. Создайте `internal/application/<name>/` и `internal/infra/features/<name>/`
2. Добавьте:
   - модуль фичи, реализующий `pkg/app.Feature`
   - query/handler пару, если нужен CQRS-поток
3. Зарегистрируйте фичу в `cmd/server/main.go`:
   - `WithFeature(newFeature)`
4. Добавьте шаблоны в `internal/site/web/views/` и DTO моделей
5. Добавьте fixtures или слой источника данных

Так как `AppBuilder` автоматически композитует `NavItems`, каждая фича вносит свои пункты меню.

---

## Команды

- Установка зависимостей:
  - `npm install`
  - `go mod download`
- Синхронизация UI8Kit:
  - `npm run sync:ui8kit`
- Запуск разработки:
  - `npm run build:css`
  - `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`
  - `go run ./cmd/server`
  - или `make dev`, если он доступен
- Сборка:
  - `make build` и запуск `./bin/framework`
- Тесты:
  - `go test ./...`
- Линтеры и архитектурные проверки:
  - `make lint` (тесты + проверка импортов)
  - `make ci` (то же, что выполняет CI)
  - `make lint-ci` (псевдоним для `make ci`)
  - без `make`:
    - `go test ./...`
    - `go run ./scripts/check-no-root-imports.go`
- Наблюдение за CSS:
  - `npm run dev:css`

## CI и проверки линтера

- `scripts/check-no-root-imports.go` — главный линтер архитектурных ограничений импорта:
  - парсит Go-код через AST
  - запрещает импорты из:
    - `github.com/fastygo/framework/internal/features`
    - `github.com/fastygo/framework/internal/site/features`
    - `github.com/fastygo/framework/views`
    - `github.com/fastygo/framework/fixtures`
  - при нарушении печатает имя файла и строку и возвращает ошибку `1`
- GitHub Actions workflow находится в `.github/workflows/no-root-imports.yml`:
  - триггеры: `push` в `main` и `pull_request`
  - в шаге CI выполняет `make ci`:
    - `make lint`
    - `go test ./...`
    - `go run ./scripts/check-no-root-imports.go`

Рекомендуемый локальный запуск без `make`:

```bash
go test ./...
go run ./scripts/check-no-root-imports.go
```

Для сайта документации применяются те же цели и проверки через единый `Makefile`.

---

## Troubleshooting

- **Кнопка смены языка не работает**:
  - убедитесь, что загружен актуальный JS (`Ctrl+F5`)
  - проверьте, что после клика в URL появился `?lang`
- **Ошибка bind: Only one usage of each socket address**:
  - завершите процесс на том же порту и перезапустите
- **Нужно посмотреть сгенеренный шаблонный код**:
  - выполните `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`

---

## Notes

Это базовая реализация для старта. Она сознательно не перегружена бизнес-сложностью, чтобы команды могли быстро начать scaffold и расширять его:

- реальные доменные модели
- репозитории и persistence
- аутентификация
- слои валидации
- более богатые события и background jobs
