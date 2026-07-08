#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_PATH="${1:-$ROOT_DIR/kotlin-compiler-golang/inventory/atomicfu-index.json}"

ATOMICFU_JAR="$($ROOT_DIR/scripts/fetch_kotlinx_atomicfu_jar.sh "$ROOT_DIR/tools")"

go run "$ROOT_DIR/cmd/ktcg-inventory" --jar "$ATOMICFU_JAR" --symbols=false --out "$OUT_PATH"
