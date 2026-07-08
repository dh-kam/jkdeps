#!/usr/bin/env bash
set -euo pipefail

ARTIFACT_DIR="${1:-/tmp/jkdeps-porting-ci}"
SUMMARY_FILE="${2:-}"

if [[ -z "$SUMMARY_FILE" ]]; then
  echo "usage: append_porting_step_summary.sh <artifact-dir> <summary-file>" >&2
  exit 2
fi

python3 - "$ARTIFACT_DIR" "$SUMMARY_FILE" <<'PY'
import json
import sys
from pathlib import Path

artifact_dir = Path(sys.argv[1])
summary_path = Path(sys.argv[2])


def load_json(path: Path):
    if not path.is_file():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


lines = []

audit = load_json(artifact_dir / "porting-audit.json")
if audit:
    lines.extend(
        [
            "### Porting Audit",
            "",
            f"- Status: `{audit.get('status', '')}`",
            f"- Failed stage: `{audit.get('failed_stage', '')}`",
            f"- Out dir: `{audit.get('out_dir', '')}`",
            f"- Baseline file: `{audit.get('baseline_file', '')}`",
            f"- Sync check: `{audit.get('sync_check', {}).get('status', '')}`",
            "",
        ]
    )

verify = load_json(artifact_dir / "porting-completion-verify.json")
if verify:
    lines.extend(
        [
            "### Porting Completion Verify",
            "",
            f"- Status: `{verify.get('status', '')}`",
            f"- Out dir: `{verify.get('out_dir', '')}`",
        ]
    )
    checks = verify.get("checks", [])
    if checks:
        lines.append("")
        lines.append("Key checks:")
        for item in checks[:8]:
            lines.append(f"- {item}")
    errors = verify.get("errors", [])
    if errors:
        lines.append("")
        lines.append("Errors:")
        for item in errors[:8]:
            lines.append(f"- {item}")
    lines.append("")

compare = load_json(artifact_dir / "porting-baseline-compare.json")
if compare:
    lines.extend(
        [
            "### Porting Baseline Compare",
            "",
            f"- Status: `{compare.get('status', '')}`",
            f"- Candidate: `{compare.get('candidate', '')}`",
            f"- Baseline: `{compare.get('baseline', '')}`",
        ]
    )
    checks = compare.get("checks", [])
    if checks:
        lines.append("")
        lines.append("Key checks:")
        for item in checks[:8]:
            lines.append(f"- {item}")
    errors = compare.get("errors", [])
    if errors:
        lines.append("")
        lines.append("Errors:")
        for item in errors[:8]:
            lines.append(f"- {item}")
    lines.append("")

parity = load_json(artifact_dir / "kotlin-official-parity.json")
if parity:
    summary = parity.get("summary", {})
    lines.extend(
        [
            "### Kotlin Official Parity",
            "",
            f"- Status: `{parity.get('status', '')}`",
            f"- Repo root: `{parity.get('repo_root', '')}`",
            f"- Files compared: `{summary.get('files_compared', '')}`",
            f"- Parse mismatch files: `{summary.get('parse_status_mismatch_files', '')}`",
            f"- Package mismatch files: `{summary.get('package_mismatch_files', '')}`",
            f"- Import mismatch files: `{summary.get('import_mismatch_files', '')}`",
            f"- Declaration mismatch files: `{summary.get('declaration_mismatch_files', '')}`",
        ]
    )
    errors = parity.get("errors", [])
    if errors:
        lines.append("")
        lines.append("Errors:")
        for item in errors[:8]:
            lines.append(f"- {item}")
    lines.append("")

if lines:
    summary_path.open("a", encoding="utf-8").write("\n".join(lines) + "\n")
PY
