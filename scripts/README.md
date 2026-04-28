# Framework Scripts

This directory contains small repository-maintenance scripts for the
framework module. They are intentionally stdlib-first and runnable with
`go run` so local checks do not require a full GNU Make or CGO setup.

## Common Local Preflight

Use `preflight.sh` for the closest local equivalent of the framework CI
job:

```bash
./scripts/preflight.sh
```

The script expands the CI steps directly instead of calling `make`, so it
works in environments where `make` is not installed. It runs dependency
download, tests, the no-root-imports check, `go vet`, coverage profile
generation, and the coverage gate.

Optional tools are skipped cleanly:

- `golangci-lint` runs only when the binary is present.
- `golangci-lint` can be explicitly skipped with `PREFLIGHT_SKIP_LINT=1`.
- Race tests run only when `PREFLIGHT_RUN_RACE=1` and CGO is enabled.
- Example builds run only when `PREFLIGHT_BUILD_EXAMPLES=1` and the needed
  tools (`bun`, `templ`) are available.

Useful local variants:

```bash
# Skip optional lint and dirty-tree checks.
PREFLIGHT_SKIP_LINT=1 PREFLIGHT_SKIP_DIRTY=1 ./scripts/preflight.sh

# Match CI module resolution by disabling go.work.
PREFLIGHT_CI_PARITY=1 ./scripts/preflight.sh

# Collect every failure instead of stopping at the first one.
PREFLIGHT_FAIL_FAST=0 ./scripts/preflight.sh
```

CI remains the source of truth for full lint, race-capable setups, and
platform-specific validation.

## Script Reference

### `preflight.sh`

Local CI parity runner. It is safe to run without `make`; optional checks
are controlled by environment variables documented in the script header.

### `check-no-root-imports.go`

Ensures framework code imports only `github.com/fastygo/framework/pkg/...`
from inside the framework module. This keeps examples and app-level code
from leaking back into `pkg/`.

Run:

```bash
go run ./scripts/check-no-root-imports.go
```

### `coverage-gate`

Enforces per-package coverage thresholds from `coverage.out`.

Run:

```bash
go test -covermode=atomic -coverprofile=coverage.out ./pkg/...
go run ./scripts/coverage-gate -profile=coverage.out
```

Thresholds live in `scripts/coverage-gate/main.go` so coverage policy
changes go through code review.

### `godoc-audit`

Checks exported API documentation under `pkg/`.

Run:

```bash
go run ./scripts/godoc-audit
```

### `layout-audit`

Validates static UI shell wiring in custom `.templ` views that do not use
the standard UI8Kit shell.

Run:

```bash
go run ./scripts/layout-audit ./examples/blog/internal/site/views
```
