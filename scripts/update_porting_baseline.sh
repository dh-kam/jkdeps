#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="/tmp/jkdeps-porting"
BASELINE_FILE="$ROOT_DIR/docs/porting-baseline.json"
CHECK_MODE=0
ARTIFACT_DIR_OVERRIDE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check)
      CHECK_MODE=1
      shift
      ;;
    --artifact-dir)
      ARTIFACT_DIR_OVERRIDE="$2"
      shift 2
      ;;
    -h|--help)
      cat <<'EOF'
Usage:
  update_porting_baseline.sh [out-dir] [baseline-file] [--check] [--artifact-dir <value>]

Examples:
  ./scripts/update_porting_baseline.sh /tmp/jkdeps-porting ./docs/porting-baseline.json
  ./scripts/update_porting_baseline.sh /tmp/jkdeps-porting ./docs/porting-baseline.json --check
  ./scripts/update_porting_baseline.sh /tmp/jkdeps-porting ./docs/porting-baseline.json --artifact-dir /tmp/jkdeps-porting-final13

Notes:
  - BASELINE_DATE env can override baseline_date.
  - Without BASELINE_DATE, existing baseline_date is preserved when possible.
EOF
      exit 0
      ;;
    --*)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
    *)
      if [[ "$OUT_DIR" == "/tmp/jkdeps-porting" ]]; then
        OUT_DIR="$1"
      elif [[ "$BASELINE_FILE" == "$ROOT_DIR/docs/porting-baseline.json" ]]; then
        BASELINE_FILE="$1"
      else
        echo "unexpected extra positional argument: $1" >&2
        exit 2
      fi
      shift
      ;;
  esac
done

python3 - "$OUT_DIR" "$BASELINE_FILE" "$CHECK_MODE" "$ARTIFACT_DIR_OVERRIDE" <<'PY'
import datetime
import json
import os
import re
import sys
from pathlib import Path

out_dir = Path(sys.argv[1])
baseline_path = Path(sys.argv[2])
check_mode = sys.argv[3] == "1"
artifact_dir_override = sys.argv[4]

acceptance_files = [
    "kotlin-common-acceptance.json",
    "kotlin-js-acceptance-lenient.json",
    "kotlin-jvm-acceptance-lenient.json",
    "kotlin-core-acceptance-lenient.json",
]


def load_json(path: Path, label: str):
    if not path.is_file():
        raise SystemExit(f"missing {label}: {path}")
    try:
        with path.open() as f:
            return json.load(f)
    except Exception as exc:
        raise SystemExit(f"invalid {label}: {path}: {exc}")


def as_int(doc: dict, path: tuple[str, ...], label: str) -> int:
    cur = doc
    for key in path:
        if not isinstance(cur, dict):
            raise SystemExit(f"{label}: missing field {'.'.join(path)}")
        cur = cur.get(key)
    if not isinstance(cur, int):
        raise SystemExit(f"{label}: field {'.'.join(path)} must be int")
    return cur


def matches_hex(value: str, length: int) -> bool:
    return isinstance(value, str) and re.fullmatch(rf"[0-9a-f]{{{length}}}", value) is not None


existing = {}
existing_load_error = ""
if baseline_path.is_file():
    try:
        with baseline_path.open() as f:
            loaded = json.load(f)
            if isinstance(loaded, dict):
                existing = loaded
            else:
                existing_load_error = "baseline root must be an object"
    except Exception as exc:
        existing_load_error = str(exc)

if existing_load_error:
    raise SystemExit(f"invalid existing baseline: {baseline_path}: {existing_load_error}")

acceptance = {}
existing_acceptance = existing.get("acceptance", {})
if not isinstance(existing_acceptance, dict):
    existing_acceptance = {}

