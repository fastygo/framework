#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="$ROOT_DIR/internal/site/web/static/css/ui8kit"

GOMODCACHE="$(go env GOMODCACHE)"
UI8KIT_VERSION="v0.2.1"
UI8KIT_MODULE_PATH="$GOMODCACHE/github.com/fastygo/ui8kit@$UI8KIT_VERSION"

if [ ! -d "$UI8KIT_MODULE_PATH" ]; then
  echo "Go module cache does not contain github.com/fastygo/ui8kit@$UI8KIT_VERSION."
  echo "Run: go mod download github.com/fastygo/ui8kit@$UI8KIT_VERSION"
  exit 1
fi

mkdir -p "$TARGET_DIR"

cp "$UI8KIT_MODULE_PATH/styles/base.css" "$TARGET_DIR/base.css"
cp "$UI8KIT_MODULE_PATH/styles/shell.css" "$TARGET_DIR/shell.css"
cp "$UI8KIT_MODULE_PATH/styles/components.css" "$TARGET_DIR/components.css"
cp "$UI8KIT_MODULE_PATH/styles/latty.css" "$TARGET_DIR/latty.css"

echo "ui8kit CSS synced to $TARGET_DIR"
