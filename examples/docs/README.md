# examples/docs

Markdown documentation site rendered server-side with the
**[fastygo/framework](https://github.com/fastygo/framework)** content
library and **[fastygo/ui8kit](https://github.com/fastygo/ui8kit)** prose
styles.

## How it works

- `content/i18n/<locale>/<slug>.md` holds the markdown sources for each page.
  They are embedded into the binary via `content/embed.go` (`embed.FS`).
- `pkg/content-markdown.NewLibrary` is built once at startup. It reads every
  declared `PageMeta`, renders it through goldmark, and caches the HTML in
  memory (no I/O at request time).
- The docs feature in `internal/site/docs/feature.go` hooks the library into
  the AppBuilder, owns the routes, navigation, and per-locale layout data.
- Templates live in `internal/site/views`. The page partials (header,
  language toggle) live next door under `views/partials`.

To add a new docs page:

1. Drop the markdown file at `content/i18n/<locale>/<slug>.md` (the default
   locale must always be present; other locales are optional).
2. Append `{Slug, Title}` to `Pages` in `internal/site/docs/feature.go`.
3. Add a translated title under `pages` in
   `internal/site/i18n/<locale>/common.json` if you need a localized label.

## Quick start

```bash
cd examples/docs
bun install
go mod download
bun run vendor:assets
bun run build:css
templ generate ./...
go run ./cmd/server
```

Open <http://127.0.0.1:8081>.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `APP_BIND` | `127.0.0.1:8081` | HTTP listener address |
| `APP_STATIC_DIR` | `web/static` | Directory served under `/static/` |
| `APP_DEFAULT_LOCALE` | `en` | Default locale |
| `APP_AVAILABLE_LOCALES` | `en,ru` | Comma-separated locale list |
