# Быстрый старт

## Что нужно установить

- Go `1.25.5` или новее
- Node.js `20+`
- Bash (для скрипта синхронизации CSS UI8Kit)

## 1) Клонирование и установка зависимостей

```bash
git clone <URL-вашего-fork-or-repo> fastygo-framework
cd fastygo-framework

npm install
go mod download
```

## 2) Подготовка CSS UI8Kit

```bash
go mod download github.com/fastygo/ui8kit@v0.2.5
npm run sync:ui8kit
```

## 3) Запуск приложения в режиме разработки

```bash
npm run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

Если `make` доступен:

```bash
make dev
```

Запуск docs-сайта:

```bash
make dev-docs
```

Сборка production-версии docs:

```bash
make build-docs
```

## 4) Откройте страницу в браузере

Перейдите на:

- `http://127.0.0.1:8080` (основное приложение)
- `http://127.0.0.1:8081/docs` (документация)

Вы должны увидеть:

- Боковое меню для фичей
- Адаптивное мобильное поведение `Sheet` на узких экранах
- Кнопку переключения темы в хедере
- Кнопку смены языка в хедере
- Welcome-страницу с заголовком, описанием и кнопкой
- Индекс документов и страницы `/`, `/quickstart`, `/developer-guide`, `/api-reference` в docs

## Альтернативная production-сборка

```bash
make build
./bin/framework
```

Сборка и запуск docs production-версии:

```bash
make build-docs
./bin/docs
```

## CI и проверки линтера

CI в GitHub задается через `.github/workflows/no-root-imports.yml`.
Workflow запускает `make ci` для `push` в `main` и для `pull_request`.

`make ci` выполняет цепочку:

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

Приложение читает значения по умолчанию из `pkg/app/config.go`:

- `APP_BIND` (по умолчанию: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (по умолчанию: `internal/site/web/static`)
- `APP_DEFAULT_LOCALE` (по умолчанию: `en`)
- `APP_AVAILABLE_LOCALES` (по умолчанию: `en,ru`)
- `APP_DATA_SOURCE` (по умолчанию: `fixture`)
