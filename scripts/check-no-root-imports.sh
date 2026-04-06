#!/usr/bin/env bash

set -euo pipefail

MODULE_ROOT="github.com/fastygo/framework"
FORBIDDEN_PATHS=(
  "$MODULE_ROOT/internal/features"
  "$MODULE_ROOT/internal/site/features"
  "$MODULE_ROOT/views"
  "$MODULE_ROOT/fixtures"
)

FAILED=0

forbidden_regex="$(printf "%s|" "${FORBIDDEN_PATHS[@]}")"
forbidden_regex="${forbidden_regex%|}"

matches=$(grep -R --line-number \
  --include="*.go" \
  --exclude-dir=".git" \
  --exclude-dir="node_modules" \
  -E "\"($forbidden_regex)(/|\\\")" \
  . || true)

if [[ -n "$matches" ]]; then
  echo "Detected forbidden root imports:"
  echo "$matches"
  FAILED=1
fi

if [[ "$FAILED" -eq 1 ]]; then
  echo "Failing due to root-import policy violations."
  exit 1
fi

echo "No-root imports check passed."
