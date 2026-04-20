#!/usr/bin/env bash
# Syncs UI8Kit CSS/JS bundles into the static directory of an example app.
#
# Usage:
#   scripts/sync-ui8kit-css.sh <static_root>
#
# Where <static_root> is the example's static directory, e.g.
#   examples/web/web/static
#
# Layout produced under <static_root>:
#   <static_root>/css/ui8kit/{base,shell,components,latty,prose}.css
#   <static_root>/css/fonts.css
#   <static_root>/fonts/outfit/{cyrillic,latin}/...
#   <static_root>/js/ui8kit.js
#
# Resolution order for the UI8Kit module:
#   1. `go list -m github.com/fastygo/ui8kit` (uses local go.work / replace).
#   2. `go mod download -json github.com/fastygo/ui8kit` (downloads to module cache).
#
# Run from the example directory (the directory that contains its own go.mod):
#   cd examples/web && bash ../../scripts/sync-ui8kit-css.sh web/static
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <static_root>" >&2
  exit 64
fi

STATIC_ROOT="$1"
mkdir -p "$STATIC_ROOT"
STATIC_ROOT="$(cd "$STATIC_ROOT" && pwd)"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FONT_SOURCE_DIR="$REPO_ROOT/pkg/fonts"

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to locate github.com/fastygo/ui8kit" >&2
  exit 1
fi

resolve_ui8kit_dir() {
  local dir=""
  local download_json=""

  dir="$(go list -m -f '{{.Dir}}' github.com/fastygo/ui8kit 2>/dev/null || true)"
  dir="$(printf '%s' "$dir" | tr -d '\r\n')"
  if [ -n "$dir" ] && [ -d "$dir/styles" ] && [ -d "$dir/js" ]; then
    printf '%s\n' "$dir"
    return 0
  fi

  download_json="$(go mod download -json github.com/fastygo/ui8kit 2>/dev/null || true)"
  dir="$(printf '%s\n' "$download_json" | sed -n 's/^[[:space:]]*"Dir":[[:space:]]*"\(.*\)",$/\1/p' | head -n 1)"
  dir="$(printf '%s' "$dir" | tr -d '\r\n')"
  dir="${dir//\\\\/\\}"
  if [ -n "$dir" ] && [ -d "$dir/styles" ] && [ -d "$dir/js" ]; then
    printf '%s\n' "$dir"
    return 0
  fi

  return 1
}

UI8KIT_DIR="$(resolve_ui8kit_dir || true)"
if [ -z "$UI8KIT_DIR" ]; then
  echo "Failed to resolve github.com/fastygo/ui8kit." >&2
  echo "Make sure the example's go.mod requires it (and run 'go mod download' first)." >&2
  exit 1
fi

STYLES_SOURCE_DIR="$UI8KIT_DIR/styles"
JS_SOURCE_DIR="$UI8KIT_DIR/js"

if [ ! -d "$STYLES_SOURCE_DIR" ]; then
  echo "Source styles directory not found: $STYLES_SOURCE_DIR" >&2
  exit 1
fi
if [ ! -d "$JS_SOURCE_DIR" ]; then
  echo "Source js directory not found: $JS_SOURCE_DIR" >&2
  exit 1
fi
if [ ! -d "$FONT_SOURCE_DIR" ]; then
  echo "Source fonts directory not found: $FONT_SOURCE_DIR" >&2
  exit 1
fi

JS_BUNDLE_ORDER=(
  "$JS_SOURCE_DIR/core.js"
  "$JS_SOURCE_DIR/theme.js"
  "$JS_SOURCE_DIR/dialog.js"
  "$JS_SOURCE_DIR/accordion.js"
  "$JS_SOURCE_DIR/tabs.js"
  "$JS_SOURCE_DIR/combobox.js"
  "$JS_SOURCE_DIR/tooltip.js"
  "$JS_SOURCE_DIR/alert.js"
  "$JS_SOURCE_DIR/locale.js"
)

CSS_TARGET_DIR="$STATIC_ROOT/css/ui8kit"
JS_TARGET_DIR="$STATIC_ROOT/js"
mkdir -p "$CSS_TARGET_DIR" "$JS_TARGET_DIR"

cp "$STYLES_SOURCE_DIR/base.css"       "$CSS_TARGET_DIR/base.css"
cp "$STYLES_SOURCE_DIR/shell.css"      "$CSS_TARGET_DIR/shell.css"
cp "$STYLES_SOURCE_DIR/components.css" "$CSS_TARGET_DIR/components.css"
cp "$STYLES_SOURCE_DIR/latty.css"      "$CSS_TARGET_DIR/latty.css"
cp "$STYLES_SOURCE_DIR/prose.css"      "$CSS_TARGET_DIR/prose.css"

cp "$FONT_SOURCE_DIR/outfit.css"       "$STATIC_ROOT/css/fonts.css"
rm -rf "$STATIC_ROOT/fonts/outfit"
mkdir -p "$STATIC_ROOT/fonts"
cp -r "$FONT_SOURCE_DIR/outfit"        "$STATIC_ROOT/fonts/"

cat "${JS_BUNDLE_ORDER[@]}" > "$JS_TARGET_DIR/ui8kit.js"

echo "ui8kit assets synced into $STATIC_ROOT"
echo " - css:   $CSS_TARGET_DIR"
echo " - fonts: $STATIC_ROOT/fonts/outfit"
echo " - js:    $JS_TARGET_DIR/ui8kit.js"
