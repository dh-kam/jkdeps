#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_PATH="${1:-$ROOT_DIR/kotlin-compiler-golang/inventory/runtime-index.json}"

EMBEDDABLE_JAR="$($ROOT_DIR/scripts/fetch_kotlin_compiler_embeddable.sh "$ROOT_DIR/tools")"
STDLIB_JS_KLIB="$($ROOT_DIR/scripts/fetch_kotlin_stdlib_js_klib.sh "$ROOT_DIR/tools")"
BROWSER_JAR="$($ROOT_DIR/scripts/fetch_kotlinx_browser_jar.sh "$ROOT_DIR/tools")"
ATOMICFU_JAR="$($ROOT_DIR/scripts/fetch_kotlinx_atomicfu_jar.sh "$ROOT_DIR/tools")"

go run "$ROOT_DIR/cmd/ktcg-inventory" \
  --jar "$EMBEDDABLE_JAR" \
  --jar "$STDLIB_JS_KLIB" \
  --jar "$BROWSER_JAR" \
  --jar "$ATOMICFU_JAR" \
  --out "$OUT_PATH"
