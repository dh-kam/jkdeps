#!/usr/bin/env bash
set -euo pipefail

out_dir="/tmp/jkdeps-porting"
json_stdout=0
json_out_path=""

while [[ $# -gt 0 ]]; do
  case "$1" in
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
  verify_porting_completion.sh [out-dir] [--json] [--json-out <path>]

Options:
  --json              Print machine-readable result JSON to stdout
  --json-out <path>   Write machine-readable result JSON to a file
EOF
      exit 0
      ;;
    --*)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
    *)
      if [[ "$out_dir" != "/tmp/jkdeps-porting" ]]; then
        echo "unexpected extra positional argument: $1" >&2
        exit 2
      fi
      out_dir="$1"
      shift
      ;;
  esac
done

python3 - "$out_dir" "$json_stdout" "$json_out_path" <<'PY'
import json
import re
import sys
from pathlib import Path

out_dir = Path(sys.argv[1])
json_stdout = sys.argv[2] == "1"
json_out_path = sys.argv[3]
errors = []
checks = []
result = {
    "status": "FAILED",
    "out_dir": str(out_dir),
    "checks": [],
    "errors": [],
}


def emit_result(status: str) -> None:
    result["status"] = status
    result["checks"] = checks
    result["errors"] = errors
    text = json.dumps(result, ensure_ascii=True)
    if json_stdout:
        print(text)
    if json_out_path:
        path = Path(json_out_path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text + "\n", encoding="utf-8")


def matches_hex(value, length: int) -> bool:
    return isinstance(value, str) and re.fullmatch(rf"[0-9a-f]{{{length}}}", value) is not None


def parse_truthy(value) -> bool:
    return str(value).lower() in {"1", "true", "yes", "y"}


def require_file(path: Path) -> bool:
    if not path.is_file():
        errors.append(f"missing artifact: {path}")
        return False
    return True


def load_json(path: Path):
    if not require_file(path):
        return None
    try:
        with path.open() as f:
            return json.load(f)
    except Exception as exc:
        errors.append(f"invalid json: {path}: {exc}")
        return None


acceptance_files = {
    "kotlin-common-acceptance.json": ("kotlin-common-acceptance.json", 10, 70),
    "kotlin-js-acceptance-lenient.json": ("kotlin-js-acceptance-lenient.json", 2, 10),
    "kotlin-jvm-acceptance-lenient.json": ("kotlin-jvm-acceptance-lenient.json", 8, 40),
    "kotlin-core-acceptance-lenient.json": ("kotlin-core-acceptance-lenient.json", 35, 220),
}

for label, (rel, max_diag_files, max_total_diag) in acceptance_files.items():
    path = out_dir / rel
    doc = load_json(path)
    if doc is None:
        continue
    parse_failed = doc.get("parse", {}).get("failed_files")
    unresolved = doc.get("resolve", {}).get("unresolved_imports")
    unknown_nodes = doc.get("graph", {}).get("unknown_nodes")
    diag_files = doc.get("parse", {}).get("files_with_diagnostics")
    total_diag = doc.get("parse", {}).get("total_diagnostics")
    if parse_failed != 0:
        errors.append(f"{label}: parse.failed_files expected 0, got {parse_failed}")
    if unresolved != 0:
        errors.append(f"{label}: resolve.unresolved_imports expected 0, got {unresolved}")
    if unknown_nodes != 0:
        errors.append(f"{label}: graph.unknown_nodes expected 0, got {unknown_nodes}")
    if not isinstance(diag_files, int):
        errors.append(f"{label}: parse.files_with_diagnostics missing or invalid")
    elif diag_files > max_diag_files:
        errors.append(
            f"{label}: parse.files_with_diagnostics expected <= {max_diag_files}, got {diag_files}"
        )
    if not isinstance(total_diag, int):
        errors.append(f"{label}: parse.total_diagnostics missing or invalid")
    elif total_diag > max_total_diag:
        errors.append(
            f"{label}: parse.total_diagnostics expected <= {max_total_diag}, got {total_diag}"
        )
    checks.append(
        f"{label}: parse_failed={parse_failed} unresolved={unresolved} unknown_nodes={unknown_nodes} "
        f"diag_files={diag_files} total_diag={total_diag}"
    )

