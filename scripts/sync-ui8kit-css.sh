#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="$ROOT_DIR/.project/styles"
WEB_TARGET_DIR="$ROOT_DIR/internal/site/web/static/css/ui8kit"
DOCS_TARGET_DIR="$ROOT_DIR/internal/site/docs/web/static/css/ui8kit"

if [ ! -d "$SOURCE_DIR" ]; then
  echo "Source styles directory not found: $SOURCE_DIR"
  exit 1
fi

for target in "$WEB_TARGET_DIR" "$DOCS_TARGET_DIR"; do
  mkdir -p "$target"

  cp "$SOURCE_DIR/base.css" "$target/base.css"
  cp "$SOURCE_DIR/shell.css" "$target/shell.css"
  cp "$SOURCE_DIR/components.css" "$target/components.css"
  cp "$SOURCE_DIR/latty.css" "$target/latty.css"

  INPUT_FILE="$ROOT_DIR/.tmp-ui8kit-input.css"
  cat > "$INPUT_FILE" <<EOF
@import "tailwindcss";
@import "./.project/styles/base.css";
@import "./.project/styles/shell.css";
@import "./.project/styles/components.css";
@import "./.project/styles/latty.css";
EOF

  npx tailwindcss -i "$INPUT_FILE" -o "$target/ui8kit.css" --minify
  rm -f "$INPUT_FILE"
done

echo "ui8kit CSS synced from $SOURCE_DIR"
echo " - web:  $WEB_TARGET_DIR"
echo " - docs: $DOCS_TARGET_DIR"
