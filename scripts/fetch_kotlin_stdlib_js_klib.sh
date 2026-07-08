#!/usr/bin/env bash
set -euo pipefail

DEST_DIR="${1:-$(pwd)/tools}"
KOTLIN_VERSION="${KOTLIN_VERSION:-2.0.21}"
KLIB_PATH="$DEST_DIR/kotlin-stdlib-js-${KOTLIN_VERSION}.klib"

mkdir -p "$DEST_DIR"

if [[ ! -f "$KLIB_PATH" ]]; then
  echo "Downloading kotlin-stdlib-js ${KOTLIN_VERSION} ..." >&2
  curl -fsSL "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-stdlib-js/${KOTLIN_VERSION}/kotlin-stdlib-js-${KOTLIN_VERSION}.klib" -o "$KLIB_PATH"
fi

echo "$KLIB_PATH"
