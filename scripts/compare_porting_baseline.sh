#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-/tmp/jkdeps-porting}"
BASELINE_FILE="${2:-$ROOT_DIR/docs/porting-baseline.json}"
REPORT_FILE_MD="${3:-$OUT_DIR/porting-baseline-compare.md}"
REPORT_FILE_JSON="${4:-$OUT_DIR/porting-baseline-compare.json}"

python3 - "$OUT_DIR" "$BASELINE_FILE" "$REPORT_FILE_MD" "$REPORT_FILE_JSON" <<'PY'
import json
import re
import sys
from pathlib import Path

out_dir = Path(sys.argv[1])
baseline_path = Path(sys.argv[2])
report_md_path = Path(sys.argv[3])
report_json_path = Path(sys.argv[4])
errors = []
checks = []
result = {
    "status": "FAILED",
    "candidate": str(out_dir),
    "baseline": str(baseline_path),
    "report_md": str(report_md_path),
    "report_json": str(report_json_path),
    "checks": [],
    "errors": [],
    "acceptance": {},
    "mixed_graph": {},
    "mixed_dir_graph": {},
    "sample_refs": {},
    "runtime_inventory": {},
}


def fail(msg: str) -> None:
    errors.append(msg)


def matches_hex(value, length: int) -> bool:
    return isinstance(value, str) and re.fullmatch(rf"[0-9a-f]{{{length}}}", value) is not None


def parse_truthy(value) -> bool:
    return str(value).lower() in {"1", "true", "yes", "y"}


def load_json(path: Path, label: str):
    if not path.is_file():
        fail(f"missing {label}: {path}")
        return None
    try:
        with path.open() as f:
            return json.load(f)
    except Exception as exc:
        fail(f"invalid {label}: {path}: {exc}")
        return None


def write_reports(status: str) -> None:
    result["status"] = status
    result["checks"] = checks
    result["errors"] = errors

    report_md_path.parent.mkdir(parents=True, exist_ok=True)
    report_json_path.parent.mkdir(parents=True, exist_ok=True)
    report_json_path.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")

    if status == "PASSED":
        body = [
            "# Porting Baseline Compare",
            "",
            "Status: PASSED",
            "",
            f"- Candidate: `{out_dir}`",
            f"- Baseline: `{baseline_path}`",
            f"- JSON: `{report_json_path}`",
            "",
            "## Checks",
            "",
        ]
        body.extend(f"- {item}" for item in checks)
    else:
        body = [
            "# Porting Baseline Compare",
            "",
            "Status: FAILED",
            "",
            f"- Candidate: `{out_dir}`",
            f"- Baseline: `{baseline_path}`",
            f"- JSON: `{report_json_path}`",
            "",
            "## Errors",
            "",
        ]
        body.extend(f"- {item}" for item in errors)
        if checks:
            body.extend(["", "## Checks", ""])
            body.extend(f"- {item}" for item in checks)

    report_md_path.write_text("\n".join(body) + "\n", encoding="utf-8")


baseline = load_json(baseline_path, "baseline file")
if baseline is None:
    write_reports("FAILED")
    print("Porting baseline compare FAILED:", file=sys.stderr)
    for item in errors:
        print(f"- {item}", file=sys.stderr)
    print(f"- report_md:   {report_md_path}", file=sys.stderr)
    print(f"- report_json: {report_json_path}", file=sys.stderr)
    sys.exit(1)

acceptance = baseline.get("acceptance", {})
if not isinstance(acceptance, dict) or not acceptance:
    fail(f"baseline acceptance section is missing or empty: {baseline_path}")

baseline_sample_refs = baseline.get("sample_refs", {})
baseline_runtime_inventory = baseline.get("runtime_inventory", {})
needs_candidate_metadata = bool(baseline_sample_refs) or bool(baseline_runtime_inventory)
candidate_metadata = None
candidate_metadata_path = out_dir / "porting-run-metadata.json"
if needs_candidate_metadata:
    candidate_metadata = load_json(candidate_metadata_path, "candidate run metadata")

candidate_run_flags = {}
if candidate_metadata is not None:
    candidate_run_flags = candidate_metadata.get("run_flags", {})
    if not isinstance(candidate_run_flags, dict):
        candidate_run_flags = {}

run_kotlin_core_mixed_graph = parse_truthy(candidate_run_flags.get("run_kotlin_core_mixed_graph", "1"))
run_kotlin_core_mixed_dir_graph = parse_truthy(
    candidate_run_flags.get("run_kotlin_core_mixed_dir_graph", "1" if run_kotlin_core_mixed_graph else "0")
)

