# Quickstart

This guide is for the `examples/docs` app: a dedicated server-side documentation example.

## Prerequisites

- Go `1.25.5` or newer
- Node.js `20+`
- Bash (for the UI8Kit CSS sync script)

## 1) Clone and install dependencies

```bash
git clone <your-fork-or-repo-url> fastygo-framework
cd fastygo-framework
cd examples/docs

npm install
go mod download
```

## 2) Sync UI8Kit CSS

```bash
npm run sync:ui8kit
```

## 3) Run in development mode

```bash
npm run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

If `make` is available:

```bash
make dev
```

`make dev` generates templates, builds CSS, and starts the local server.

## 4) Production build

```bash
make build
./bin/docs
```

Without `make`:

```bash
npm run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go build -o bin/docs ./cmd/server
./bin/docs
```

## 5) Open in browser

Visit:

- `http://127.0.0.1:8081/quickstart`

You should see:

- Sidebar via `Shell`
- Theme toggle
- Language switcher
- Links to `/quickstart`, `/developer-guide`, `/api-reference`

## 6) Why Reader mode is inconsistent across pages

Browser reader mode uses heuristics that prefer long narrative text.

- All markdown pages are rendered through the same pipeline in this example:
  - `pkg/content-markdown` renders markdown to HTML,
  - `templ` places it into `<article class="prose max-w-none">`.
- Pages with a lot of command snippets and short explanatory text (for example `quickstart`) may not be ideal for Reader mode.
- Adding short explanatory paragraphs between code blocks usually improves how reader mode detects the page.

This behavior is browser heuristics, not an issue in this docs app.

## 7) CI and lint checks

CI runs `make ci` from `.github/workflows/ci.yml`.

`make ci` executes:

- `make lint-ci`
- `make lint`
- `go test ./...`
- `go run ./scripts/check-no-root-imports.go`

Without `make`, run manually:

```bash
go test ./...
go run ./scripts/check-no-root-imports.go
```

## Environment

Defaults are configured in `pkg/app/config.go`:

- `APP_BIND` (default: `127.0.0.1:8081`)
- `APP_STATIC_DIR` (default: `web/static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)
