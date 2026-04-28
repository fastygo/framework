# Continuous Integration

This project uses GitHub Actions to validate the framework module and the
example applications that consume it. The main workflow is
`.github/workflows/ci.yml`.

## When CI Runs

The workflow runs on:

- pushes to `main`
- pull requests

CI is the source of truth for the full repository validation. Local scripts are
available for fast feedback, but CI is expected to run with the complete tool
chain and GitHub checkout layout.

## Workflow Structure

The workflow has two jobs:

- `framework / lint+test`
- `build examples`

The examples job depends on the framework job. Example builds run only after the
framework module has passed tests, vet, lint, and coverage checks.

## Framework Job

The framework job checks the root Go module only.

Important settings:

- Go version comes from `go.mod`.
- `GOWORK=off` is set explicitly so the root module is tested independently of
  any local sibling repositories.
- Dependencies are downloaded with `go mod download`.
- `make lint-ci` runs the framework test and vet path through the Makefile.
- `make coverage-gate` generates `coverage.out` and enforces per-package
  thresholds.
- `coverage.out` is uploaded as a short-lived artifact for debugging failed
  coverage runs.
- `golangci-lint` is pinned through `golangci/golangci-lint-action@v9` with
  `version: v2.11.4`.

Keep the `golangci-lint` version in `.github/workflows/ci.yml`, `Makefile`,
and `.golangci.yml` aligned.

## Coverage Gate

Coverage policy lives in `scripts/coverage-gate/main.go`.

The gate reads `coverage.out` and fails CI when a tracked package drops below
its threshold. When adding a new important package under `pkg/`, decide whether
it should be tracked by the gate in the same pull request.

Run the gate manually with:

```bash
go test -covermode=atomic -coverprofile=coverage.out ./pkg/...
go run ./scripts/coverage-gate -profile=coverage.out
```

## Examples Job

The examples job builds each example application from the matrix:

- `landing`
- `web`
- `blog`
- `docs`
- `dashboard`

Each example is its own Go module under `examples/*` and has its own Bun build
scripts. The job checks out the framework repository into `framework`, then
checks out sibling UI repositories at the paths expected by the examples'
`replace` directives:

- `@UI8Kit`
- `Blocks`
- `Elements`

This checkout layout matters. Example modules currently replace:

- `github.com/fastygo/framework => ../..`
- `github.com/fastygo/ui8kit => ../../../@UI8Kit`
- `github.com/fastygo/blocks => ../../../Blocks`
- `github.com/fastygo/elements => ../../../Elements`

The examples build step runs:

```bash
bun run build
```

That script vendors UI assets, builds CSS, and runs `templ generate ./...`.
After assets are generated, CI downloads Go dependencies and runs:

```bash
go build ./...
```

## Tool Versions

Current pinned or configured tools:

- Go: from `go.mod`
- Bun: `1.3.3`
- templ: `github.com/a-h/templ/cmd/templ@v0.3.1001`
- golangci-lint: `v2.11.4`

When changing one of these versions, update the workflow and any matching
documentation or comments in the same change.

## Local Checks

For local framework checks, use:

```bash
./scripts/preflight.sh
```

Useful variants are documented in `scripts/README.md`.

The local preflight script can build examples when requested, but the GitHub
Actions examples job remains the canonical validation for the sibling checkout
layout used by CI.

## Maintenance Checklist

When changing CI, verify the following:

- The framework job still runs with `GOWORK=off`.
- Makefile targets used by CI exist and do not recursively duplicate work.
- Coverage thresholds include any new package that should be tracked.
- The examples matrix matches the supported example directories.
- Sibling repository checkout paths still match the examples' local `replace`
  directives.
- Tool version pins are consistent across workflow files, Makefile comments,
  and linter configuration.
