# 12-Factor configuration

Every runtime knob exposed by `pkg/app.Config` is settable through an
environment variable. This page is the single source of truth for
operators: defaults, allowed values, and the corresponding factor.

> Heroku coined the [12 Factor App](https://12factor.net/) doctrine.
> Factors III (config), IX (disposability), XI (logs), XII (admin
> processes) are the ones we touch directly here.

## Application identity & runtime

| Variable                    | Default                              | Factor | Notes |
| --------------------------- | ------------------------------------ | ------ | ----- |
| `APP_BIND`                  | `127.0.0.1:8080`                     | III    | listening address |
| `APP_DATA_SOURCE`           | `fixture`                            | IV     | example DSN, app-defined |
| `APP_STATIC_DIR`            | `internal/site/web/static`           | III    | static files root |
| `APP_DEFAULT_LOCALE`        | `en`                                 | III    | i18n fallback |
| `APP_AVAILABLE_LOCALES`     | `en,ru`                              | III    | comma-separated |

## HTTP server (introduced in v0.1.0 — Phase 1)

| Variable                          | Default | Factor | Notes |
| --------------------------------- | ------- | ------ | ----- |
| `APP_HTTP_READ_TIMEOUT`           | `5s`    | IX     | full request read |
| `APP_HTTP_READ_HEADER_TIMEOUT`    | `2s`    | IX     | Slowloris defence |
| `APP_HTTP_WRITE_TIMEOUT`          | `10s`   | IX     | response write |
| `APP_HTTP_IDLE_TIMEOUT`           | `120s`  | IX     | keep-alive idle |
| `APP_HTTP_MAX_HEADER_BYTES`       | `1MB`   | IX     | header size cap |
| `APP_HTTP_SHUTDOWN_TIMEOUT`       | `5s`    | IX     | graceful drain |

Garbage / negative values fall back to the default. Durations use
Go's `time.ParseDuration` syntax (`500ms`, `2s`, `1m30s`).

## Authentication & session

| Variable             | Default | Factor | Notes |
| -------------------- | ------- | ------ | ----- |
| `OIDC_ISSUER`        | —       | III    | OpenID Connect issuer URL |
| `OIDC_CLIENT_ID`     | —       | III    | OIDC client ID |
| `OIDC_CLIENT_SECRET` | —       | III    | OIDC client secret |
| `OIDC_REDIRECT_URI`  | —       | III    | post-callback redirect |
| `SESSION_KEY`        | —       | III    | HMAC key for cookie sessions |

## Observability (introduced in v0.2.0 — Phase 2)

| Variable                       | Default | Factor | Notes |
| ------------------------------ | ------- | ------ | ----- |
| `LOG_LEVEL`                    | `info`  | XI     | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`                   | `text`  | XI     | `text` or `json` |
| `HEALTH_LIVE_PATH`             | —       | IX     | empty disables liveness probe |
| `HEALTH_READY_PATH`            | —       | IX     | empty disables readiness probe |
| `METRICS_PATH`                 | —       | XI     | empty disables `/metrics` |
| `OTEL_SERVICE_NAME`            | —       | XI     | passed to optional `fastygo/otel` adapter |
| `OTEL_EXPORTER_OTLP_ENDPOINT`  | —       | XI     | passed to optional `fastygo/otel` adapter |

`pkg/app` does **not** install a `slog` handler nor configure an OTel
exporter automatically. The composition root reads these fields and
wires the appropriate adapter. This keeps the framework's `go.mod`
free of tracing SDKs.

## Composition example

A 12-factor-friendly main looks like this:

```go
func main() {
    cfg, err := app.LoadConfig()
    if err != nil { log.Fatal(err) }

    // 1. logger from LOG_LEVEL / LOG_FORMAT
    slog.SetDefault(buildLogger(cfg.LogLevel, cfg.LogFormat))

    // 2. tracer from OTEL_*  (optional, separate dep)
    tracer, shutdown := otel.New(cfg.OTelServiceName, cfg.OTelExporterEndpoint)
    defer shutdown(context.Background())

    a := app.New(cfg).
        WithTracer(tracer).
        WithMetricsEndpoint(cfg.MetricsPath).
        WithHealthEndpoints(cfg.HealthLivePath, cfg.HealthReadyPath).
        Use(myFeature{}).
        Build()

    if err := a.Run(); err != nil { log.Fatal(err) }
}
```

Empty paths (`METRICS_PATH=""`) cleanly disable the corresponding
endpoint — the builder treats empty strings as "feature off".

## Process model

Per Factor IX (Disposability), `App.Run` traps `SIGINT` and `SIGTERM`,
calls `http.Server.Shutdown` with `HTTPShutdownTimeout`, then waits
for `WorkerService.Stop` so background tasks finish their current
iteration before the process exits. Containers can rely on a clean
`exit 0` within `HTTPShutdownTimeout` of `SIGTERM`.
