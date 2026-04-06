# Framework Phase 0 Boilerplate

This repository is a minimal dashboard skeleton for the Universal Framework.

## What this project includes

Phase 0 provides a working baseline with zero domain business logic:

- UI8Kit Shell-based layout with header and sidebar navigation
- Responsive mobile Sheet panel
- Theme toggle (light/dark) with persistence
- Locale switcher for English and Russian using embedded fixtures
- One preconfigured `Welcome` feature with a greeting page
- Server-side rendering with `a-h/templ`
- Feature-based module system (`Feature` + `AppBuilder`)
- CQRS dispatcher with validation and logging pipeline behaviors
- Structured request logging, request IDs, panic recovery middleware
- Tailwind CSS 4 build pipeline and static asset serving

## Stack

- Go `1.25.5`+
- `net/http` (stdlib)
- `log/slog`
- `github.com/a-h/templ`
- `github.com/fastygo/ui8kit v0.2.1`
- `github.com/google/uuid`
- Tailwind CSS 4 (`tailwindcss`, `@tailwindcss/cli`)

## Project structure

Core parts:

- `cmd/server/main.go` is the composition root
- `pkg/app` has config, app builder, and feature interfaces
- `pkg/core/cqrs` has dispatcher, behaviors, and handler interfaces
- `pkg/web` has middleware, templ render helper, and error handling
- `internal/features/welcome` has one demo feature with query/handler
- `views` contains `layout`, `welcome` page, and language partial
- `fixtures` stores embedded `en` / `ru` JSON content
- `static` stores Tailwind entry, UI8Kit CSS, and browser scripts
- `docs/QUICKSTART.md` has a shorter startup guide

## Prerequisites

- Go `1.25.5` or newer
- Node.js `20+`
- Bash (required for the `sync:ui8kit` script in `.sh`)

## Setup

From repository root:

```bash
npm install
go mod download
go mod download github.com/fastygo/ui8kit@v0.2.1
npm run sync:ui8kit
```

## Build and run commands

### Generate templates and styles, then run in development mode

```bash
npm run build:css
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/server
```

If `make` is available:

```bash
make dev
```

### Production build

```bash
make build
./bin/framework
```

### Template generation only

```bash
npm run generate
```

### Run tests

```bash
make test
```

or

```bash
go test ./...
```

### CSS build/watch

```bash
npm run build:css
npm run dev:css
```

Or using Makefile:

```bash
make css-build
make css-dev
```

## View the app

Open:

- `http://127.0.0.1:8080`

You should see:

- Dashboard shell with sidebar and mobile sheet behavior
- Header language switcher
- Header theme switcher
- Welcome page with title, description, and button

## Important environment variables

- `APP_BIND` (default: `127.0.0.1:8080`)
- `APP_STATIC_DIR` (default: `static`)
- `APP_DEFAULT_LOCALE` (default: `en`)
- `APP_AVAILABLE_LOCALES` (default: `en,ru`)
- `APP_DATA_SOURCE` (default: `fixture`)

## Why this framework was created

This project started as a practical bootstrap for teams that need a repeatable dashboard baseline
without waiting on a monolithic framework setup.

Goals at the start:

- Reduce time from `git clone` to first render to a few minutes.
- Keep the runtime stack simple and transparent.
- Provide a standard structure for future features and modules without forcing business rules early.
- Make frontend shell, shell behavior, i18n, and page rendering available from day one.

The result is a Phase 0 skeleton where most teams can replace only feature modules and content
while keeping the same delivery pattern.

## The three pillars

### 1) Deterministic Go core

- `net/http` as the only runtime server layer.
- `log/slog` for structured logging.
- No generated framework bootstrap (except template generation for SSR).
- Predictable request flow through explicit middleware and typed app builder.

### 2) UI shell and design consistency first

- UI8Kit provides a ready-to-use shell, components, icons, and layout behaviors.
- Tailwind CSS 4 is used for deterministic utility-first styling and fast compile-time changes.
- A stable HTML entry (Shell + Sheet + header actions) is guaranteed from day one.

### 3) Clean vertical modularity

- `Feature` modules isolate routes, navigation, and handlers.
- CQRS with pipeline behaviors provides a predictable command/query shape even in simple skeletons.
- Embedded fixtures make locale/content changes easy without introducing extra services in Phase 0.

## Why UI8Kit

UI8Kit was selected as the default UI layer for this phase because:

- It gives a complete shell pattern (`Shell`, `Nav`, header actions) out of the box.
- It reduces custom layout code and CSS drift.
- It ships with ready components (`Title`, `Text`, `Button`, cards, mobile Sheet, icon set).
- It matches Tailwind utilities and supports dark mode and responsive behavior naturally.

For a starter boilerplate this means less UI plumbing and more time focused on business features.

## Distinctive features compared to a generic template

- No `hubcore`/`hubrelay` dependency and no hidden reflection-based DI by default.
- Pure Go feature composition in `main.go`.
- Embedded fixtures (`go:embed`) for i18n and content bootstrap.
- Strong defaults for locale, static path, and bind address through `pkg/app/config.go`.
- Explicit and low-ceremony CQRS usage with logging + validation behaviors.
- One repository command flow (`build`, `dev`, `test`) that stays reproducible.

## Troubleshooting

If `make` is not found, run the explicit command sequence from the “Build and run” section.

If you still see old behaviour after pulling changes, do a hard refresh in the browser
(Ctrl+F5) to clear old cached JS/CSS output.

## Port already in use (address already bound)

If you run:

```bash
go run ./cmd/server
```

and get:

```text
bind: Only one usage of each socket address (protocol/network address/port) is normally permitted.
```

it means another process is already listening on port `8080`.

### Windows

```bat
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

### PowerShell (Windows)

```powershell
Get-NetTCPConnection -LocalPort 8080 -State Listen | Select-Object -ExpandProperty OwningProcess
Stop-Process -Id <PID> -Force
```

### macOS / Linux

```bash
lsof -i :8080
kill -9 <PID>
```

For quick local start with a different port:

```bash
APP_BIND=127.0.0.1:8081 go run ./cmd/server
```
