#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAMPLES_DIR="${1:-/tmp/jkdeps-samples}"
OUT_DIR="${2:-/tmp/jkdeps-porting}"
WORKERS="${WORKERS:-$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)}"
INVENTORY_PATH="${INVENTORY_PATH:-$ROOT_DIR/kotlin-compiler-golang/inventory/runtime-index.json}"

GUAVA_SRC="$SAMPLES_DIR/guava/guava/src/com/google/common/base"
GUAVA_STRESS_SRC="$SAMPLES_DIR/guava/guava/src/com/google/common/util/concurrent"
KOTLIN_COMMON_SRC="$SAMPLES_DIR/kotlinx.coroutines/kotlinx-coroutines-core/common/src"
KOTLIN_JS_SRC="$SAMPLES_DIR/kotlinx.coroutines/kotlinx-coroutines-core/js/src"
KOTLIN_JVM_SRC="$SAMPLES_DIR/kotlinx.coroutines/kotlinx-coroutines-core/jvm/src"
KOTLIN_CORE_SRC="$SAMPLES_DIR/kotlinx.coroutines/kotlinx-coroutines-core"
RUN_GUAVA_STRESS="${RUN_GUAVA_STRESS:-0}"
RUN_GUAVA_STRESS_STRICT="${RUN_GUAVA_STRESS_STRICT:-0}"
GUAVA_STRESS_TIMEOUT="${GUAVA_STRESS_TIMEOUT:-3s}"
KCG_FILE_TIMEOUT="${KCG_FILE_TIMEOUT:-}"
RUN_KOTLIN_JVM="${RUN_KOTLIN_JVM:-0}"
RUN_KOTLIN_CORE="${RUN_KOTLIN_CORE:-0}"
RUN_KOTLIN_CORE_MIXED_GRAPH="${RUN_KOTLIN_CORE_MIXED_GRAPH:-$RUN_KOTLIN_CORE}"
RUN_KOTLIN_CORE_MIXED_DIR_GRAPH="${RUN_KOTLIN_CORE_MIXED_DIR_GRAPH:-$RUN_KOTLIN_CORE_MIXED_GRAPH}"
RUN_KOTLIN_OFFICIAL_PARITY="${RUN_KOTLIN_OFFICIAL_PARITY:-0}"
KOTLIN_CORE_MIXED_INCLUDE_PREFIX="${KOTLIN_CORE_MIXED_INCLUDE_PREFIX:-kotlinx.coroutines}"
KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX="${KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX:-}"
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO="${KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO:-0}"
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL="${KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL:-0}"
KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH="${KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH:-0}"
KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH="${KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH:-0}"
KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH="${KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH:-0}"
KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH="${KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH:-0}"
KCG_COMMON_MAX_FILES_WITH_DIAGNOSTICS="${KCG_COMMON_MAX_FILES_WITH_DIAGNOSTICS:-10}"
KCG_COMMON_MAX_TOTAL_DIAGNOSTICS="${KCG_COMMON_MAX_TOTAL_DIAGNOSTICS:-70}"
KCG_JS_MAX_FILES_WITH_DIAGNOSTICS="${KCG_JS_MAX_FILES_WITH_DIAGNOSTICS:-2}"
KCG_JS_MAX_TOTAL_DIAGNOSTICS="${KCG_JS_MAX_TOTAL_DIAGNOSTICS:-10}"
KCG_JVM_MAX_FILES_WITH_DIAGNOSTICS="${KCG_JVM_MAX_FILES_WITH_DIAGNOSTICS:-8}"
KCG_JVM_MAX_TOTAL_DIAGNOSTICS="${KCG_JVM_MAX_TOTAL_DIAGNOSTICS:-40}"
KCG_CORE_MAX_FILES_WITH_DIAGNOSTICS="${KCG_CORE_MAX_FILES_WITH_DIAGNOSTICS:-35}"
KCG_CORE_MAX_TOTAL_DIAGNOSTICS="${KCG_CORE_MAX_TOTAL_DIAGNOSTICS:-220}"
KCG_LOW_MEMORY="${KCG_LOW_MEMORY:-0}"
KCG_MIXED_GRAPH_MIN_EDGE_COUNT_EXPLICIT=0
if [[ -n "${KCG_MIXED_GRAPH_MIN_EDGE_COUNT+x}" ]]; then
  KCG_MIXED_GRAPH_MIN_EDGE_COUNT_EXPLICIT=1
