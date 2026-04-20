# 3. Phase 2 — observability without the SDK tax

Date: 2026-04-19

## Status

Accepted (released as `v0.2.0`).

## Context

After Phase 1 (`v0.1.0`) closed the leak/timeout/CI gaps, the next
critical hole was operability. Without health probes, structured
logs, metrics, and traces, the framework could not be deployed to
any modern orchestrator with confidence.

The naïve solution — depend on `prometheus/client_golang` and
`go.opentelemetry.io/otel` — would have brought 25+ transitive
modules (gRPC, protobuf, x/sys, OTLP exporters) into a project that
today has exactly one indirect dep (`go.uber.org/goleak`, tests only).
For a framework intended to also run in constrained environments
(CLI, embedded, FaaS), that cost is unacceptable.

## Decision

Implement the four observability pillars **inside the framework with
zero external dependencies**, exposing extension points where a real
SDK is genuinely required:

1. **Health** (`pkg/web/health`)
   Custom aggregator with parallel checks and per-check timeout.
   Mounted by `AppBuilder.WithHealthEndpoints` *before* the security
   chain so probes bypass rate limit and anti-bot.

2. **Logs** (`pkg/web/middleware/logger.go`)
   `slog.LogAttrs`-based middleware that decorates every line with
   `request_id` and, when an upstream tracer is wired, `trace_id` /
   `span_id`. Migrated to `LogAttrs` to drop variadic-allocation cost
   on the hot path.

3. **Metrics** (`pkg/web/metrics`)
   From-scratch Prometheus text exposition v0.0.4 with Counter,
   Gauge, and Histogram. ~500 LOC with full label support and
   stable, sorted output. Cardinality is the caller's responsibility.

4. **Traces** (`pkg/observability`)
   Interface-only: `Tracer`, `Span`, `SpanContext`, plus a
   `NoopTracer` default. The framework consumes the interface; a
   real OpenTelemetry adapter ships separately as
   `github.com/fastygo/otel` (planned). `WithTracer` wires the
   middleware that bridges spans to the logger correlation.

`AppBuilder` gains four new opt-in methods:

- `WithHealthEndpoints(live, ready string)` + `AddHealthChecker(c)`
- `WithMetricsRegistry(*metrics.Registry)` + `WithMetricsEndpoint(path)`
- `WithTracer(observability.Tracer)`

Plus six new env-var-driven `Config` fields:

- `LOG_LEVEL`, `LOG_FORMAT`
- `HEALTH_LIVE_PATH`, `HEALTH_READY_PATH`
- `METRICS_PATH`
- `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`

Audit events (`session_issued`, `session_tampered`, `session_expired`,
…) added to `pkg/auth.CookieSession` use the same `slog` handler so
SIEM pipelines can route security events by level (`>= WARN`) without
parsing message text.

## Consequences

### Positive

- Framework `go.mod` still contains exactly one indirect dependency.
- Every observability feature is opt-in: omitting the builder call
  costs zero allocations.
- `/metrics`, `/healthz`, `/readyz` bypass the security chain and
  the metrics-recording chain itself, so probes are reliable and
  scrape traffic does not pollute application RPS dashboards.
- Logger correlation is automatic the moment a tracer is installed —
  no per-handler plumbing.
- Cardinality contract is documented (`docs/OBSERVABILITY.md`)
  rather than enforced at runtime, matching how `prometheus/client_golang`
  treats it.

### Negative

- We re-implement the Prometheus text format. Future spec changes
  (e.g. native histograms, exemplars) need manual upkeep. Estimated
  cost: low — the v0.0.4 format has been stable for a decade.
- Tracing requires a separate adapter package to be useful. Until
  `github.com/fastygo/otel` ships, applications can only consume
  the interface (e.g. with their own minimal tracer or no tracer).

### Deferred to v0.2.1+

- `summary` and `exemplar` metric types (rare in HTTP middleware).
- Native histograms (Prometheus 2.40+).
- A first OTel adapter under `github.com/fastygo/otel`.
- Configurable span name templates (today: raw path).

## References

- Phase 2 plan: `.project/PLAN-phase2-observability.md`
- Operator guide: `docs/OBSERVABILITY.md`
- 12-factor mapping: `docs/12-FACTOR.md`
- Roadmap: `.project/roadmap-framework.md`, section "Phase 2".
