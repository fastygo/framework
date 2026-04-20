# examples/web

A small marketing-style website that demonstrates how to compose
**[fastygo/framework](https://github.com/fastygo/framework)** features with
**[fastygo/ui8kit](https://github.com/fastygo/ui8kit)** primitives.

It ships:

- A welcome page with localized hero, feature cards, theme toggle, and
  language toggle.
- An optional `/cab/` cabinet behind an OpenID Connect SSO flow (powered
  by `pkg/auth`).
- Tailwind CSS 4 build pipeline plus the synchronized UI8Kit CSS bundles.

## Layout

```
examples/web/
├── cmd/server/main.go           # composition root
├── internal/
│   └── site/
│       ├── i18n/                # embedded JSON locale bundles
│       ├── views/               # templ templates + generated *_templ.go
│       ├── welcome/             # welcome feature (handler + query)
│       └── cab/                 # OIDC cabinet feature
├── web/static/                  # CSS / JS / images (synced with UI8Kit)
├── go.mod                       # depends on github.com/fastygo/framework
├── package.json                 # npm scripts for tailwind + ui8kit sync
└── Dockerfile
```

The example is a **standalone Go module**. While it lives inside the
framework monorepo, a top-level `go.work` makes the local framework module
available without publishing tags. To split it out into its own repository
just copy the directory and remove `go.work`.

## Quick start

```bash
cd examples/web
npm install
go mod download
npm run sync:ui8kit                 # writes web/static/css/ui8kit/*.css
npm run build:css                   # tailwind build
templ generate ./...                # render templ files
go run ./cmd/server
```

Open <http://127.0.0.1:8080>.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `APP_BIND` | `127.0.0.1:8080` | HTTP listener address |
| `APP_STATIC_DIR` | `web/static` | Directory served under `/static/` |
| `APP_DEFAULT_LOCALE` | `en` | Default locale |
| `APP_AVAILABLE_LOCALES` | `en,ru` | Comma-separated locale list |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI` | — | Enable `/cab/` SSO when all four are set |
| `SESSION_KEY` | — | HMAC secret for cookie sessions |

## What to copy when starting from this example

1. `internal/site/welcome` and `internal/site/views` — replace with your
   own feature(s) and templates.
2. `internal/site/i18n` — keep the loader, replace the JSON bundles.
3. `web/static/css/web-components.css` — your site-specific CSS overlays.
4. `cmd/server/main.go` — adjust `WithFeature(...)` calls to enable only
   the modules you need.