fi
KCG_MIXED_GRAPH_MIN_EDGE_COUNT="${KCG_MIXED_GRAPH_MIN_EDGE_COUNT:-}"
KCG_PARSER_BACKEND="${KCG_PARSER_BACKEND:-antlr}"
KCG_MIXED_GRAPH_WITH_INVENTORY_EXPLICIT=0
if [[ -n "${KCG_MIXED_GRAPH_WITH_INVENTORY+x}" ]]; then
  KCG_MIXED_GRAPH_WITH_INVENTORY_EXPLICIT=1
fi
KCG_MIXED_GRAPH_WITH_INVENTORY="${KCG_MIXED_GRAPH_WITH_INVENTORY:-1}"

if [[ "$KCG_LOW_MEMORY" == "1" ]]; then
  if [[ "$KCG_MIXED_GRAPH_WITH_INVENTORY_EXPLICIT" -eq 0 ]]; then
    KCG_MIXED_GRAPH_WITH_INVENTORY=0
  fi
  if [[ "$KCG_MIXED_GRAPH_MIN_EDGE_COUNT_EXPLICIT" -eq 0 ]]; then
    KCG_MIXED_GRAPH_MIN_EDGE_COUNT=5
  fi
  if [[ "$KCG_PARSER_BACKEND" == "antlr" ]]; then
    KCG_PARSER_BACKEND="embeddable"
  fi
  WORKERS=1
fi

run_and_log() {
  local name="$1"
  shift
  local log_path="$OUT_DIR/${name}.log"
  echo
  echo "== $name =="
  "$@" 2>&1 | tee "$log_path"
}

echo "== Fetch samples =="
"$ROOT_DIR/scripts/fetch_samples.sh" "$SAMPLES_DIR"

echo "== Ensure runtime inventory =="
needs_rebuild=0
if [[ ! -f "$INVENTORY_PATH" ]]; then
  needs_rebuild=1
else
  if ! python3 - "$INVENTORY_PATH" <<'PY'
import json,sys
path=sys.argv[1]
with open(path) as f:
    data=json.load(f)
packages={item.get("package","") for item in data.get("packages",[])}
sys.exit(0 if "kotlinx.atomicfu" in packages else 1)
PY
  then
    needs_rebuild=1
  fi
fi

if [[ "$needs_rebuild" -eq 1 ]]; then
  "$ROOT_DIR/scripts/build_kcg_runtime_inventory.sh" "$INVENTORY_PATH"
fi

mkdir -p "$OUT_DIR"
cd "$ROOT_DIR"

guava_head_ref="$(git -C "$SAMPLES_DIR/guava" rev-parse HEAD)"
kotlinx_coroutines_head_ref="$(git -C "$SAMPLES_DIR/kotlinx.coroutines" rev-parse HEAD)"

