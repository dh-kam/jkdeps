#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_PATH="${1:-/tmp/jkdeps-samples/kotlinx.coroutines/kotlinx-coroutines-core/js/src}"
OUT_DIR="${2:-/tmp/jkdeps-porting}"
REPORT_JSON="${3:-$OUT_DIR/kotlin-official-parity.json}"
INCLUDE_KTS="${INCLUDE_KTS:-1}"
MAX_MISSING_IN_GO="${MAX_MISSING_IN_GO:-0}"
MAX_MISSING_IN_OFFICIAL="${MAX_MISSING_IN_OFFICIAL:-0}"
MAX_PARSE_STATUS_MISMATCH="${MAX_PARSE_STATUS_MISMATCH:-0}"
MAX_PACKAGE_MISMATCH="${MAX_PACKAGE_MISMATCH:-0}"
MAX_IMPORT_MISMATCH="${MAX_IMPORT_MISMATCH:-0}"
MAX_DECLARATION_MISMATCH="${MAX_DECLARATION_MISMATCH:-0}"
KCG_PARSER_BACKEND="${KCG_PARSER_BACKEND:-antlr}"
INCLUDE_BUILD_SCRIPTS="${INCLUDE_BUILD_SCRIPTS:-0}"

mkdir -p "$OUT_DIR"

EMBEDDABLE_JAR="$($ROOT_DIR/scripts/fetch_kotlin_compiler_embeddable.sh "$ROOT_DIR/tools")"
mapfile -t RUNTIME_JARS < <("$ROOT_DIR/scripts/fetch_kotlin_compiler_runtime_jars.sh" "$ROOT_DIR/tools")

CP_ENTRIES=("$EMBEDDABLE_JAR")
for jar in "${RUNTIME_JARS[@]}"; do
  CP_ENTRIES+=("$jar")
done
CP_JARS="$(IFS=:; echo "${CP_ENTRIES[*]}")"

PARITY_SRC="$ROOT_DIR/tools/kotlin-official-parity/KotlinOfficialSnapshot.java"
PARITY_CLASSES_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kotlin-official-parity.XXXXXX")"
trap 'rm -rf "$PARITY_CLASSES_DIR"' EXIT

javac -proc:none -cp "$CP_JARS" -d "$PARITY_CLASSES_DIR" "$PARITY_SRC"

if [[ "$INCLUDE_KTS" == "1" || "$INCLUDE_KTS" == "true" || "$INCLUDE_KTS" == "yes" ]]; then
  include_kts_bool="true"
else
  include_kts_bool="false"
fi
if [[ "$INCLUDE_BUILD_SCRIPTS" == "1" || "$INCLUDE_BUILD_SCRIPTS" == "true" || "$INCLUDE_BUILD_SCRIPTS" == "yes" ]]; then
  include_build_scripts_bool="true"
else
  include_build_scripts_bool="false"
fi

official_json="$OUT_DIR/kotlin-official-snapshot.json"
go_json="$OUT_DIR/kotlin-go-parse.json"

java -cp "$PARITY_CLASSES_DIR:$CP_JARS" KotlinOfficialSnapshot \
  --repo "$REPO_PATH" \
  --out "$official_json" \
  --include-kts "$include_kts_bool"

go run ./cmd/kotlin-compiler-golang parse \
  --repo "$REPO_PATH" \
  --include-kts="$include_kts_bool" \
  --include-build-scripts="$include_build_scripts_bool" \
  --fail-on-error=false \
  --parser-backend "$KCG_PARSER_BACKEND" \
  --json > "$go_json"

python3 - "$REPO_PATH" "$go_json" "$official_json" "$REPORT_JSON" \
  "$MAX_MISSING_IN_GO" "$MAX_MISSING_IN_OFFICIAL" "$MAX_PARSE_STATUS_MISMATCH" "$MAX_PACKAGE_MISMATCH" "$MAX_IMPORT_MISMATCH" "$MAX_DECLARATION_MISMATCH" "$include_build_scripts_bool" <<'PY'
