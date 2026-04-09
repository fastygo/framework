#!/usr/bin/env bash
# Syncs ui8kit CSS/JS from the resolved github.com/fastygo/ui8kit module (local go.work/replace or module cache).
# Framework-owned fonts remain under pkg/fonts.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to locate github.com/fastygo/ui8kit (go list -m)." >&2
  exit 1
fi

# Resolves to local module dir or GOMODCACHE copy; works with go.work and plain require.
UI8KIT_DIR="$(go list -m -f '{{.Dir}}' github.com/fastygo/ui8kit)"
STYLES_SOURCE_DIR="$UI8KIT_DIR/styles"
JS_SOURCE_DIR="$UI8KIT_DIR/js"
FONT_SOURCE_DIR="$ROOT_DIR/pkg/fonts"

WEB_TARGET_DIR="$ROOT_DIR/internal/site/web/static/css/ui8kit"
DOCS_TARGET_DIR="$ROOT_DIR/internal/site/docs/web/static/css/ui8kit"
WEB_JS_TARGET_DIR="$ROOT_DIR/internal/site/web/static/js"
DOCS_JS_TARGET_DIR="$ROOT_DIR/internal/site/docs/web/static/js"
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

echo "ui8kit module: $UI8KIT_DIR"

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

for target in "$WEB_TARGET_DIR" "$DOCS_TARGET_DIR"; do
  STATIC_DIR="$(dirname "$(dirname "$target")")"

  mkdir -p "$target"

  cp "$STYLES_SOURCE_DIR/base.css" "$target/base.css"
  cp "$STYLES_SOURCE_DIR/shell.css" "$target/shell.css"
  cp "$STYLES_SOURCE_DIR/components.css" "$target/components.css"
  cp "$STYLES_SOURCE_DIR/latty.css" "$target/latty.css"
  cp "$STYLES_SOURCE_DIR/prose.css" "$target/prose.css"
  cp "$FONT_SOURCE_DIR/outfit.css" "$target/../fonts.css"
  rm -rf "$STATIC_DIR/fonts/outfit"
  mkdir -p "$STATIC_DIR/fonts"
  cp -r "$FONT_SOURCE_DIR/outfit" "$STATIC_DIR/fonts/"
done

for target in "$WEB_JS_TARGET_DIR" "$DOCS_JS_TARGET_DIR"; do
  mkdir -p "$target"

  cat "${JS_BUNDLE_ORDER[@]}" > "$target/ui8kit.js"
done

echo "ui8kit assets synced"
echo " - web:  $WEB_TARGET_DIR"
echo " - docs: $DOCS_TARGET_DIR"
echo " - js:   ui8kit.js"
