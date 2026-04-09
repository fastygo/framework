#!/usr/bin/env bash
set -euo pipefail

# Local preflight aligned with CI:
#   .github/workflows/no-root-imports.yml → go mod download && make ci
#
# Mirrors Makefile `ci` / `lint`: go test ./... && go run ./scripts/check-no-root-imports.go
#
# Usage:
#   ./scripts/preflight.sh
#   PREFLIGHT_CI_PARITY=1 ./scripts/preflight.sh
#   PREFLIGHT_RUN_RACE=1 ./scripts/preflight.sh
#
# Optional:
#   PREFLIGHT_CI_PARITY=1 — set GOWORK=off so resolution matches CI (no local go.work).
#   PREFLIGHT_RUN_RACE=1 — run go test -race after make ci (extra; not in CI workflow today).

RUN_RACE="${PREFLIGHT_RUN_RACE:-0}"

echo "Running go mod download..."
go mod download

if [[ "${PREFLIGHT_CI_PARITY:-}" == "1" ]]; then
	export GOWORK=off
	echo "PREFLIGHT_CI_PARITY=1: GOWORK=off (same as CI without go.work)"
fi

# Same as `make ci` / `make lint` — explicit so it runs without GNU make (e.g. Windows).
echo "Running go test ./..."
go test ./...

echo "Running check-no-root-imports..."
go run ./scripts/check-no-root-imports.go

if [[ "$RUN_RACE" == "1" ]]; then
	if [[ "$(go env CGO_ENABLED)" == "1" ]]; then
		echo "Running race tests..."
		go test ./... -race -count=1
	else
		echo "Warning: PREFLIGHT_RUN_RACE=1 but CGO is disabled; skipping race tests."
	fi
else
	echo "Skipping race tests (set PREFLIGHT_RUN_RACE=1 to enable)."
fi

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	echo "Checking for uncommitted changes..."
	git diff --exit-code
	git diff --cached --exit-code
fi

echo "Preflight OK."