metadata_path="$OUT_DIR/porting-run-metadata.json"
SAMPLES_DIR="$SAMPLES_DIR" \
OUT_DIR="$OUT_DIR" \
INVENTORY_PATH="$INVENTORY_PATH" \
GUAVA_HEAD_REF="$guava_head_ref" \
KOTLINX_COROUTINES_HEAD_REF="$kotlinx_coroutines_head_ref" \
WORKERS="$WORKERS" \
RUN_GUAVA_STRESS="$RUN_GUAVA_STRESS" \
RUN_GUAVA_STRESS_STRICT="$RUN_GUAVA_STRESS_STRICT" \
GUAVA_STRESS_TIMEOUT="$GUAVA_STRESS_TIMEOUT" \
KCG_FILE_TIMEOUT="$KCG_FILE_TIMEOUT" \
RUN_KOTLIN_JVM="$RUN_KOTLIN_JVM" \
RUN_KOTLIN_CORE="$RUN_KOTLIN_CORE" \
RUN_KOTLIN_CORE_MIXED_GRAPH="$RUN_KOTLIN_CORE_MIXED_GRAPH" \
RUN_KOTLIN_CORE_MIXED_DIR_GRAPH="$RUN_KOTLIN_CORE_MIXED_DIR_GRAPH" \
RUN_KOTLIN_OFFICIAL_PARITY="$RUN_KOTLIN_OFFICIAL_PARITY" \
KOTLIN_CORE_MIXED_INCLUDE_PREFIX="$KOTLIN_CORE_MIXED_INCLUDE_PREFIX" \
KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX="$KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX" \
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO="$KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO" \
KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL="$KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL" \
KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH" \
KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH" \
KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH" \
KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH" \
KCG_PARSER_BACKEND="$KCG_PARSER_BACKEND" \
KCG_LOW_MEMORY="$KCG_LOW_MEMORY" \
KCG_MIXED_GRAPH_WITH_INVENTORY="$KCG_MIXED_GRAPH_WITH_INVENTORY" \
KCG_MIXED_GRAPH_MIN_EDGE_COUNT="$KCG_MIXED_GRAPH_MIN_EDGE_COUNT" \
python3 - "$metadata_path" <<'PY'
import datetime
import hashlib
import json
import os
import sys
from pathlib import Path

metadata_path = Path(sys.argv[1])
inventory_path = Path(os.environ["INVENTORY_PATH"])
inventory_sha256 = ""
if inventory_path.is_file():
    inventory_sha256 = hashlib.sha256(inventory_path.read_bytes()).hexdigest()

payload = {
    "generated_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "samples_dir": os.environ["SAMPLES_DIR"],
    "out_dir": os.environ["OUT_DIR"],
    "sample_refs": {
        "guava": os.environ["GUAVA_HEAD_REF"],
        "kotlinx.coroutines": os.environ["KOTLINX_COROUTINES_HEAD_REF"],
    },
    "inventory": {
        "path": str(inventory_path),
        "sha256": inventory_sha256,
    },
    "run_flags": {
        "workers": os.environ["WORKERS"],
        "run_guava_stress": os.environ["RUN_GUAVA_STRESS"],
        "run_guava_stress_strict": os.environ["RUN_GUAVA_STRESS_STRICT"],
        "guava_stress_timeout": os.environ["GUAVA_STRESS_TIMEOUT"],
        "kcg_file_timeout": os.environ["KCG_FILE_TIMEOUT"],
        "run_kotlin_jvm": os.environ["RUN_KOTLIN_JVM"],
        "run_kotlin_core": os.environ["RUN_KOTLIN_CORE"],
        "run_kotlin_core_mixed_graph": os.environ["RUN_KOTLIN_CORE_MIXED_GRAPH"],
        "run_kotlin_core_mixed_dir_graph": os.environ["RUN_KOTLIN_CORE_MIXED_DIR_GRAPH"],
        "run_kotlin_official_parity": os.environ["RUN_KOTLIN_OFFICIAL_PARITY"],
        "kcg_parser_backend": os.environ["KCG_PARSER_BACKEND"],
        "kcg_low_memory": os.environ["KCG_LOW_MEMORY"],
        "kotlin_official_parity_max_missing_in_go": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO"],
        "kotlin_official_parity_max_missing_in_official": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL"],
        "kotlin_official_parity_max_parse_status_mismatch": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH"],
        "kotlin_official_parity_max_package_mismatch": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH"],
        "kotlin_official_parity_max_import_mismatch": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH"],
        "kotlin_official_parity_max_declaration_mismatch": os.environ["KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH"],
        "kcg_mixed_graph_with_inventory": os.environ["KCG_MIXED_GRAPH_WITH_INVENTORY"],
        "kotlin_core_mixed_include_prefix": os.environ["KOTLIN_CORE_MIXED_INCLUDE_PREFIX"],
        "kotlin_core_mixed_dir_include_prefix": os.environ["KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX"],
        "kcg_mixed_graph_min_edge_count": os.environ["KCG_MIXED_GRAPH_MIN_EDGE_COUNT"],
    },
}

