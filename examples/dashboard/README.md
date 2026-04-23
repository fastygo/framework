# examples/dashboard

A starter for internal tools, CRMs, and admin panels. It demonstrates how to:

- Compose **two features** (`auth`, `dashboard`) inside one composition root.
- Stand up a **typed cookie session** with `pkg/auth.CookieSession[T]`.
- Plug an **auth middleware** in front of every protected route via the
  feature's `Middleware` method.
- Ship a **simple CRUD** (contacts) backed by an in-memory repository so
  you can swap in your own database without touching the HTTP layer.
- Carry a **domain layer** in `internal/domain/*` separate from infra.

## Layout

```
examples/dashboard/
├── cmd/server/main.go               # composition root
├── internal/
│   ├── domain/contact.go            # domain entity
│   └── site/
│       ├── auth/feature.go          # session, login/logout, middleware
│       ├── contacts/repo.go         # in-memory repository
│       ├── dashboard/feature.go     # protected routes
│       └── views/                   # templ pages
├── web/static/                      # Tailwind + UI8Kit assets
└── go.mod
```

## Quick start

```bash
cd examples/dashboard
bun install
go mod download
bun run vendor:assets
bun run build:css
templ generate ./...
SESSION_KEY="$(openssl rand -base64 32)" go run ./cmd/server
```

Open <http://127.0.0.1:8080>. Use any email + password to sign in.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `APP_BIND` | `127.0.0.1:8080` | HTTP listener address |
| `APP_STATIC_DIR` | `web/static` | Directory served under `/static/` |
| `SESSION_KEY` | `demo-session-key-change-me` | HMAC secret for session cookies. **Must be replaced in production.** |
