SHELL := /bin/bash

# This Makefile operates on the framework module only.
# Each example under examples/* ships its own Makefile / Bun scripts.

NO_ROOT_IMPORT_CHECK := go run ./scripts/check-no-root-imports.go
COVERAGE_PROFILE := coverage.out
COVERAGE_GATE := go run ./scripts/coverage-gate

.PHONY: test lint lint-go lint-ci vet ci examples coverage coverage-gate verify

test:
	go test ./...

vet:
	go vet ./...

lint:
	go test ./...
	$(NO_ROOT_IMPORT_CHECK)

# lint-go runs golangci-lint. Requires golangci-lint v2.4+ (Go 1.25 support):
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
# CI pins v2.11 via golangci/golangci-lint-action@v9 — keep these in sync.
lint-go:
	golangci-lint run --timeout=5m ./...

lint-ci: lint vet

# coverage runs the test suite for pkg/ with a coverage profile. The
# profile is consumed by coverage-gate (see below) and can also be
# inspected manually with `go tool cover -html=coverage.out`.
coverage:
	go test -covermode=atomic -coverprofile=$(COVERAGE_PROFILE) ./pkg/...

# coverage-gate fails the build when any tracked package drops below
# its declared threshold (see scripts/coverage-gate/main.go).
# Security-critical packages (pkg/auth, pkg/web/security) and pkg/core
# get the tightest bars.
coverage-gate: coverage
	$(COVERAGE_GATE) -profile=$(COVERAGE_PROFILE)

ci: lint-ci coverage-gate

# Build every example. Useful as a smoke test for the framework API surface.
examples:
	bun install
	@for example in examples/*/; do \
		if [ -f "$${example}go.mod" ]; then \
			echo "==> building $${example}"; \
			(cd "$${example}" && bun run vendor:assets && bun run build:css && templ generate ./... && go build ./...) || exit 1; \
		fi; \
	done

verify:
	go build ./pkg/...
	templ generate ./examples/...
	go build ./examples/...
	go test ./pkg/web/...
