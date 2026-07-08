#!/usr/bin/env bash
set -euo pipefail

DEST_DIR="${1:-$(pwd)/tools}"
KOTLINX_ATOMICFU_VERSION="${KOTLINX_ATOMICFU_VERSION:-0.26.1}"
JAR_PATH="$DEST_DIR/atomicfu-${KOTLINX_ATOMICFU_VERSION}.jar"

mkdir -p "$DEST_DIR"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Downloading atomicfu ${KOTLINX_ATOMICFU_VERSION} ..." >&2
  curl -fsSL "https://repo1.maven.org/maven2/org/jetbrains/kotlinx/atomicfu/${KOTLINX_ATOMICFU_VERSION}/atomicfu-${KOTLINX_ATOMICFU_VERSION}.jar" -o "$JAR_PATH"
fi

echo "$JAR_PATH"
