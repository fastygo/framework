# Быстрый старт

Это руководство для примера `examples/docs` — отдельного серверного приложения документации.

## Что нужно установить

- Go `1.25.5` или новее
- Bun `1.3+`

## 1) Клонирование и установка зависимостей

```bash
git clone <URL-вашего-fork-or-repo> fastygo-framework
cd fastygo-framework
cd examples/docs

bun install
go mod download
```

## 2) Подготовка ассетов UI8Kit

```bash
bun run vendor:assets
```

## 3) Запуск приложения в режиме разработки

```bash
bun run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

Если `make` доступен:

```bash
make dev
```

Команда `make dev` собирает шаблоны Templ и CSS, затем запускает локальный сервер.

## 4) Production-сборка

```bash
make build
./bin/docs
```

Без `make`:

```bash
bun run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go build -o bin/docs ./cmd/server
./bin/docs
```

## 5) Откройте страницу в браузере

Перейдите на:

- `http://127.0.0.1:8081/quickstart`

Убедитесь, что отображаются:

- Sidebar через `Shell`
- Кнопка смены темы
- Переключатель языка
- Ссылки на `/quickstart`, `/developer-guide`, `/api-reference`

## 6) Про режим чтения в браузере

У разных браузеров своя логика активации Reader mode:

- Режим чтения анализирует текстовый контент страницы и строит свой «чистый» вид.
- Все markdown-страницы проходят через одинаковый шаблон:
  - `pkg/content-markdown` рендерит HTML,
  - `templ` вставляет результат в `<article class="prose max-w-none">`.
- Если страница состоит в основном из блоков кода (например, шаги в `quickstart`), алгоритм Reader может предложить более короткий вариант или «не предлагать» его.
- Для лучшего попадания в Reader mode лучше держать между примерами кода короткие описательные абзацы.

На текущий момент это поведение обусловлено не сервером, а heuristics браузера.

## 7) CI и проверки линтера

В репозитории CI запускает `make ci` в `.github/workflows/ci.yml`.

`make ci` по умолчанию выполняет:

- `make lint-ci`
- `make lint`
- `go test ./...`
- `go run ./scripts/check-no-root-imports.go`

Без `make` те же проверки можно запустить вручную:

```bash
go test ./...
go run ./scripts/check-no-root-imports.go
```

## Переменные окружения

По умолчанию значение берутся из `pkg/app/config.go`:

- `APP_BIND` (по умолчанию: `127.0.0.1:8081`)
- `APP_STATIC_DIR` (по умолчанию: `web/static`)
- `APP_DEFAULT_LOCALE` (по умолчанию: `en`)
- `APP_AVAILABLE_LOCALES` (по умолчанию: `en,ru`)
- `APP_DATA_SOURCE` (по умолчанию: `fixture`)