metadata_path.parent.mkdir(parents=True, exist_ok=True)
metadata_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
echo "saved: $metadata_path"

kcg_timeout_args=()
if [[ -n "$KCG_FILE_TIMEOUT" ]]; then
  kcg_timeout_args+=(--file-timeout "$KCG_FILE_TIMEOUT")
fi

kcg_mixed_graph_edge_args=()
if [[ -n "$KCG_MIXED_GRAPH_MIN_EDGE_COUNT" ]]; then
  kcg_mixed_graph_edge_args+=(--min-edge-count "$KCG_MIXED_GRAPH_MIN_EDGE_COUNT")
fi

kcg_parser_backend_args=()
if [[ -n "$KCG_PARSER_BACKEND" ]]; then
  kcg_parser_backend_args+=(--parser-backend "$KCG_PARSER_BACKEND")
fi

kcg_dir_include_args=()
if [[ -n "$KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX" ]]; then
  kcg_dir_include_args+=(--include-prefix "$KOTLIN_CORE_MIXED_DIR_INCLUDE_PREFIX")
fi

kcg_mixed_graph_inventory_args=()
if [[ "$KCG_MIXED_GRAPH_WITH_INVENTORY" == "1" ]]; then
  kcg_mixed_graph_inventory_args+=(--inventory "$INVENTORY_PATH")
fi

run_and_log "guava_smoke_parse" \
  go run ./cmd/jkdeps smoke-parse \
    --repo "$GUAVA_SRC" \
    --java-grammar java20 \
    --workers "$WORKERS" \
    --max-errors 20 \
    --fail-on-error=true

run_and_log "guava_graph_filtered" \
  go run ./cmd/jkdeps graph \
    --repo "$GUAVA_SRC" \
    --java-grammar java20 \
    --fail-on-error=true \
    --group-by package \
    --include-prefix com.google.common \
    --min-edge-count 2 \
    --out "$OUT_DIR/guava-package"

if [[ "$RUN_GUAVA_STRESS_STRICT" -eq 1 ]]; then
  run_and_log "guava_stress_smoke_strict" \
    go run ./cmd/jkdeps smoke-parse \
      --repo "$GUAVA_STRESS_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --max-errors 20 \
      --fail-on-error=true

  run_and_log "guava_stress_graph_strict" \
    go run ./cmd/jkdeps graph \
      --repo "$GUAVA_STRESS_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --fail-on-error=true \
      --group-by package \
      --include-prefix com.google.common \
      --min-edge-count 2 \
      --out "$OUT_DIR/guava-stress-package"
elif [[ "$RUN_GUAVA_STRESS" -eq 1 ]]; then
  run_and_log "guava_stress_smoke_timeout_lenient" \
    go run ./cmd/jkdeps smoke-parse \
      --repo "$GUAVA_STRESS_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --max-errors 20 \
      --file-timeout "$GUAVA_STRESS_TIMEOUT" \
      --fail-on-error=false

  run_and_log "guava_stress_graph_timeout_lenient" \
    go run ./cmd/jkdeps graph \
      --repo "$GUAVA_STRESS_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --max-errors-per-file 20 \
      --file-timeout "$GUAVA_STRESS_TIMEOUT" \
      --lenient \
      --group-by package \
      --include-prefix com.google.common \
      --min-edge-count 2 \
      --out "$OUT_DIR/guava-stress-package"
