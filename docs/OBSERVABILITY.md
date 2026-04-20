# Observability

`fastygo/framework` ships a deliberately small, dependency-free
observability surface. The goal is to give every application the
*hooks* needed for production-grade logs, metrics, traces, and
health checks while keeping the framework's `go.mod` pristine.

The four pillars covered by `v0.2.0`:

| Pillar  | Package                                | Brings any deps? |
| ------- | -------------------------------------- | ---------------- |
| Health  | `pkg/web/health` + `AppBuilder`        | No               |
| Logs    | `pkg/web/middleware/logger.go`         | No (`log/slog`)  |
| Metrics | `pkg/web/metrics` + `MetricsMiddleware`| No               |
| Traces  | `pkg/observability` (interface only)   | No               |

Real OpenTelemetry integration ships separately as
[`github.com/fastygo/otel`](https://github.com/fastygo/otel) (planned)
so applications opt into the OTel SDK explicitly without polluting
the framework's dependency graph.

## 1. Health probes

`AppBuilder` mounts liveness/readiness handlers in front of the
security chain so orchestrators (Kubernetes, Nomad, ECS) get reliable
probes that bypass rate limiting and anti-bot challenges.

```go
app.New(cfg).
    WithHealthEndpoints("/healthz", "/readyz").
    AddHealthChecker(health.CheckerFunc("db", pingDB)).
    Use(myFeature{}). // features implementing HealthChecker auto-register
    Build()
```

- `/healthz` — always 200 once the server is listening.
- `/readyz` — aggregates every registered `Checker` in parallel with a
  per-check timeout (default 2s). 503 + JSON body when any check fails.

A feature contributes a check by implementing the `HealthChecker`
interface; explicit checks (registered via `AddHealthChecker`) are
appended to the list.

## 2. Structured logging

`LoggerMiddleware` emits two `slog` events per request:

- `http.request`  — `request_id`, `method`, `path` (+ optional `trace_id`, `span_id`)
- `http.response` — `request_id`, `status`, `duration_ms`, `size` (+ correlation)

The middleware uses `slog.LogAttrs`, so the hot path stays allocation-flat.
Configure the global handler via `slog.SetDefault`:

```go
h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
slog.SetDefault(slog.New(h))
```

Audit events from `pkg/auth.CookieSession` use the same handler under
the message `auth.audit`. Filter on `level >= WARN` to surface
`session_tampered` and `session_issue_failed` to your SIEM.

## 3. Metrics

`pkg/web/metrics` is a from-scratch implementation of the Prometheus
[text exposition format v0.0.4](https://prometheus.io/docs/instrumenting/exposition_formats/).
It supports `Counter`, `Gauge`, and `Histogram` — enough to cover
~95 % of HTTP middleware needs without pulling in
`prometheus/client_golang` (which would bring `protobuf`,
`x/sys/unix`, and others).

Wiring:

```go
app.New(cfg).
    WithMetricsEndpoint("/metrics"). // also creates the registry
    Build()
```

…or BYO registry to register application metrics:

```go
reg := metrics.NewRegistry()
queueDepth := reg.Gauge("orders_queue_depth", "Pending orders.", "shard")

app.New(cfg).
    WithMetricsRegistry(reg).
    WithMetricsEndpoint("/metrics").
    Build()
```

Built-in metrics emitted by `MetricsMiddleware`:

| Metric                                | Type      | Labels             |
| ------------------------------------- | --------- | ------------------ |
| `http_requests_total`                 | counter   | `method`, `status` |
| `http_requests_in_flight`             | gauge     | —                  |
| `http_request_duration_seconds`       | histogram | `method`           |

The `/metrics` endpoint, like the health probes, is mounted *outside*
the security chain (no rate limit, no anti-bot) and *outside* the
metrics-recording chain itself — scrape traffic does not pollute
`http_requests_total`.

### Cardinality contract

The registry does **not** bound label-value cardinality. Avoid
high-cardinality labels (`user_id`, `request_path`); they will
eventually exhaust memory. Stick to bounded enumerations: HTTP method,
status class, route template (`/users/:id`, not the rendered path).

## 4. Tracing

The framework only depends on the `Tracer` interface from
`pkg/observability`:

```go
type Tracer interface {
    Start(ctx context.Context, spanName string) (context.Context, Span)
    SpanContextFromContext(ctx context.Context) SpanContext
}
```

`NoopTracer{}` is the default — using it costs zero allocations.
`AppBuilder.WithTracer(t)` wires `TracingMiddleware` at the outer edge
of the request pipeline so every request:

1. Opens a span named `"http <METHOD> <PATH>"`.
2. Reads the resulting `SpanContext`.
3. Attaches `(TraceID, SpanID)` to the request context as a
   `middleware.Correlation`.
4. `LoggerMiddleware` then decorates `http.request`/`http.response`
   log lines with `trace_id` / `span_id`, joining your logs to the
   trace pipeline (Tempo, Jaeger, Datadog APM, …).

A real OpenTelemetry adapter belongs in `github.com/fastygo/otel`
(planned). The application wires it once:

```go
import "github.com/fastygo/otel" // hypothetical

tracer, shutdown := otel.New(ctx, otel.Config{ /* ... */ })
defer shutdown(ctx)

app.New(cfg).WithTracer(tracer).Build()
```

The framework itself never imports `go.opentelemetry.io/*`. This is a
deliberate design decision: OTel pulls 20+ transitive dependencies
(gRPC, protobuf, x/sys), and we want the framework to stay usable in
constrained environments (small CLI tools, embedded services, FaaS)
where tracing isn't worth the binary size cost.

## Composing the four pillars

A production-grade application typically wires all four:

```go
reg := metrics.NewRegistry()
tracer := otel.New( /* ... */ )

app.New(cfg).
    WithTracer(tracer).
    WithMetricsRegistry(reg).
    WithMetricsEndpoint("/metrics").
    WithHealthEndpoints("/healthz", "/readyz").
    AddHealthChecker(health.CheckerFunc("db", pingDB)).
    Use(myFeature{}).
    Build().
    Run()
```

Order in the chain (outermost to innermost):

1. `TracingMiddleware`         — opens span, populates Correlation
2. `MetricsMiddleware`         — records request counters/duration
3. Security chain              — rate limit, anti-bot, headers, …
4. `LoggerMiddleware`          — reads Correlation, emits log lines
5. Application handler

`/metrics`, `/healthz`, `/readyz` short-circuit *before* the security
chain so they remain reachable from probes and scrapers.
