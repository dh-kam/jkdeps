#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KOTLIN_VERSION="${KOTLIN_VERSION:-2.0.21}"
DEST_DIR="${1:-$ROOT_DIR/tools}"
JAR_PATH="$DEST_DIR/kotlin-compiler-embeddable-${KOTLIN_VERSION}.jar"

mkdir -p "$DEST_DIR"

if [[ ! -f "$JAR_PATH" ]]; then
  echo "Downloading kotlin-compiler-embeddable ${KOTLIN_VERSION}..." >&2
  curl -fsSL "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-compiler-embeddable/${KOTLIN_VERSION}/kotlin-compiler-embeddable-${KOTLIN_VERSION}.jar" -o "$JAR_PATH"
fi

echo "$JAR_PATH"