resolve_files = {
    "kotlin-common-resolve.json": "kotlin-common-resolve.json",
    "kotlin-js-resolve-lenient.json": "kotlin-js-resolve-lenient.json",
    "kotlin-jvm-resolve-lenient.json": "kotlin-jvm-resolve-lenient.json",
    "kotlin-core-resolve-lenient.json": "kotlin-core-resolve-lenient.json",
}

for label, rel in resolve_files.items():
    path = out_dir / rel
    doc = load_json(path)
    if doc is None:
        continue
    unresolved = doc.get("unresolved_imports")
    if unresolved != 0:
        errors.append(f"{label}: unresolved_imports expected 0, got {unresolved}")
    checks.append(f"{label}: unresolved_imports={unresolved}")

metadata_rel = "porting-run-metadata.json"
metadata = load_json(out_dir / metadata_rel)
run_flags = {}
if metadata is not None:
    sample_refs = metadata.get("sample_refs")
    if not isinstance(sample_refs, dict):
        errors.append(f"{metadata_rel}: sample_refs missing or invalid")
    else:
        for sample_name in ("guava", "kotlinx.coroutines"):
            value = sample_refs.get(sample_name)
            if not matches_hex(value, 40):
                errors.append(f"{metadata_rel}: sample_refs.{sample_name} must be 40-char git sha")
            else:
                checks.append(f"{metadata_rel}: sample_refs.{sample_name}={value}")

    inventory = metadata.get("inventory")
    if not isinstance(inventory, dict):
        errors.append(f"{metadata_rel}: inventory missing or invalid")
    else:
        inv_path = inventory.get("path")
        inv_sha = inventory.get("sha256")
        if not isinstance(inv_path, str) or not inv_path:
            errors.append(f"{metadata_rel}: inventory.path missing or invalid")
        else:
            checks.append(f"{metadata_rel}: inventory.path={inv_path}")
        if not matches_hex(inv_sha, 64):
            errors.append(f"{metadata_rel}: inventory.sha256 must be 64-char sha256")
        else:
            checks.append(f"{metadata_rel}: inventory.sha256={inv_sha}")

    run_flags = metadata.get("run_flags")
    if run_flags is None:
        run_flags = {}
    if not isinstance(run_flags, dict):
        errors.append(f"{metadata_rel}: run_flags missing or invalid")
        run_flags = {}

run_kotlin_core_mixed_graph = parse_truthy(run_flags.get("run_kotlin_core_mixed_graph", "1"))
run_kotlin_core_mixed_dir_graph = parse_truthy(
    run_flags.get("run_kotlin_core_mixed_dir_graph", "1" if run_kotlin_core_mixed_graph else "0")
)

