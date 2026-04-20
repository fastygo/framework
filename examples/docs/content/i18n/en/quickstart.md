# Quickstart

## Prerequisites

- Go `1.25.5` or newer
- Node.js `20+`
- Bash (for the UI8Kit CSS sync script)

## 1) Clone and install dependencies

```bash
git clone <your-fork-or-repo-url> fastygo-framework
cd fastygo-framework

npm install
go mod download
```

## 2) Prepare UI8Kit CSS

```bash
go mod download github.com/fastygo/ui8kit@v0.2.5
npm run sync:ui8kit
```

## 3) Run the app in development mode

```bash
npm run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

Alternatively, if `make` is available:

```bash
make dev
```

Run docs application:

```bash
make dev-docs
```

Build docs statically:

```bash
make build-docs
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
- Documentation index and article pages at `/` and `/quickstart`, `/developer-guide`, `/api-reference`

## Optional production build

```bash
make build
./bin/framework
```

To run both app and docs in production build:

```bash
make build-docs
./bin/docs
```

## Environment

The app reads these defaults in `pkg/app/config.go`:

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)

## CI and lint checks

The repository runs the no-root-import policy in CI via `.github/workflows/no-root-imports.yml`.
It executes `make ci` on pushes to `main` and pull requests.

`make ci` -> `make lint-ci` -> `make lint`:

- `go test ./...`
- `go run ./scripts/check-no-root-imports.go`

Without `make`, run the same checks manually:

```bash
go test ./...
go run ./scripts/check-no-root-imports.go
```
