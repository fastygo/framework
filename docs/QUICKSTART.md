# Quickstart

## Prerequisites

- Go `1.25.5` or newer
- Bun `1.3+`

## 1) Clone and install dependencies

```bash
git clone <your-fork-or-repo-url> fastygo-framework
cd fastygo-framework

bun install
go mod download
```

## 2) Prepare UI8Kit assets

```bash
go mod download github.com/fastygo/ui8kit@v0.2.5
(cd examples/web && bun run vendor:assets)
```

## 3) Run the app in development mode

```bash
(cd examples/web && bun run build:css)
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

Alternatively, if `make` is available:

```bash
make dev
```

## 4) Open in your browser

Go to:

- `http://127.0.0.1:8080`

You should see the dashboard shell with:

- Sidebar navigation from features
- Mobile `Sheet` panel behavior on narrow viewports
- Theme toggle in the header
- Language switcher in the header
- Welcome page with title, description and button

## Optional production build

```bash
make build
./bin/framework
```

## Environment

The app reads these defaults in `pkg/app/config.go`:

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)