parity_requested = str(run_flags.get("run_kotlin_official_parity", "0")).lower() in {"1", "true", "yes"}
if parity_requested:
    parity_rel = "kotlin-official-parity.json"
    parity_doc = load_json(out_dir / parity_rel)
    if parity_doc is not None:
        parity_status = parity_doc.get("status")
        if parity_status != "PASSED":
            errors.append(f"{parity_rel}: status expected PASSED, got {parity_status}")

        summary = parity_doc.get("summary")
        thresholds = parity_doc.get("thresholds")
        if not isinstance(summary, dict):
            errors.append(f"{parity_rel}: summary missing or invalid")
        if not isinstance(thresholds, dict):
            errors.append(f"{parity_rel}: thresholds missing or invalid")
        if isinstance(summary, dict) and isinstance(thresholds, dict):
            checks_to_thresholds = (
                ("missing_in_go_files", "max_missing_in_go"),
                ("missing_in_official_files", "max_missing_in_official"),
                ("parse_status_mismatch_files", "max_parse_status_mismatch"),
                ("package_mismatch_files", "max_package_mismatch"),
                ("import_mismatch_files", "max_import_mismatch"),
                ("declaration_mismatch_files", "max_declaration_mismatch"),
            )
            for field, threshold_key in checks_to_thresholds:
                value = summary.get(field)
                threshold = thresholds.get(threshold_key)
                if not isinstance(value, int):
                    errors.append(f"{parity_rel}: summary.{field} missing or invalid")
                    continue
                if not isinstance(threshold, int):
                    errors.append(f"{parity_rel}: thresholds.{threshold_key} missing or invalid")
                    continue
                if value > threshold:
                    errors.append(
                        f"{parity_rel}: summary.{field} expected <= {threshold} "
                        f"(thresholds.{threshold_key}), got {value}"
                    )
            checks.append(
                f"{parity_rel}: missing_in_go={summary.get('missing_in_go_files')} "
                f"(<= {thresholds.get('max_missing_in_go')}) "
                f"missing_in_official={summary.get('missing_in_official_files')} "
                f"(<= {thresholds.get('max_missing_in_official')}) "
                f"parse_mismatch={summary.get('parse_status_mismatch_files')} "
                f"(<= {thresholds.get('max_parse_status_mismatch')}) "
                f"package_mismatch={summary.get('package_mismatch_files')} "
                f"(<= {thresholds.get('max_package_mismatch')}) "
                f"import_mismatch={summary.get('import_mismatch_files')} "
                f"(<= {thresholds.get('max_import_mismatch')}) "
                f"declaration_mismatch={summary.get('declaration_mismatch_files')} "
                f"(<= {thresholds.get('max_declaration_mismatch')})"
            )

log_rules = [
    ("guava_smoke_parse.log", r"Result:\s+parsed=\d+\s+failed=0", "strict parse gate"),
    ("guava_graph_filtered.log", r"ParseStatus:\s+parsed=\d+\s+failed=0", "strict parse gate"),
    ("guava_stress_smoke_strict.log", r"Result:\s+parsed=\d+\s+failed=0", "strict parse gate"),
    ("guava_stress_graph_strict.log", r"ParseStatus:\s+parsed=\d+\s+failed=0", "strict parse gate"),
]

if run_kotlin_core_mixed_graph:
    log_rules.append(
        ("kotlin_core_mixed_graph_lenient.log", r"ParseStatus:\s+parsed=\d+\s+failed=0", "strict parse gate")
    )
    log_rules.append(
        (
            "kotlin_core_mixed_graph_lenient.log",
            r"Files:\s+total=\d+\s+java=[1-9]\d*\s+kotlin=[1-9]\d*",
            "mixed java+kotlin gate",
        )
    )

if run_kotlin_core_mixed_dir_graph:
    log_rules.append(
        ("kotlin_core_mixed_dir_graph_lenient.log", r"ParseStatus:\s+parsed=\d+\s+failed=0", "strict parse gate")
    )
    log_rules.append(
        (
            "kotlin_core_mixed_dir_graph_lenient.log",
            r"Files:\s+total=\d+\s+java=[1-9]\d*\s+kotlin=[1-9]\d*",
            "mixed java+kotlin gate",
        )
    )

for rel, pattern, label in log_rules:
    path = out_dir / rel
    if not require_file(path):
        continue
    text = path.read_text(errors="replace")
    if re.search(pattern, text) is None:
        errors.append(f"{rel}: {label} not satisfied (pattern not found)")
        continue
    checks.append(f"{rel}: {label} passed")

if errors:
    emit_result("FAILED")
    print("Porting completion check FAILED:", file=sys.stderr)
    for item in errors:
        print(f"- {item}", file=sys.stderr)
    sys.exit(1)

emit_result("PASSED")
print(f"Porting completion check PASSED: {out_dir}")
for item in checks:
    print(f"- {item}")
PY
