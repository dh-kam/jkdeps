#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-/tmp/jkdeps-porting}"
BASELINE_FILE="${2:-$ROOT_DIR/docs/porting-baseline.json}"
AUDIT_JSON="${3:-$OUT_DIR/porting-audit.json}"
VERIFY_JSON="$OUT_DIR/porting-completion-verify.json"
COMPARE_JSON="$OUT_DIR/porting-baseline-compare.json"
SYNC_LOG="$OUT_DIR/porting-baseline-sync.log"

status="PASSED"
failed_stage=""
sync_status="NOT_RUN"

write_audit_json() {
  AUDIT_STATUS="$status" \
  AUDIT_FAILED_STAGE="$failed_stage" \
  AUDIT_OUT_DIR="$OUT_DIR" \
  AUDIT_BASELINE_FILE="$BASELINE_FILE" \
  AUDIT_VERIFY_JSON="$VERIFY_JSON" \
  AUDIT_COMPARE_JSON="$COMPARE_JSON" \
  AUDIT_SYNC_LOG="$SYNC_LOG" \
  AUDIT_SYNC_STATUS="$sync_status" \
  AUDIT_JSON_PATH="$AUDIT_JSON" \
  python3 - <<'PY'
import datetime
import json
import os
from pathlib import Path

def load_json(path: str):
    p = Path(path)
    if not p.is_file():
        return None
    try:
        return json.loads(p.read_text(encoding="utf-8"))
    except Exception:
        return None

payload = {
    "status": os.environ.get("AUDIT_STATUS", ""),
    "failed_stage": os.environ.get("AUDIT_FAILED_STAGE", ""),
    "generated_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "out_dir": os.environ.get("AUDIT_OUT_DIR", ""),
    "baseline_file": os.environ.get("AUDIT_BASELINE_FILE", ""),
    "artifacts": {
        "verify_json": os.environ.get("AUDIT_VERIFY_JSON", ""),
        "compare_json": os.environ.get("AUDIT_COMPARE_JSON", ""),
        "sync_log": os.environ.get("AUDIT_SYNC_LOG", ""),
    },
    "sync_check": {
        "status": os.environ.get("AUDIT_SYNC_STATUS", ""),
    },
}

payload["verify"] = load_json(payload["artifacts"]["verify_json"])
payload["compare"] = load_json(payload["artifacts"]["compare_json"])

sync_log = Path(payload["artifacts"]["sync_log"])
if sync_log.is_file():
    payload["sync_check"]["log_tail"] = sync_log.read_text(encoding="utf-8", errors="replace").strip().splitlines()[-20:]

out_path = Path(os.environ["AUDIT_JSON_PATH"])
out_path.parent.mkdir(parents=True, exist_ok=True)
out_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY
}

if ! "$ROOT_DIR/scripts/verify_porting_completion.sh" "$OUT_DIR" --json-out "$VERIFY_JSON"; then
  status="FAILED"
  failed_stage="verify"
  write_audit_json
  exit 1
fi

if ! "$ROOT_DIR/scripts/compare_porting_baseline.sh" "$OUT_DIR" "$BASELINE_FILE"; then
  status="FAILED"
  failed_stage="compare"
  write_audit_json
  exit 1
fi

if "$ROOT_DIR/scripts/update_porting_baseline.sh" "$OUT_DIR" "$BASELINE_FILE" --check >"$SYNC_LOG" 2>&1; then
  sync_status="PASSED"
else
  sync_status="FAILED"
  status="FAILED"
  failed_stage="baseline_sync"
  cat "$SYNC_LOG" >&2 || true
  write_audit_json
  exit 1
fi

write_audit_json
echo "porting audit PASSED: $AUDIT_JSON"