fi

echo
echo "== kotlin_common_acceptance =="
go run ./cmd/kotlin-compiler-golang acceptance \
  --repo "$KOTLIN_COMMON_SRC" \
  "${kcg_parser_backend_args[@]}" \
  --inventory "$INVENTORY_PATH" \
  "${kcg_timeout_args[@]}" \
  --lenient \
  --max-failed-files 0 \
  --max-unresolved-imports 0 \
  --max-files-with-diagnostics "$KCG_COMMON_MAX_FILES_WITH_DIAGNOSTICS" \
  --max-total-diagnostics "$KCG_COMMON_MAX_TOTAL_DIAGNOSTICS" \
  --out "$OUT_DIR/kotlin-common-acceptance.json"

echo
echo "== kotlin_common_resolve =="
go run ./cmd/kotlin-compiler-golang resolve \
  --repo "$KOTLIN_COMMON_SRC" \
  "${kcg_parser_backend_args[@]}" \
  --inventory "$INVENTORY_PATH" \
  "${kcg_timeout_args[@]}" \
  --json > "$OUT_DIR/kotlin-common-resolve.json"
echo "saved: $OUT_DIR/kotlin-common-resolve.json"

echo
echo "== kotlin_js_acceptance_lenient =="
go run ./cmd/kotlin-compiler-golang acceptance \
  --repo "$KOTLIN_JS_SRC" \
  "${kcg_parser_backend_args[@]}" \
  --inventory "$INVENTORY_PATH" \
  "${kcg_timeout_args[@]}" \
  --lenient \
  --max-failed-files 0 \
  --max-unresolved-imports 0 \
  --max-files-with-diagnostics "$KCG_JS_MAX_FILES_WITH_DIAGNOSTICS" \
  --max-total-diagnostics "$KCG_JS_MAX_TOTAL_DIAGNOSTICS" \
  --out "$OUT_DIR/kotlin-js-acceptance-lenient.json"

echo
echo "== kotlin_js_resolve_lenient =="
go run ./cmd/kotlin-compiler-golang resolve \
  --repo "$KOTLIN_JS_SRC" \
  "${kcg_parser_backend_args[@]}" \
  --inventory "$INVENTORY_PATH" \
  "${kcg_timeout_args[@]}" \
  --lenient \
  --json > "$OUT_DIR/kotlin-js-resolve-lenient.json"
echo "saved: $OUT_DIR/kotlin-js-resolve-lenient.json"

if [[ "$RUN_KOTLIN_OFFICIAL_PARITY" -eq 1 ]]; then
  run_and_log "kotlin_official_parity_js" \
    env \
      MAX_MISSING_IN_GO="$KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_GO" \
      MAX_MISSING_IN_OFFICIAL="$KOTLIN_OFFICIAL_PARITY_MAX_MISSING_IN_OFFICIAL" \
      MAX_PARSE_STATUS_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH" \
      MAX_PACKAGE_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_PACKAGE_MISMATCH" \
      MAX_IMPORT_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_IMPORT_MISMATCH" \
      MAX_DECLARATION_MISMATCH="$KOTLIN_OFFICIAL_PARITY_MAX_DECLARATION_MISMATCH" \
      "$ROOT_DIR/scripts/kcg_official_parity.sh" \
      "$KOTLIN_JS_SRC" \
      "$OUT_DIR" \
      "$OUT_DIR/kotlin-official-parity.json"
fi

