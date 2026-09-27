# Agent notes

The root module is the process: `pkg/app` (`Handler`, `Run`), `pkg/core`, `pkg/auth`, HTTP middleware, security, health, and cache.

Templ rendering, markdown, fonts, site-shell view data, and mail are separate modules under `pkg/`. Do not add their dependencies back to the root `go.mod`.

## Rules

1. `Handler()` returns the HTTP handler. `Run(ctx)` owns the process, workers, and graceful shutdown. Do not rename them after a deployment target.
2. Do not import product schemas, FormSet, Codex, or Panel.
3. `pkg/observe` is the tracing interface. Do not add an OpenTelemetry SDK to this module.
4. Comments and documentation are written in English.
5. Consumers pin a published tag. No local `replace` outside this repo's nested modules.

Run before completion:

```text
go test ./...
go vet ./...
gofmt -w .
```