for rel in acceptance_files:
    doc = load_json(out_dir / rel, f"acceptance report ({rel})")
    entry = {
        "failed_files": as_int(doc, ("parse", "failed_files"), rel),
        "unresolved_imports": as_int(doc, ("resolve", "unresolved_imports"), rel),
        "unknown_nodes": as_int(doc, ("graph", "unknown_nodes"), rel),
        "files_with_diagnostics": as_int(doc, ("parse", "files_with_diagnostics"), rel),
        "total_diagnostics": as_int(doc, ("parse", "total_diagnostics"), rel),
        "max_regression_files_with_diagnostics": 0,
        "max_regression_total_diagnostics": 0,
    }
    old = existing_acceptance.get(rel, {})
    if isinstance(old, dict):
        if isinstance(old.get("max_regression_files_with_diagnostics"), int):
            entry["max_regression_files_with_diagnostics"] = old["max_regression_files_with_diagnostics"]
        if isinstance(old.get("max_regression_total_diagnostics"), int):
            entry["max_regression_total_diagnostics"] = old["max_regression_total_diagnostics"]
    acceptance[rel] = entry

metadata = load_json(out_dir / "porting-run-metadata.json", "run metadata")
sample_refs = metadata.get("sample_refs")
if not isinstance(sample_refs, dict):
    raise SystemExit("invalid run metadata: sample_refs missing or invalid")
guava_ref = sample_refs.get("guava")
coroutines_ref = sample_refs.get("kotlinx.coroutines")
if not matches_hex(guava_ref, 40):
    raise SystemExit("invalid run metadata: sample_refs.guava must be 40-char git sha")
if not matches_hex(coroutines_ref, 40):
    raise SystemExit("invalid run metadata: sample_refs.kotlinx.coroutines must be 40-char git sha")

inventory = metadata.get("inventory")
if not isinstance(inventory, dict):
    raise SystemExit("invalid run metadata: inventory missing or invalid")
inventory_sha = inventory.get("sha256")
if not matches_hex(inventory_sha, 64):
    raise SystemExit("invalid run metadata: inventory.sha256 must be 64-char sha256")

mixed_graph = existing.get("mixed_graph")
if not isinstance(mixed_graph, dict):
    mixed_graph = {
        "log_file": "kotlin_core_mixed_graph_lenient.log",
        "min_java_files": 1,
        "min_kotlin_files": 1,
        "require_failed_zero": True,
    }

mixed_dir_graph = existing.get("mixed_dir_graph")
if not isinstance(mixed_dir_graph, dict):
    mixed_dir_graph = {}

baseline_date = os.environ.get("BASELINE_DATE", "").strip()
if not baseline_date:
    existing_baseline_date = existing.get("baseline_date")
    if isinstance(existing_baseline_date, str) and existing_baseline_date:
        baseline_date = existing_baseline_date
    else:
        baseline_date = datetime.date.today().isoformat()

artifact_dir_value = artifact_dir_override.strip()
if not artifact_dir_value:
    existing_artifact_dir = existing.get("artifact_dir")
    if isinstance(existing_artifact_dir, str) and existing_artifact_dir:
        artifact_dir_value = existing_artifact_dir
    else:
        artifact_dir_value = str(out_dir)

payload = {
    "baseline_date": baseline_date,
    "artifact_dir": artifact_dir_value,
    "sample_refs": {
        "guava": guava_ref,
        "kotlinx.coroutines": coroutines_ref,
    },
    "runtime_inventory": {
        "sha256": inventory_sha,
    },
    "acceptance": acceptance,
    "mixed_graph": mixed_graph,
}

if mixed_dir_graph:
    payload["mixed_dir_graph"] = mixed_dir_graph

if check_mode:
    if not baseline_path.is_file():
        raise SystemExit(f"baseline file does not exist for check mode: {baseline_path}")
    if existing != payload:
        generated_path = Path(str(baseline_path) + ".generated")
        generated_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
        raise SystemExit(
            f"baseline is out of date: {baseline_path}\n"
            f"generated candidate written to: {generated_path}\n"
            "run update_porting_baseline.sh without --check to refresh"
        )
    print(f"baseline is up to date: {baseline_path}")
else:
    baseline_path.parent.mkdir(parents=True, exist_ok=True)
    baseline_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"updated baseline: {baseline_path}")
PY
