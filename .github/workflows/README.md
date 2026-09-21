# GitHub Actions Workflows

This directory contains the GitHub Actions workflow definitions for the
framework repository. The active workflow is `ci.yml`.

## `ci.yml`

The CI workflow validates the framework Go module on pushes to `main`
and on pull requests.

## Framework Job

The `framework / test` job validates the root module in isolation.

Key settings:

- `actions/setup-go` reads the Go version from `go.mod`.
- `GOWORK=off` is exported before Go commands run. This prevents local
  workspace paths from affecting framework package validation.
- `go test ./...` runs the default offline suite.
- `go vet ./...` runs the standard Go analyzers.

Example applications under `examples/*` are built in their own
repositories. Local golangci-lint remains optional via `make lint-go`.
