#!/usr/bin/env bash
set -euo pipefail

event_name="${EVENT_NAME:-pull_request}"
base_sha="${BASE_SHA:-}"
head_ref="${HEAD_REF:-HEAD}"
labels_csv="${PR_LABELS:-}"
required_label="${BASELINE_APPROVAL_LABEL:-baseline-approved}"
baseline_file="${BASELINE_FILE_PATH:-docs/porting-baseline.json}"
changed_files_override="${CHANGED_FILES:-}"
json_stdout=0
json_out_path="${JSON_OUT_PATH:-}"
baseline_changed=0
used_changed_files_override=0

emit_json() {
  local status="$1"
  local reason="$2"
  local message="$3"

  JSON_STATUS="$status" \
  JSON_REASON="$reason" \
  JSON_MESSAGE="$message" \
  JSON_EVENT_NAME="$event_name" \
  JSON_BASE_SHA="$base_sha" \
  JSON_HEAD_REF="$head_ref" \
  JSON_BASELINE_FILE="$baseline_file" \
  JSON_REQUIRED_LABEL="$required_label" \
  JSON_LABELS_CSV="$labels_csv" \
  JSON_BASELINE_CHANGED="$baseline_changed" \
  JSON_USED_CHANGED_FILES_OVERRIDE="$used_changed_files_override" \
  JSON_STDOUT="$json_stdout" \
  JSON_OUT_PATH="$json_out_path" \
  python3 - <<'PY'
import json
import os
from pathlib import Path

def as_bool(value: str) -> bool:
    return str(value).lower() in ("1", "true", "yes")

payload = {
    "status": os.environ.get("JSON_STATUS", ""),
    "reason": os.environ.get("JSON_REASON", ""),
    "message": os.environ.get("JSON_MESSAGE", ""),
    "event_name": os.environ.get("JSON_EVENT_NAME", ""),
    "base_sha": os.environ.get("JSON_BASE_SHA", ""),
    "head_ref": os.environ.get("JSON_HEAD_REF", ""),
    "baseline_file": os.environ.get("JSON_BASELINE_FILE", ""),
    "required_label": os.environ.get("JSON_REQUIRED_LABEL", ""),
    "labels_csv": os.environ.get("JSON_LABELS_CSV", ""),
    "baseline_changed": as_bool(os.environ.get("JSON_BASELINE_CHANGED", "0")),
    "used_changed_files_override": as_bool(os.environ.get("JSON_USED_CHANGED_FILES_OVERRIDE", "0")),
}

text = json.dumps(payload, ensure_ascii=True)
if as_bool(os.environ.get("JSON_STDOUT", "0")):
    print(text)

out_path = os.environ.get("JSON_OUT_PATH", "")
if out_path:
    path = Path(out_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text + "\n", encoding="utf-8")
PY
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --event-name)
      event_name="$2"
      shift 2
      ;;
    --base-sha)
      base_sha="$2"
      shift 2
      ;;
    --head-ref)
      head_ref="$2"
      shift 2
      ;;
    --labels)
      labels_csv="$2"
      shift 2
      ;;
    --required-label)
      required_label="$2"
      shift 2
      ;;
    --baseline-file)
      baseline_file="$2"
      shift 2
      ;;
    --changed-files)
      changed_files_override="$2"
      shift 2
      ;;
    --json)
      json_stdout=1
      shift
      ;;
    --json-out)
      json_out_path="$2"
      shift 2
      ;;
    -h|--help)
      cat <<'EOF'
Usage:
  check_baseline_label_gate.sh [options]

Options:
  --event-name <name>         Event name (default: pull_request)
  --base-sha <sha>            Base commit SHA for diff
  --head-ref <ref>            Head ref/sha for diff (default: HEAD)
  --labels <csv>              PR labels CSV (default from PR_LABELS)
  --required-label <label>    Required approval label (default: baseline-approved)
  --baseline-file <path>      Baseline file path to watch (default: docs/porting-baseline.json)
  --changed-files <csv>       Override changed files list (comma-separated), skip git diff
  --json                      Print gate result JSON to stdout
  --json-out <path>           Write gate result JSON to this path

Env fallback:
  EVENT_NAME, BASE_SHA, HEAD_REF, PR_LABELS, BASELINE_APPROVAL_LABEL, BASELINE_FILE_PATH, CHANGED_FILES, JSON_OUT_PATH
EOF
      exit 0
      ;;
    *)
      msg="unknown argument: $1"
      echo "$msg" >&2
      emit_json "error" "unknown_argument" "$msg"
      exit 2
      ;;
  esac
done

if [[ "$event_name" != "pull_request" ]]; then
  msg="Baseline label gate skipped: event_name=$event_name"
  echo "$msg"
  emit_json "skipped" "event_not_pull_request" "$msg"
  exit 0
fi

if [[ -n "$changed_files_override" ]]; then
  used_changed_files_override=1
  changed_files_normalized="${changed_files_override//$'\n'/,}"
  IFS=',' read -r -a changed_files <<< "$changed_files_normalized"
  for changed in "${changed_files[@]}"; do
    trimmed="$(echo "$changed" | xargs)"
    if [[ "$trimmed" == "$baseline_file" ]]; then
      baseline_changed=1
      break
    fi
  done
else
  if [[ -z "$base_sha" ]]; then
    msg="BASE_SHA is required for pull_request when --changed-files is not provided"
    echo "$msg" >&2
    emit_json "error" "missing_base_sha" "$msg"
    exit 2
  fi

  if ! git cat-file -e "${base_sha}^{commit}" >/dev/null 2>&1; then
    git fetch --no-tags --prune --depth=1 origin "$base_sha"
  fi

  if git diff --name-only "$base_sha" "$head_ref" | grep -Fqx "$baseline_file"; then
    baseline_changed=1
  fi
fi

if [[ "$baseline_changed" -eq 0 ]]; then
  msg="Baseline label gate skipped: ${baseline_file} not changed"
  echo "$msg"
  emit_json "skipped" "baseline_file_not_changed" "$msg"
  exit 0
fi

echo "${baseline_file} was changed in this PR."
echo "PR labels: ${labels_csv}"
echo "Required label: ${required_label}"

if [[ ",${labels_csv}," != *",${required_label},"* ]]; then
  msg="Missing required PR label: ${required_label}"
  echo "$msg" >&2
  emit_json "failed" "missing_required_label" "$msg"
  exit 1
fi

msg="Baseline label gate satisfied."
echo "$msg"
emit_json "passed" "approved" "$msg"
