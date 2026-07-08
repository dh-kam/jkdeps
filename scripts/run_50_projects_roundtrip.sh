#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-/tmp/jkdeps-50-projects-out}"
TARGETS_FILE="${TARGETS_FILE:-$OUT_DIR/selected-targets.csv}"
OUT_FILE="${OUT_FILE:-$OUT_DIR/roundtrip-summary.csv}"
WORKERS="${WORKERS:-$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)}"
MAX_ERRORS="${MAX_ERRORS:-20}"
PROJECT_LIMIT="${PROJECT_LIMIT:-0}"
PROJECT_TIMEOUT="${PROJECT_TIMEOUT:-0}"
ROUNDTRIP_RESUME="${ROUNDTRIP_RESUME:-1}"
REWRITE_MODE="${REWRITE_MODE:-lossless}"
JKDEPS_BIN="${JKDEPS_BIN:-$ROOT_DIR/.tmp/jkdeps-roundtrip-bin}"
JAVA_FORMAT_CMD="${JAVA_FORMAT_CMD:-}"
KOTLIN_FORMAT_CMD="${KOTLIN_FORMAT_CMD:-}"

if [[ ! -f "$TARGETS_FILE" ]]; then
  echo "targets file missing: $TARGETS_FILE" >&2
  exit 1
fi

mkdir -p "$(dirname "$JKDEPS_BIN")" "$OUT_DIR"
if [[ ! -x "$JKDEPS_BIN" ]]; then
  go build -o "$JKDEPS_BIN" ./cmd/jkdeps
fi

if [[ "$ROUNDTRIP_RESUME" == "1" && -f "$OUT_FILE" ]]; then
  :
else
  echo "status,name,language,total_files,checked_files,passed_files,diff_files,parse_failed_files,unsupported_files,format_error_files,exact_rate,duration_seconds,parse_duration_seconds,rewrite_mode,java_format_cmd,kotlin_format_cmd,target_dir" > "$OUT_FILE"
fi

summary_has_entry() {
  local name="$1"
  [[ -f "$OUT_FILE" ]] || return 1
  rg -q "^[^,]+,${name//\//\\/}," "$OUT_FILE"
}

json_field() {
  local payload="$1"
  local field="$2"
  python3 -c 'import json,sys; data=json.load(sys.stdin); print(data.get(sys.argv[1], ""))' "$field" <<<"$payload"
}

append_result() {
  local status="$1"
  local name="$2"
  local language="$3"
  local payload="$4"
  local target_dir="$5"
  local total checked passed diff parse_failed unsupported format_error exact duration parse_duration rewrite_mode java_cmd kotlin_cmd

  total="$(json_field "$payload" total_files)"
  checked="$(json_field "$payload" checked_files)"
  passed="$(json_field "$payload" passed_files)"
  diff="$(json_field "$payload" diff_files)"
  parse_failed="$(json_field "$payload" parse_failed_files)"
  unsupported="$(json_field "$payload" unsupported_files)"
  format_error="$(json_field "$payload" format_error_files)"
  duration="$(json_field "$payload" duration_seconds)"
  parse_duration="$(json_field "$payload" parse_duration_seconds)"
  rewrite_mode="$(json_field "$payload" rewrite_mode)"
  java_cmd="$(json_field "$payload" java_format_cmd)"
  kotlin_cmd="$(json_field "$payload" kotlin_format_cmd)"
  if [[ -z "$total" || "$total" == "0" ]]; then
    exact="0.0000"
  else
    exact="$(python3 -c 'import sys; print(f"{(int(sys.argv[1]) / int(sys.argv[2])) * 100:.4f}")' "$passed" "$total")"
  fi
  java_cmd="${java_cmd//,/;}"
  kotlin_cmd="${kotlin_cmd//,/;}"
  target_dir="${target_dir//,/;}"
  echo "$status,$name,$language,$total,$checked,$passed,$diff,$parse_failed,$unsupported,$format_error,$exact,$duration,$parse_duration,$rewrite_mode,$java_cmd,$kotlin_cmd,$target_dir" >> "$OUT_FILE"
}

scanned=0
{
  IFS= read -r header || true
  while IFS=, read -r name language repo_dir target_dir selection_mode relative_target include_kts java_grammar rest; do
    [[ -z "$name" ]] && continue
    scanned=$((scanned + 1))
    if [[ "$PROJECT_LIMIT" != "0" && "$scanned" -gt "$PROJECT_LIMIT" ]]; then
      break
    fi
    if [[ "$ROUNDTRIP_RESUME" == "1" ]] && summary_has_entry "$name"; then
      echo "[skip] $name already present in roundtrip summary"
      continue
    fi
    include_kts="${include_kts:-true}"
    java_grammar="${java_grammar:-java20}"
    if [[ ! -d "$target_dir" ]]; then
      echo "missing-dir,$name,$language,0,0,0,0,0,0,0,0.0000,0,0,$REWRITE_MODE,,,${target_dir//,/;}" >> "$OUT_FILE"
      echo "[missing-dir] $name $target_dir"
      continue
    fi

    cmd=("$JKDEPS_BIN" roundtrip-check
      --repo "$target_dir"
      --java-grammar "$java_grammar"
      --workers "$WORKERS"
      --max-errors-per-file "$MAX_ERRORS"
      --include-kts="$include_kts"
      --rewrite-mode "$REWRITE_MODE"
      --json)
    if [[ -n "$JAVA_FORMAT_CMD" ]]; then
      cmd+=(--java-format-cmd "$JAVA_FORMAT_CMD")
    fi
    if [[ -n "$KOTLIN_FORMAT_CMD" ]]; then
      cmd+=(--kotlin-format-cmd "$KOTLIN_FORMAT_CMD")
    fi
    if [[ "$PROJECT_TIMEOUT" != "0" && "$PROJECT_TIMEOUT" != "" ]]; then
      cmd=(timeout "$PROJECT_TIMEOUT" "${cmd[@]}")
    fi

    echo "==> $name ($language)"
    stderr_file="$OUT_DIR/$(echo "$name" | tr -c 'A-Za-z0-9._-' '_').roundtrip.stderr"
    set +e
    payload="$("${cmd[@]}" 2>"$stderr_file")"
    rc=$?
    set -e

    if ! python3 -c 'import json,sys; json.load(sys.stdin)' <<<"$payload" 2>/dev/null; then
      if [[ "$PROJECT_TIMEOUT" != "0" && ( "$rc" -eq 124 || "$rc" -eq 137 ) ]]; then
        status="timeout"
      else
        status="runtime-error"
      fi
      echo "$status,$name,$language,0,0,0,0,0,0,0,0.0000,0,0,$REWRITE_MODE,,,${target_dir//,/;}" >> "$OUT_FILE"
      sed -n '1,5p' "$stderr_file" >&2 || true
      continue
    fi

    findings="$(python3 -c 'import json,sys; data=json.load(sys.stdin); print(sum(int(data.get(k,0)) for k in ("diff_files","parse_failed_files","unsupported_files","format_error_files")))' <<<"$payload")"
    if [[ "$rc" -eq 0 && "$findings" == "0" ]]; then
      status="ok"
    else
      status="findings"
    fi
    append_result "$status" "$name" "$language" "$payload" "$target_dir"
    echo "[$status] $name pass=$(json_field "$payload" passed_files) total=$(json_field "$payload" total_files) exact=$(tail -n1 "$OUT_FILE" | cut -d, -f11)%"
  done
} < "$TARGETS_FILE"

echo
echo "Round-trip summary: $OUT_FILE"