if baseline_sample_refs:
    if not isinstance(baseline_sample_refs, dict):
        fail(f"baseline sample_refs must be an object: {baseline_path}")
    else:
        result["sample_refs"]["baseline"] = baseline_sample_refs
        result["sample_refs"]["candidate"] = {}
        if candidate_metadata is not None:
            candidate_sample_refs = candidate_metadata.get("sample_refs")
            if not isinstance(candidate_sample_refs, dict):
                fail(f"{candidate_metadata_path}: sample_refs missing or invalid")
            else:
                for sample_name, expected_ref in baseline_sample_refs.items():
                    candidate_ref = candidate_sample_refs.get(sample_name)
                    result["sample_refs"]["candidate"][sample_name] = candidate_ref
                    if not matches_hex(candidate_ref, 40):
                        fail(
                            f"{candidate_metadata_path}: sample_refs.{sample_name} must be a 40-char git sha "
                            f"(got {candidate_ref!r})"
                        )
                    elif candidate_ref != expected_ref:
                        fail(
                            f"sample_refs mismatch for {sample_name}: candidate {candidate_ref} != baseline {expected_ref}"
                        )
                    else:
                        checks.append(
                            f"sample_refs.{sample_name}: {candidate_ref} (matches baseline)"
                        )

if baseline_runtime_inventory:
    if not isinstance(baseline_runtime_inventory, dict):
        fail(f"baseline runtime_inventory must be an object: {baseline_path}")
    else:
        expected_inventory_sha = baseline_runtime_inventory.get("sha256")
        result["runtime_inventory"]["baseline"] = {
            "sha256": expected_inventory_sha,
        }
        result["runtime_inventory"]["candidate"] = {}

        if not matches_hex(expected_inventory_sha, 64):
            fail(
                f"baseline runtime_inventory.sha256 must be a 64-char sha256: {expected_inventory_sha!r}"
            )

        if candidate_metadata is not None:
            candidate_inventory = candidate_metadata.get("inventory")
            if not isinstance(candidate_inventory, dict):
                fail(f"{candidate_metadata_path}: inventory missing or invalid")
            else:
                candidate_inventory_sha = candidate_inventory.get("sha256")
                result["runtime_inventory"]["candidate"]["sha256"] = candidate_inventory_sha
                if not matches_hex(candidate_inventory_sha, 64):
                    fail(
                        f"{candidate_metadata_path}: inventory.sha256 must be a 64-char sha256 "
                        f"(got {candidate_inventory_sha!r})"
                    )
                elif candidate_inventory_sha != expected_inventory_sha:
                    fail(
                        f"runtime inventory sha mismatch: candidate {candidate_inventory_sha} != "
                        f"baseline {expected_inventory_sha}"
                    )
                else:
                    checks.append(
                        f"runtime_inventory.sha256: {candidate_inventory_sha} (matches baseline)"
                    )

for rel, expected in acceptance.items():
    if not isinstance(expected, dict):
        fail(f"baseline acceptance entry must be object: {rel}")
        continue
    cand_path = out_dir / rel
    cand = load_json(cand_path, f"candidate acceptance report ({rel})")
    if cand is None:
        continue

    parse = cand.get("parse", {})
    resolve = cand.get("resolve", {})
    graph = cand.get("graph", {})

    cand_failed = parse.get("failed_files")
    cand_unresolved = resolve.get("unresolved_imports")
    cand_unknown = graph.get("unknown_nodes")
    cand_diag_files = parse.get("files_with_diagnostics")
    cand_total_diag = parse.get("total_diagnostics")

    exp_failed = expected.get("failed_files", 0)
    exp_unresolved = expected.get("unresolved_imports", 0)
    exp_unknown = expected.get("unknown_nodes", 0)
    exp_diag_files = expected.get("files_with_diagnostics", 0)
    exp_total_diag = expected.get("total_diagnostics", 0)
    max_reg_diag_files = expected.get("max_regression_files_with_diagnostics", 0)
    max_reg_total_diag = expected.get("max_regression_total_diagnostics", 0)
    allowed_diag_files = exp_diag_files + max_reg_diag_files
    allowed_total_diag = exp_total_diag + max_reg_total_diag

    result["acceptance"][rel] = {
        "baseline": {
            "failed_files": exp_failed,
            "unresolved_imports": exp_unresolved,
            "unknown_nodes": exp_unknown,
            "files_with_diagnostics": exp_diag_files,
            "total_diagnostics": exp_total_diag,
        },
        "allowed": {
            "files_with_diagnostics": allowed_diag_files,
            "total_diagnostics": allowed_total_diag,
        },
        "candidate": {
            "failed_files": cand_failed,
            "unresolved_imports": cand_unresolved,
            "unknown_nodes": cand_unknown,
            "files_with_diagnostics": cand_diag_files,
            "total_diagnostics": cand_total_diag,
        },
    }

    if not isinstance(cand_failed, int):
        fail(f"{rel}: parse.failed_files missing or invalid")
    elif cand_failed > exp_failed:
        fail(f"{rel}: parse.failed_files regression: {cand_failed} > baseline {exp_failed}")

    if not isinstance(cand_unresolved, int):
        fail(f"{rel}: resolve.unresolved_imports missing or invalid")
    elif cand_unresolved > exp_unresolved:
        fail(f"{rel}: resolve.unresolved_imports regression: {cand_unresolved} > baseline {exp_unresolved}")

    if not isinstance(cand_unknown, int):
        fail(f"{rel}: graph.unknown_nodes missing or invalid")
    elif cand_unknown > exp_unknown:
        fail(f"{rel}: graph.unknown_nodes regression: {cand_unknown} > baseline {exp_unknown}")

    if not isinstance(cand_diag_files, int):
        fail(f"{rel}: parse.files_with_diagnostics missing or invalid")
    else:
        if cand_diag_files > allowed_diag_files:
            fail(
                f"{rel}: parse.files_with_diagnostics regression: {cand_diag_files} > allowed {allowed_diag_files} "
                f"(baseline={exp_diag_files}, max_regression={max_reg_diag_files})"
            )

    if not isinstance(cand_total_diag, int):
        fail(f"{rel}: parse.total_diagnostics missing or invalid")
    else:
        if cand_total_diag > allowed_total_diag:
            fail(
                f"{rel}: parse.total_diagnostics regression: {cand_total_diag} > allowed {allowed_total_diag} "
                f"(baseline={exp_total_diag}, max_regression={max_reg_total_diag})"
            )

    if isinstance(cand_diag_files, int) and isinstance(cand_total_diag, int):
        checks.append(
            f"{rel}: diag_files={cand_diag_files} (baseline {exp_diag_files}), "
            f"total_diag={cand_total_diag} (baseline {exp_total_diag})"
        )

