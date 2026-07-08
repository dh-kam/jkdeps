#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_PATH="${1:-$ROOT_DIR/kotlin-compiler-golang/inventory/browser-index.json}"

BROWSER_JAR="$($ROOT_DIR/scripts/fetch_kotlinx_browser_jar.sh "$ROOT_DIR/tools")"

go run "$ROOT_DIR/cmd/ktcg-inventory" --jar "$BROWSER_JAR" --symbols=false --out "$OUT_PATH"
