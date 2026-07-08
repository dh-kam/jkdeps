#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_PATH="${1:-$ROOT_DIR/kotlin-compiler-golang/inventory/stdlib-js-index.json}"

KLIB_PATH="$($ROOT_DIR/scripts/fetch_kotlin_stdlib_js_klib.sh "$ROOT_DIR/tools")"

go run "$ROOT_DIR/cmd/ktcg-inventory" --jar "$KLIB_PATH" --symbols=false --out "$OUT_PATH"
