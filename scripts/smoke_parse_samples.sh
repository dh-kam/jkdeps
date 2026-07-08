#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAMPLES_DIR="${1:-/tmp/jkdeps-samples}"
WORKERS="${WORKERS:-$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)}"
RUN_GUAVA_STRESS="${RUN_GUAVA_STRESS:-0}"
RUN_GUAVA_STRESS_STRICT="${RUN_GUAVA_STRESS_STRICT:-0}"
GUAVA_STRESS_TIMEOUT="${GUAVA_STRESS_TIMEOUT:-3s}"

"$ROOT_DIR/scripts/fetch_samples.sh" "$SAMPLES_DIR"

cd "$ROOT_DIR"

echo "\\n== Guava (Java-heavy module) =="
go run ./cmd/jkdeps smoke-parse \
  --repo "$SAMPLES_DIR/guava/guava/src/com/google/common/base" \
  --java-grammar java20 \
  --workers "$WORKERS" \
  --max-errors 20 \
  --fail-on-error=true

if [[ "$RUN_GUAVA_STRESS_STRICT" -eq 1 ]]; then
  echo "\\n== Guava stress module (strict) =="
  go run ./cmd/jkdeps smoke-parse \
    --repo "$SAMPLES_DIR/guava/guava/src/com/google/common/util/concurrent" \
    --java-grammar java20 \
    --workers "$WORKERS" \
    --max-errors 20 \
    --fail-on-error=true
elif [[ "$RUN_GUAVA_STRESS" -eq 1 ]]; then
  echo "\\n== Guava stress module (timeout-lenient) =="
  go run ./cmd/jkdeps smoke-parse \
    --repo "$SAMPLES_DIR/guava/guava/src/com/google/common/util/concurrent" \
    --java-grammar java20 \
    --workers "$WORKERS" \
    --max-errors 20 \
    --file-timeout "$GUAVA_STRESS_TIMEOUT" \
    --fail-on-error=false
fi

echo "\\n== kotlinx.coroutines (Kotlin-heavy module) =="
go run ./cmd/jkdeps smoke-parse \
  --repo "$SAMPLES_DIR/kotlinx.coroutines/kotlinx-coroutines-core/common/src" \
  --java-grammar java20 \
  --workers "$WORKERS" \
  --max-errors 20 \
  --fail-on-error=false