if [[ "$RUN_KOTLIN_JVM" -eq 1 ]]; then
  echo
  echo "== kotlin_jvm_acceptance_lenient =="
  go run ./cmd/kotlin-compiler-golang acceptance \
    --repo "$KOTLIN_JVM_SRC" \
    "${kcg_parser_backend_args[@]}" \
    --inventory "$INVENTORY_PATH" \
    "${kcg_timeout_args[@]}" \
    --lenient \
    --max-failed-files 0 \
    --max-unresolved-imports 0 \
    --max-files-with-diagnostics "$KCG_JVM_MAX_FILES_WITH_DIAGNOSTICS" \
    --max-total-diagnostics "$KCG_JVM_MAX_TOTAL_DIAGNOSTICS" \
    --out "$OUT_DIR/kotlin-jvm-acceptance-lenient.json"

  echo
  echo "== kotlin_jvm_resolve_lenient =="
  go run ./cmd/kotlin-compiler-golang resolve \
    --repo "$KOTLIN_JVM_SRC" \
    "${kcg_parser_backend_args[@]}" \
    --inventory "$INVENTORY_PATH" \
    "${kcg_timeout_args[@]}" \
    --lenient \
    --json > "$OUT_DIR/kotlin-jvm-resolve-lenient.json"
  echo "saved: $OUT_DIR/kotlin-jvm-resolve-lenient.json"
fi

if [[ "$RUN_KOTLIN_CORE" -eq 1 ]]; then
  echo
  echo "== kotlin_core_acceptance_lenient =="
  go run ./cmd/kotlin-compiler-golang acceptance \
    --repo "$KOTLIN_CORE_SRC" \
    "${kcg_parser_backend_args[@]}" \
    --inventory "$INVENTORY_PATH" \
    "${kcg_timeout_args[@]}" \
    --lenient \
    --max-failed-files 0 \
    --max-unresolved-imports 0 \
    --max-files-with-diagnostics "$KCG_CORE_MAX_FILES_WITH_DIAGNOSTICS" \
    --max-total-diagnostics "$KCG_CORE_MAX_TOTAL_DIAGNOSTICS" \
    --out "$OUT_DIR/kotlin-core-acceptance-lenient.json"

  echo
  echo "== kotlin_core_resolve_lenient =="
  go run ./cmd/kotlin-compiler-golang resolve \
    --repo "$KOTLIN_CORE_SRC" \
    "${kcg_parser_backend_args[@]}" \
    --inventory "$INVENTORY_PATH" \
    "${kcg_timeout_args[@]}" \
    --lenient \
    --json > "$OUT_DIR/kotlin-core-resolve-lenient.json"
  echo "saved: $OUT_DIR/kotlin-core-resolve-lenient.json"
fi

if [[ "$RUN_KOTLIN_CORE_MIXED_GRAPH" -eq 1 ]]; then
  run_and_log "kotlin_core_mixed_graph_lenient" \
    go run ./cmd/jkdeps graph \
      --repo "$KOTLIN_CORE_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --group-by package \
      --include-prefix "$KOTLIN_CORE_MIXED_INCLUDE_PREFIX" \
      "${kcg_mixed_graph_inventory_args[@]}" \
      "${kcg_mixed_graph_edge_args[@]}" \
      --lenient \
      --out "$OUT_DIR/kotlin-core-mixed-package"
fi

if [[ "$RUN_KOTLIN_CORE_MIXED_DIR_GRAPH" -eq 1 ]]; then
  run_and_log "kotlin_core_mixed_dir_graph_lenient" \
    go run ./cmd/jkdeps graph \
      --repo "$KOTLIN_CORE_SRC" \
      --java-grammar java20 \
      --workers "$WORKERS" \
      --group-by dir \
      "${kcg_dir_include_args[@]}" \
      "${kcg_mixed_graph_inventory_args[@]}" \
      "${kcg_mixed_graph_edge_args[@]}" \
      --lenient \
      --out "$OUT_DIR/kotlin-core-mixed-dir"
fi

run_and_log "kotlin_js_graph_lenient" \
  go run ./cmd/jkdeps graph \
    --repo "$KOTLIN_JS_SRC" \
    --java-grammar java20 \
    --group-by package \
    --inventory "$INVENTORY_PATH" \
    --lenient \
    --out "$OUT_DIR/kotlin-js-package"

echo
echo "== Artifacts =="
ls -1 "$OUT_DIR"
