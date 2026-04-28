# GitHub Actions Workflows

This directory contains the GitHub Actions workflow definitions for the
framework repository. The active workflow is `ci.yml`.

## `ci.yml`

The CI workflow validates two layers:

- the framework Go module
- the example applications under `examples/*`

It runs on pushes to `main` and on pull requests.

## Framework Job

The `framework / lint+test` job validates the root module in isolation.

Key settings:

- `actions/setup-go` reads the Go version from `framework/go.mod` when the
  repository is checked out at the root.
- `GOWORK=off` is exported before Go commands run. This prevents local
  workspace paths from affecting framework package validation.
- `make lint-ci` runs tests, the no-root-imports check, and `go vet`.
- `make coverage-gate` creates `coverage.out` and enforces package coverage
  thresholds from `scripts/coverage-gate/main.go`.
- `golangci-lint` is pinned to `v2.11.4`.

## Examples Job

The `build examples` job runs after the framework job passes.

The job uses a matrix for:

- `landing`
- `web`
- `blog`
- `docs`
- `dashboard`

The framework repository is checked out into `framework`. The examples also
depend on local `replace` directives for sibling UI modules, so CI checks out:

- `fastygo/ui8kit` into `@UI8Kit`
- `fastygo/blocks` into `Blocks`
- `fastygo/elements` into `Elements`

Do not change these checkout paths unless the example `go.mod` files are
updated at the same time.

Each example runs:

```bash
bun run build
go mod download
go build ./...
```

`bun run build` vendors UI assets, builds CSS, and runs `templ generate ./...`.

## Updating Workflow Settings

When editing `ci.yml`, keep these files in sync when relevant:

- `Makefile`
- `.golangci.yml`
- `scripts/README.md`
- `docs/CI.md`
- example `go.mod` files under `examples/*`

Tool pins currently used by CI:

- Bun: `1.3.3`
- templ: `v0.3.1001`
- golangci-lint: `v2.11.4`

See `docs/CI.md` for the broader project-level CI guide.
