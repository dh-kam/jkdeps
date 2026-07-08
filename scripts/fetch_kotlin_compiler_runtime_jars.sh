#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST_DIR="${1:-$ROOT_DIR/tools}"
KOTLIN_VERSION="${KOTLIN_VERSION:-2.0.21}"
KOTLIN_COROUTINES_VERSION="${KOTLIN_COROUTINES_VERSION:-1.6.4}"
TROVE4J_VERSION="${TROVE4J_VERSION:-1.0.20200330}"

mkdir -p "$DEST_DIR"

fetch() {
  local url="$1"
  local out="$2"
  if [[ ! -f "$out" ]]; then
    echo "Downloading $(basename "$out")..." >&2
    curl -fsSL "$url" -o "$out"
  fi
  echo "$out"
}

fetch "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-stdlib/${KOTLIN_VERSION}/kotlin-stdlib-${KOTLIN_VERSION}.jar" \
  "$DEST_DIR/kotlin-stdlib-${KOTLIN_VERSION}.jar"
fetch "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-script-runtime/${KOTLIN_VERSION}/kotlin-script-runtime-${KOTLIN_VERSION}.jar" \
  "$DEST_DIR/kotlin-script-runtime-${KOTLIN_VERSION}.jar"
fetch "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-reflect/${KOTLIN_VERSION}/kotlin-reflect-${KOTLIN_VERSION}.jar" \
  "$DEST_DIR/kotlin-reflect-${KOTLIN_VERSION}.jar"
fetch "https://repo1.maven.org/maven2/org/jetbrains/kotlin/kotlin-daemon-embeddable/${KOTLIN_VERSION}/kotlin-daemon-embeddable-${KOTLIN_VERSION}.jar" \
  "$DEST_DIR/kotlin-daemon-embeddable-${KOTLIN_VERSION}.jar"
fetch "https://repo1.maven.org/maven2/org/jetbrains/intellij/deps/trove4j/${TROVE4J_VERSION}/trove4j-${TROVE4J_VERSION}.jar" \
  "$DEST_DIR/trove4j-${TROVE4J_VERSION}.jar"
fetch "https://repo1.maven.org/maven2/org/jetbrains/kotlinx/kotlinx-coroutines-core-jvm/${KOTLIN_COROUTINES_VERSION}/kotlinx-coroutines-core-jvm-${KOTLIN_COROUTINES_VERSION}.jar" \
  "$DEST_DIR/kotlinx-coroutines-core-jvm-${KOTLIN_COROUTINES_VERSION}.jar"