import collections
import datetime
import json
import os
import sys
from pathlib import Path

repo_root = Path(sys.argv[1]).resolve()
go_json_path = Path(sys.argv[2])
official_json_path = Path(sys.argv[3])
report_path = Path(sys.argv[4])
max_missing_in_go = int(sys.argv[5])
max_missing_in_official = int(sys.argv[6])
max_parse_status = int(sys.argv[7])
max_package = int(sys.argv[8])
max_import = int(sys.argv[9])
max_decl = int(sys.argv[10])
include_build_scripts = sys.argv[11].lower() == "true"

def is_build_script(path: str) -> bool:
    lower = path.lower()
    return lower.endswith("build.gradle.kts") or lower.endswith("settings.gradle.kts")

with go_json_path.open() as f:
    go_doc = json.load(f)
with official_json_path.open() as f:
    official_doc = json.load(f)


def norm_rel(path: str, root: Path) -> str:
    p = Path(path)
    if not p.is_absolute():
        p = (root / p).resolve()
    else:
        p = p.resolve()
    try:
        rel = p.relative_to(root)
    except Exception:
        rel = Path(os.path.relpath(str(p), str(root)))
    return rel.as_posix()


def normalize_decl(item: dict) -> str:
    kind = str(item.get("kind", "")).strip()
    name = str(item.get("name", "")).strip()
    if not kind or not name:
        return ""
    # Official KtFile declarations are top-level only.
    if "." in name:
        return ""
    return f"{kind}:{name}"


go_root = Path(go_doc.get("root", str(repo_root))).resolve()

go_files = {}
for unit in go_doc.get("files", []):
    path = unit.get("path")
    if not isinstance(path, str) or not path:
        continue
    rel = norm_rel(path, go_root)
    imports = sorted(set(str(v) for v in unit.get("imports", []) if isinstance(v, str)))
    decls = [normalize_decl(v) for v in unit.get("declarations", []) if isinstance(v, dict)]
    decls = sorted(v for v in decls if v)
    go_files[rel] = {
        "parsed": bool(unit.get("parsed", False)),
        "package_name": str(unit.get("package_name", "")),
        "imports": imports,
        "declarations": decls,
    }

official_files = {}
for unit in official_doc.get("files", []):
    if not isinstance(unit, dict):
        continue
    path = unit.get("path")
    if not isinstance(path, str) or not path:
        continue
    if not include_build_scripts and is_build_script(path):
        continue
    rel = path.replace("\\", "/")
    imports = sorted(set(str(v) for v in unit.get("imports", []) if isinstance(v, str)))
    decls = [normalize_decl(v) for v in unit.get("declarations", []) if isinstance(v, dict)]
    decls = sorted(v for v in decls if v)
    error_count = unit.get("error_count", 0)
    if not isinstance(error_count, int):
        error_count = 0
    official_files[rel] = {
        "error_count": error_count,
        "package_name": str(unit.get("package_name", "")),
        "imports": imports,
        "declarations": decls,
    }

all_go = set(go_files)
all_official = set(official_files)
shared = sorted(all_go & all_official)
missing_in_go = sorted(all_official - all_go)
missing_in_official = sorted(all_go - all_official)

parse_status_mismatch = []
package_mismatch = []
import_mismatch = []
decl_mismatch = []

