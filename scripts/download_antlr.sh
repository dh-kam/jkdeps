#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANTLR_VERSION="${ANTLR_VERSION:-4.13.2}"
DEST_DIR="${1:-$ROOT_DIR/tools}"
JAR_PATH="$DEST_DIR/antlr-${ANTLR_VERSION}-complete.jar"

mkdir -p "$DEST_DIR"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Downloading ANTLR ${ANTLR_VERSION}..." >&2
  curl -fsSL "https://www.antlr.org/download/antlr-${ANTLR_VERSION}-complete.jar" -o "$JAR_PATH"
fi

echo "$JAR_PATH"
