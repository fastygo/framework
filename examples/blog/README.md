# examples/blog

Markdown-driven blog example. Posts live under `content/posts/<slug>.md`,
get embedded into the binary, and are rendered through the framework's
content library.

This example intentionally uses a custom site shell instead of
`ui8kit/layout.Shell`. The shell still keeps the UI8Kit ARIA/theme hooks for
the mobile navigation sheet, burger trigger, and dark mode toggle, and is
checked by `scripts/layout-audit`.

## Layout

```
examples/blog/
├── cmd/server/main.go               # composition root
├── content/
│   ├── embed.go                     # //go:embed posts/*.md
│   └── posts/                       # markdown sources
├── internal/site/
│   ├── blog/feature.go              # routes + post registry
│   └── views/                       # templ + generated _templ.go
├── web/static/                      # CSS / JS / images
└── go.mod                           # depends on github.com/fastygo/framework
```

## Add a post

1. Create `content/posts/<slug>.md`.
2. Append `{Slug, Title, Summary}` to the `Posts` slice in
   `internal/site/blog/feature.go`.
3. Run `templ generate ./... && go build ./...`.

## Validate the custom shell

```bash
bun run layout:audit
```

The audit runs from the Framework root and checks the custom sheet, burger
trigger, close controls, theme toggle, and `<main>` landmark wiring.

## Quick start

```bash
cd examples/blog
bun install
go mod download
bun run vendor:assets
bun run build:css
templ generate ./...
go run ./cmd/server
```

Open <http://127.0.0.1:8080>.
