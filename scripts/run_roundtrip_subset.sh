#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HELPER_SCRIPT="$ROOT_DIR/scripts/roundtrip_subset_helper.py"
TARGETS_FILE="${TARGETS_FILE:-/tmp/jkdeps-50-projects-out/selected-targets.csv}"
OUT_FILE="${OUT_FILE:-/tmp/jkdeps-roundtrip-subset-results.csv}"
WORKERS="${WORKERS:-1}"
JKDEPS_BIN="${JKDEPS_BIN:-}"

if [[ ! -f "$TARGETS_FILE" ]]; then
  echo "targets file missing: $TARGETS_FILE" >&2
  exit 1
fi

if [[ ! -f "$HELPER_SCRIPT" ]]; then
  echo "helper script missing: $HELPER_SCRIPT" >&2
  exit 1
fi

if [[ "$#" -eq 0 ]]; then
  set -- \
    "apache/httpcomponents-client" \
    "spring-projects/spring-security" \
    "Kotlin/kotlinx.serialization" \
    "cashapp/turbine" \
    "touchlab/Kermit"
fi

if [[ -z "$JKDEPS_BIN" ]]; then
  JKDEPS_BIN="$ROOT_DIR/.tmp/jkdeps-roundtrip-bin"
fi

if [[ ! -x "$JKDEPS_BIN" ]]; then
  mkdir -p "$(dirname "$JKDEPS_BIN")"
  (cd "$ROOT_DIR" && go build -o "$JKDEPS_BIN" ./cmd/jkdeps)
fi

echo "name,language,target_dir,status,pass,diff,parse_failed,unsupported,format_error" > "$OUT_FILE"

lookup_target() {
  local project="$1"
  python3 "$HELPER_SCRIPT" lookup-target "$TARGETS_FILE" "$project"
}

for project in "$@"; do
  echo "==> $project"
  if ! mapfile -t target_info < <(lookup_target "$project"); then
    echo "$project,,,,missing-target,0,0,0,0,0" >> "$OUT_FILE"
    echo "  missing target metadata"
    continue
  fi

  lang="${target_info[0]}"
  target_dir="${target_info[1]}"

  if [[ ! -d "$target_dir" ]]; then
    echo "$project,$lang,$target_dir,missing-dir,0,0,0,0,0" >> "$OUT_FILE"
    echo "  missing target directory"
    continue
  fi

  status="ok"
  set +e
  result_json="$("$JKDEPS_BIN" roundtrip-check --repo "$target_dir" --workers "$WORKERS" --json 2>/tmp/jkdeps-roundtrip-subset.stderr)"
  cmd_rc=$?
  set -e

  if ! metrics="$(python3 "$HELPER_SCRIPT" summarize-json "$result_json")"; then
    status="decode-error"
    echo "$project,$lang,$target_dir,$status,0,0,0,0,0" >> "$OUT_FILE"
    echo "  failed to decode roundtrip output"
    if [[ -s /tmp/jkdeps-roundtrip-subset.stderr ]]; then
      sed -n '1,5p' /tmp/jkdeps-roundtrip-subset.stderr
    fi
    continue
  fi

  IFS=, read -r pass diff parse_failed unsupported format_error <<<"$metrics"
  if [[ "$cmd_rc" -ne 0 ]]; then
    status="findings"
  fi
  echo "$project,$lang,$target_dir,$status,$pass,$diff,$parse_failed,$unsupported,$format_error" >> "$OUT_FILE"
  echo "  pass=$pass diff=$diff parse_failed=$parse_failed unsupported=$unsupported format_error=$format_error"
done

echo
echo "wrote $OUT_FILE"
