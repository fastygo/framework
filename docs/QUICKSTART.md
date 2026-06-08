# Quickstart

## Prerequisites

- Go `1.25.5` or newer
- Bun `1.3+`

## New apps — use Blank + Templ

For new projects, start from [`github.com/fastygo/blank`](https://github.com/fastygo/blank)
(`framework` + `templ`), not the legacy `examples/` starters.

```bash
git clone <blank-repo-url> my-app
cd my-app

bun install
go mod download
bun run build:css
go tool templ generate ./...
bun run go
```

Open `http://127.0.0.1:8080/` — hero shell with EN/RU switch and dark theme.

See [`docs/EXAMPLES.md`](./EXAMPLES.md) for the active layer model and Templ brick workflow.

---

## Legacy quickstart — `examples/web` (UI8Kit stack)

> **Maintenance only.** `examples/*` still depend on canceled `ui8kit` / `elements` / `blocks`.
> Use this path only when working on legacy examples inside the Framework monorepo.

### 1) Clone and install dependencies

```bash
git clone <your-fork-or-repo-url> fastygo-framework
cd fastygo-framework

bun install
go mod download
```

### 2) Prepare legacy UI8Kit assets

```bash
go mod download github.com/fastygo/ui8kit@v0.2.5
(cd examples/web && bun run vendor:assets)
```

### 3) Run the legacy example

```bash
(cd examples/web && bun run build:css)
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./...
(cd examples/web && go run ./cmd/server)
```

Alternatively, from `examples/web` if `make` is available:

```bash
make dev
```

### 4) Open in your browser

`http://127.0.0.1:8080` — legacy marketing site shell (UI8Kit + Blocks/Elements).

## Optional production build (legacy example)

```bash
(cd examples/web && make build)
```

## Environment

Apps read defaults from `pkg/app/config.go`:

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)
