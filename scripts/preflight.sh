#!/usr/bin/env bash
set -euo pipefail

# Local preflight aligned with .github/workflows/ci.yml.
#
# Mirrors every step the `framework / lint+test` job runs, in the same
# order, so a green preflight ≈ a green CI run. Optionally also mirrors
# the `examples` matrix job and a few extras (race, go.work parity).
#
# Usage:
#   ./scripts/preflight.sh                         # full local CI parity
#   PREFLIGHT_CI_PARITY=1 ./scripts/preflight.sh   # also unset go.work like CI
#   PREFLIGHT_RUN_RACE=1  ./scripts/preflight.sh   # add `go test -race`
#   PREFLIGHT_BUILD_EXAMPLES=1 ./scripts/preflight.sh
#                                                  # build every example/* (needs templ)
#   PREFLIGHT_SKIP_LINT=1 ./scripts/preflight.sh   # skip golangci-lint (e.g. when not installed)
#   PREFLIGHT_FAIL_FAST=0 ./scripts/preflight.sh   # collect every failure instead of stopping at first
#
# Exit codes:
#   0  – everything passed
#   1  – at least one step failed (look at the summary at the bottom)

RUN_RACE="${PREFLIGHT_RUN_RACE:-0}"
BUILD_EXAMPLES="${PREFLIGHT_BUILD_EXAMPLES:-0}"
SKIP_LINT="${PREFLIGHT_SKIP_LINT:-0}"
FAIL_FAST="${PREFLIGHT_FAIL_FAST:-1}"

# CI uses GOWORK=off so the framework module resolves dependencies the
# same way it does in actions/setup-go. PREFLIGHT_CI_PARITY=1 turns this
# on locally too, which catches go.work-only issues before pushing.
if [[ "${PREFLIGHT_CI_PARITY:-0}" == "1" ]]; then
	export GOWORK=off
	echo "PREFLIGHT_CI_PARITY=1: GOWORK=off (same as CI without go.work)"
fi

# Pretty step runner. Each step prints a banner, runs, records pass/fail.
declare -a FAILURES=()

step() {
	local name="$1"
	shift
	echo
	echo "=== preflight: $name ==="
	if "$@"; then
		echo "--- preflight: $name OK ---"
		return 0
	else
		local rc=$?
		echo "!!! preflight: $name FAILED (exit $rc) !!!" >&2
		FAILURES+=("$name")
		if [[ "$FAIL_FAST" == "1" ]]; then
			summary
			exit 1
		fi
		return 0
	fi
}

summary() {
	echo
	echo "=== preflight summary ==="
	if [[ ${#FAILURES[@]} -eq 0 ]]; then
		echo "All steps passed."
		return 0
	fi
	echo "FAILED steps:"
	local f
	for f in "${FAILURES[@]}"; do
		echo "  - $f"
	done
	return 1
}

# ----- Mirror of .github/workflows/ci.yml: framework / lint+test -----

step "go mod download"            go mod download

# `make ci` = `make lint-ci` (= `make lint` + `make vet`) + `make coverage-gate`.
# We expand it explicitly so this script also works on Windows where GNU
# make may not be available.
step "go test ./..."              go test ./...
step "no-root-imports check"      go run ./scripts/check-no-root-imports.go
step "go vet ./..."               go vet ./...

# Coverage gate uses the same profile path the CI job uploads (coverage.out).
step "coverage profile (./pkg/...)" \
	go test -covermode=atomic -coverprofile=coverage.out ./pkg/...
step "coverage gate"              go run ./scripts/coverage-gate -profile=coverage.out

# golangci-lint runs in CI via golangci/golangci-lint-action@v6.
# Locally it's optional: skip cleanly if the binary is missing so the
# preflight stays usable on fresh checkouts.
if [[ "$SKIP_LINT" != "1" ]]; then
	if command -v golangci-lint >/dev/null 2>&1; then
		# v2 dropped the run.timeout config field; pass it on the CLI instead.
		step "golangci-lint run"  golangci-lint run --timeout=5m ./...
	else
		echo
		echo "=== preflight: golangci-lint SKIPPED ==="
		echo "Install v2.4+ (required for Go 1.25, CI pins v2.11):"
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4"
		echo "Or set PREFLIGHT_SKIP_LINT=1 to silence this notice."
	fi
fi

# ----- Optional: mirror of the `examples` matrix job -----
#
# Catches API breaks that only show up when an example consumes the
# framework via its own go.mod. This now mirrors the example CI more
# closely: Bun install at the workspace root, then `bun run build`
# (vendor assets + build css + templ generate) followed by `go build`.
if [[ "$BUILD_EXAMPLES" == "1" ]]; then
	if ! command -v templ >/dev/null 2>&1; then
		echo
		echo "=== preflight: examples SKIPPED ==="
		echo "templ not found. Install with:"
		echo "  go install github.com/a-h/templ/cmd/templ@v0.3.1001"
	elif ! command -v bun >/dev/null 2>&1; then
		echo
		echo "=== preflight: examples SKIPPED ==="
		echo "bun not found. Install Bun 1.3+ to mirror the example asset pipeline."
	else
		step "bun install" bun install
		for example in examples/*/; do
			[[ -f "${example}go.mod" ]] || continue
			step "example: ${example} (bun run build)" \
				bash -c "cd '${example}' && bun run build"
			step "example: ${example} (go build ./...)" \
				bash -c "cd '${example}' && go mod download && go build ./..."
		done
	fi
fi

# ----- Optional extras (not in CI today, but cheap to run locally) -----

if [[ "$RUN_RACE" == "1" ]]; then
	if [[ "$(go env CGO_ENABLED)" == "1" ]]; then
		step "go test -race"      go test ./... -race -count=1
	else
		echo
		echo "=== preflight: race tests SKIPPED ==="
		echo "PREFLIGHT_RUN_RACE=1 but CGO is disabled; race detector requires CGO."
	fi
fi

# Catches the classic "I forgot to commit generated files" footgun.
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	step "no uncommitted changes" bash -c "git diff --exit-code && git diff --cached --exit-code"
fi

if summary; then
	echo "Preflight OK."
	exit 0
fi
exit 1