def evaluate_mixed_graph(section_name: str, default_log_file: str) -> None:
    mixed = baseline.get(section_name, {})
    if not isinstance(mixed, dict) or not mixed:
        return

    log_rel = mixed.get("log_file", default_log_file)
    min_java = int(mixed.get("min_java_files", 1))
    min_kotlin = int(mixed.get("min_kotlin_files", 1))
    require_failed_zero = bool(mixed.get("require_failed_zero", True))

    log_path = out_dir / log_rel
    if not log_path.is_file():
        fail(f"missing mixed graph log: {log_path}")
        return

    text = log_path.read_text(errors="replace")
    files_match = re.search(r"Files:\s+total=(\d+)\s+java=(\d+)\s+kotlin=(\d+)", text)
    parse_match = re.search(r"(?:ParseStatus|Parse):\s+parsed=(\d+)\s+failed=(\d+)", text)
    result[section_name] = {
        "log_file": str(log_path),
        "min_java_files": min_java,
        "min_kotlin_files": min_kotlin,
        "require_failed_zero": require_failed_zero,
    }
    if files_match is None:
        fail(f"{log_rel}: could not find Files line")
    else:
        total = int(files_match.group(1))
        java = int(files_match.group(2))
        kotlin = int(files_match.group(3))
        result[section_name]["candidate_files"] = {
            "total": total,
            "java": java,
            "kotlin": kotlin,
        }
        if java < min_java:
            fail(f"{log_rel}: java files below baseline expectation: {java} < {min_java}")
        if kotlin < min_kotlin:
            fail(f"{log_rel}: kotlin files below baseline expectation: {kotlin} < {min_kotlin}")
        checks.append(
            f"{log_rel}: files total={total} java={java} kotlin={kotlin} "
            f"(min_java={min_java}, min_kotlin={min_kotlin})"
        )
    if parse_match is None:
        fail(f"{log_rel}: could not find parse status line")
    else:
        parsed = int(parse_match.group(1))
        failed = int(parse_match.group(2))
        result[section_name]["candidate_parse_status"] = {
            "parsed": parsed,
            "failed": failed,
        }
        if require_failed_zero and failed != 0:
            fail(f"{log_rel}: parse failed count regression: {failed} != 0")
        checks.append(f"{log_rel}: parse parsed={parsed} failed={failed}")


if run_kotlin_core_mixed_graph:
    evaluate_mixed_graph("mixed_graph", "kotlin_core_mixed_graph_lenient.log")
else:
    checks.append("mixed_graph: skipped by run flag")

if run_kotlin_core_mixed_dir_graph:
    evaluate_mixed_graph("mixed_dir_graph", "kotlin_core_mixed_dir_graph_lenient.log")
else:
    checks.append("mixed_dir_graph: skipped by run flag")

if errors:
    write_reports("FAILED")
    print("Porting baseline compare FAILED:", file=sys.stderr)
    print(f"- candidate: {out_dir}", file=sys.stderr)
    print(f"- baseline:  {baseline_path}", file=sys.stderr)
    print(f"- report_md:   {report_md_path}", file=sys.stderr)
    print(f"- report_json: {report_json_path}", file=sys.stderr)
    for item in errors:
        print(f"- {item}", file=sys.stderr)
    sys.exit(1)

write_reports("PASSED")

print("Porting baseline compare PASSED:")
print(f"- candidate: {out_dir}")
print(f"- baseline:  {baseline_path}")
print(f"- report_md:   {report_md_path}")
print(f"- report_json: {report_json_path}")
for item in checks:
    print(f"- {item}")
PY
