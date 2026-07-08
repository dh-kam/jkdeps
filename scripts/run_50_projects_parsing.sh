#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAMPLES_DIR="${1:-/tmp/jkdeps-50-projects-sources}"
OUT_DIR="${2:-/tmp/jkdeps-50-projects-out}"
MANIFEST="$ROOT_DIR/scripts/oss_50_projects_targets.txt"
REPORT_PATH="$ROOT_DIR/docs/50-projects-parsing-result.md"
JKDEPS_DEFAULT_BIN="/tmp/jkdeps-smoke-bin"

mkdir -p "$OUT_DIR"

if [[ ! -x "$JKDEPS_DEFAULT_BIN" ]]; then
  go build -o "$JKDEPS_DEFAULT_BIN" ./cmd/jkdeps
fi

OSS_MATRIX_MANIFEST="$MANIFEST" \
OSS_MATRIX_AUTO_TARGET="${OSS_MATRIX_AUTO_TARGET:-1}" \
OSS_MATRIX_AUTO_MAX_FILES="${OSS_MATRIX_AUTO_MAX_FILES:-120}" \
OSS_MATRIX_COMMAND="${OSS_MATRIX_COMMAND:-smoke-parse}" \
OSS_MATRIX_RESUME="${OSS_MATRIX_RESUME:-1}" \
JKDEPS_BIN="${JKDEPS_BIN:-$JKDEPS_DEFAULT_BIN}" \
PROJECT_TIMEOUT="${PROJECT_TIMEOUT:-900}" \
JAVA_GRAMMAR="${JAVA_GRAMMAR:-java20}" \
"$ROOT_DIR/scripts/oss_dependency_matrix.sh" "$SAMPLES_DIR" "$OUT_DIR"

"$ROOT_DIR/scripts/run_50_projects_roundtrip.sh" "$OUT_DIR"

python3 "$ROOT_DIR/scripts/render_50_projects_parsing_report.py" \
  --summary "$OUT_DIR/summary.csv" \
  --roundtrip-summary "$OUT_DIR/roundtrip-summary.csv" \
  --output "$REPORT_PATH" \
  --manifest "$MANIFEST" \
  --samples-dir "$SAMPLES_DIR" \
  --out-dir "$OUT_DIR"

echo "report written: $REPORT_PATH"
