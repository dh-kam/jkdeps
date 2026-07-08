#!/usr/bin/env bash
set -euo pipefail

DEST_DIR="${1:-$(pwd)/tools}"
KOTLINX_BROWSER_VERSION="${KOTLINX_BROWSER_VERSION:-0.5.0}"
JAR_PATH="$DEST_DIR/kotlinx-browser-${KOTLINX_BROWSER_VERSION}.jar"

mkdir -p "$DEST_DIR"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Downloading kotlinx-browser ${KOTLINX_BROWSER_VERSION} ..." >&2
  curl -fsSL "https://repo1.maven.org/maven2/org/jetbrains/kotlinx/kotlinx-browser/${KOTLINX_BROWSER_VERSION}/kotlinx-browser-${KOTLINX_BROWSER_VERSION}.jar" -o "$JAR_PATH"
fi

echo "$JAR_PATH"
