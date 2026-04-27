# examples/landing

Single-page landing site with no CQRS and a hand-typed feature struct directly
inside `cmd/server/main.go`. The page body is composed from
`github.com/fastygo/blocks/marketing`, so this example stays focused on app
wiring, data loading, and brand-specific class hooks.

Use it to learn:

- The shape of a `Feature` (`ID`, `Routes`, `NavItems`).
- How to render a templ component through `web.CachedRender`.
- The minimum viable Tailwind + UI8Kit pipeline (`web/static/css/input.css`).
- How examples consume reusable Blocks while keeping brand CSS local.

## Quick start

```bash
cd examples/landing
bun install
go mod download
bun run vendor:assets
bun run build:css
templ generate ./...
go run ./cmd/server
```

Open <http://127.0.0.1:8080>.

## What it does NOT include

- No locale negotiation. If you need i18n, copy the loader pattern from
  `examples/web/internal/site/i18n`.
- No CQRS dispatcher. If you need command/query handlers, see
  `examples/web/internal/site/welcome/feature.go`.
- No content management. If you serve markdown, look at `examples/docs`
  and `examples/blog`.