for rel in shared:
    go_unit = go_files[rel]
    official_unit = official_files[rel]

    official_parsed = official_unit["error_count"] == 0
    go_parsed = go_unit["parsed"]
    if official_parsed != go_parsed:
        parse_status_mismatch.append(
            {
                "path": rel,
                "official_parsed": official_parsed,
                "go_parsed": go_parsed,
                "official_error_count": official_unit["error_count"],
            }
        )

    if official_unit["package_name"] != go_unit["package_name"]:
        package_mismatch.append(
            {
                "path": rel,
                "official_package": official_unit["package_name"],
                "go_package": go_unit["package_name"],
            }
        )

    official_imports = set(official_unit["imports"])
    go_imports = set(go_unit["imports"])
    if official_imports != go_imports:
        import_mismatch.append(
            {
                "path": rel,
                "only_in_official": sorted(official_imports - go_imports),
                "only_in_go": sorted(go_imports - official_imports),
            }
        )

    # Declaration parity is meaningful only when both parsers accept the file.
    if go_parsed and official_parsed:
        official_decl = collections.Counter(official_unit["declarations"])
        go_decl = collections.Counter(go_unit["declarations"])
        if official_decl != go_decl:
            official_only = []
            go_only = []
            for key in sorted(official_decl.keys() | go_decl.keys()):
                diff = official_decl[key] - go_decl[key]
                if diff > 0:
                    official_only.extend([key] * diff)
                elif diff < 0:
                    go_only.extend([key] * (-diff))
            decl_mismatch.append(
                {
                    "path": rel,
                    "only_in_official": official_only,
                    "only_in_go": go_only,
                }
            )

summary = {
    "official_total_files": len(official_files),
    "go_total_files": len(go_files),
    "files_compared": len(shared),
    "missing_in_go_files": len(missing_in_go),
    "missing_in_official_files": len(missing_in_official),
    "parse_status_mismatch_files": len(parse_status_mismatch),
    "package_mismatch_files": len(package_mismatch),
    "import_mismatch_files": len(import_mismatch),
    "declaration_mismatch_files": len(decl_mismatch),
}

errors = []
if summary["missing_in_go_files"] > max_missing_in_go:
    errors.append(
        f"missing_in_go_files={summary['missing_in_go_files']} > max_missing_in_go={max_missing_in_go}"
    )
if summary["missing_in_official_files"] > max_missing_in_official:
    errors.append(
        f"missing_in_official_files={summary['missing_in_official_files']} > max_missing_in_official={max_missing_in_official}"
    )
if summary["parse_status_mismatch_files"] > max_parse_status:
    errors.append(
        f"parse_status_mismatch_files={summary['parse_status_mismatch_files']} > max_parse_status_mismatch={max_parse_status}"
    )
if summary["package_mismatch_files"] > max_package:
    errors.append(
        f"package_mismatch_files={summary['package_mismatch_files']} > max_package_mismatch={max_package}"
    )
if summary["import_mismatch_files"] > max_import:
    errors.append(
        f"import_mismatch_files={summary['import_mismatch_files']} > max_import_mismatch={max_import}"
    )
if summary["declaration_mismatch_files"] > max_decl:
    errors.append(
        f"declaration_mismatch_files={summary['declaration_mismatch_files']} > max_declaration_mismatch={max_decl}"
    )

status = "PASSED" if not errors else "FAILED"

report = {
    "status": status,
    "generated_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "repo_root": str(repo_root),
    "artifacts": {
        "go_parse_json": str(go_json_path),
        "official_snapshot_json": str(official_json_path),
    },
    "thresholds": {
        "max_missing_in_go": max_missing_in_go,
        "max_missing_in_official": max_missing_in_official,
        "max_parse_status_mismatch": max_parse_status,
        "max_package_mismatch": max_package,
        "max_import_mismatch": max_import,
        "max_declaration_mismatch": max_decl,
    },
    "summary": summary,
    "errors": errors,
    "mismatches": {
        "missing_in_go": missing_in_go,
        "missing_in_official": missing_in_official,
        "parse_status": parse_status_mismatch,
        "package": package_mismatch,
        "imports": import_mismatch,
        "declarations": decl_mismatch,
    },
}

report_path.parent.mkdir(parents=True, exist_ok=True)
report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")

print(f"kotlin official parity {status}: {report_path}")
for key in (
    "official_total_files",
    "go_total_files",
    "files_compared",
    "missing_in_go_files",
    "missing_in_official_files",
    "parse_status_mismatch_files",
    "package_mismatch_files",
    "import_mismatch_files",
    "declaration_mismatch_files",
):
    print(f"- {key}={summary[key]}")

if errors:
    for item in errors:
        print(f"- error: {item}", file=sys.stderr)
    raise SystemExit(1)
PY
