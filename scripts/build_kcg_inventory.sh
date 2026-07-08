#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_PATH="${1:-$ROOT_DIR/kotlin-compiler-golang/inventory/embeddable-index.json}"

JAR_PATH="$($ROOT_DIR/scripts/fetch_kotlin_compiler_embeddable.sh "$ROOT_DIR/tools")"

go run "$ROOT_DIR/cmd/ktcg-inventory" --jar "$JAR_PATH" --out "$OUT_PATH"
