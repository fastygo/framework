# FastyGo Instant Example

A standalone, deployment-oriented example for the fastest useful FastyGo page shape:

- one prebuilt HTML document;
- inline CSS only;
- no JavaScript;
- no external CSS;
- no fonts;
- no images;
- no favicon file;
- no `/static` directory;
- no localization or theme switching;
- explicit fixed memory budget through `pkg/web/instant.Store`.

The request path does not render templates, parse markdown, read files, or refresh a cache. Startup builds a fixed store, assigns an ETag, logs the budget, and then serves the same immutable bytes.

## Run

```bash
go run ./cmd/server
```

By default the server listens on `127.0.0.1:8080`.

Useful configuration:

```bash
APP_BIND=127.0.0.1:8080
APP_INSTANT_MAX_PAGES=1
APP_INSTANT_MAX_BYTES=65536
APP_HTTP_READ_HEADER_TIMEOUT=2s
APP_HTTP_WRITE_TIMEOUT=10s
```

For constrained containers, set Go runtime memory limits outside the app:

```bash
GOMEMLIMIT=128MiB GOGC=100 go run ./cmd/server
```

## Build

```bash
go build -o bin/instant ./cmd/server
```

This example intentionally has no local `replace` directive. Inside the framework workspace it builds through `go.work`; as a standalone repository, point `github.com/fastygo/framework` in `go.mod` at the published version that contains `pkg/web/instant`.

## Benchmark

Handler benchmark:

```bash
go test ./internal/site -bench=. -benchmem
```

Local load test with `bombardier`:

```bash
go run ./cmd/server
bombardier -c 256 -d 30s -m GET http://127.0.0.1:8080/
```

Local load test with `wrk`:

```bash
go run ./cmd/server
wrk -t4 -c256 -d30s http://127.0.0.1:8080/
```

The load tests measure local HTTP throughput only. They do not include TLS, CDN cache behavior, mobile radio latency, messenger WebView startup cost, or browser paint timing.
