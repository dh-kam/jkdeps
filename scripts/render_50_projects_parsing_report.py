#!/usr/bin/env python3
import argparse
import csv
import datetime as dt
import math
import pathlib
import statistics
from collections import Counter, defaultdict


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--summary", required=True)
    parser.add_argument("--roundtrip-summary")
    parser.add_argument("--output", required=True)
    parser.add_argument("--manifest", required=True)
    parser.add_argument("--samples-dir", required=True)
    parser.add_argument("--out-dir", required=True)
    return parser.parse_args()


def to_int(value: str) -> int:
    try:
        return int(value or "0")
    except ValueError:
        return 0


def to_float(value: str) -> float:
    try:
        return float(value or "0")
    except ValueError:
        return 0.0


def percentile(values, pct: float) -> float:
    if not values:
        return 0.0
    if len(values) == 1:
        return values[0]
    pos = (len(values) - 1) * pct
    lower = math.floor(pos)
    upper = math.ceil(pos)
    if lower == upper:
        return values[lower]
    fraction = pos - lower
    return values[lower] + (values[upper] - values[lower]) * fraction


def format_seconds(value: float) -> str:
    return f"{value:.2f}s"


def aggregate(rows):
    durations = sorted(row["duration_seconds"] for row in rows)
    total_files = sum(row["total_files"] for row in rows)
    parsed_files = sum(row["parsed_files"] for row in rows)
    failed_files = sum(row["failed_files"] for row in rows)
    unresolved = sum(row["unresolved_count"] for row in rows)
    dependency_count = sum(row["dependency_count"] for row in rows)
    status_counts = Counter(row["status"] for row in rows)
    success_rate = 0.0
    if total_files:
        success_rate = (parsed_files / total_files) * 100.0
    return {
        "projects": len(rows),
        "total_files": total_files,
        "parsed_files": parsed_files,
        "failed_files": failed_files,
        "dependency_count": dependency_count,
        "unresolved_count": unresolved,
        "success_rate": success_rate,
        "status_counts": status_counts,
        "total_duration": sum(durations),
        "avg_duration": statistics.fmean(durations) if durations else 0.0,
        "median_duration": statistics.median(durations) if durations else 0.0,
        "p95_duration": percentile(durations, 0.95),
        "max_duration": max(durations) if durations else 0.0,
    }


def aggregate_roundtrip(rows):
    durations = sorted(row["duration_seconds"] for row in rows)
    parse_durations = sorted(row["parse_duration_seconds"] for row in rows)
    total_files = sum(row["total_files"] for row in rows)
    passed_files = sum(row["passed_files"] for row in rows)
    diff_files = sum(row["diff_files"] for row in rows)
    parse_failed = sum(row["parse_failed_files"] for row in rows)
    unsupported = sum(row["unsupported_files"] for row in rows)
    format_errors = sum(row["format_error_files"] for row in rows)
    status_counts = Counter(row["status"] for row in rows)
    exact_rate = 0.0
    if total_files:
        exact_rate = (passed_files / total_files) * 100.0
    return {
        "projects": len(rows),
        "total_files": total_files,
        "passed_files": passed_files,
        "diff_files": diff_files,
        "parse_failed_files": parse_failed,
        "unsupported_files": unsupported,
        "format_error_files": format_errors,
        "exact_rate": exact_rate,
        "status_counts": status_counts,
        "total_duration": sum(durations),
        "avg_duration": statistics.fmean(durations) if durations else 0.0,
        "median_duration": statistics.median(durations) if durations else 0.0,
        "p95_duration": percentile(durations, 0.95),
        "max_duration": max(durations) if durations else 0.0,
        "total_parse_duration": sum(parse_durations),
        "avg_parse_duration": statistics.fmean(parse_durations) if parse_durations else 0.0,
    }


def load_targets(out_dir: pathlib.Path):
    targets = {}
    target_path = out_dir / "selected-targets.csv"
    if not target_path.exists():
        return targets
    with target_path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        for raw in reader:
            targets[raw["name"]] = raw
    return targets


def load_roundtrip_rows(path: str | None, targets):
    if not path:
        return []
    summary_path = pathlib.Path(path)
    if not summary_path.exists():
        return []
    rows = []
    with summary_path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        for raw in reader:
            target = targets.get(raw["name"], {})
            rows.append(
                {
                    "status": raw["status"],
                    "name": raw["name"],
                    "language": raw["language"],
                    "total_files": to_int(raw["total_files"]),
                    "checked_files": to_int(raw["checked_files"]),
                    "passed_files": to_int(raw["passed_files"]),
                    "diff_files": to_int(raw["diff_files"]),
                    "parse_failed_files": to_int(raw["parse_failed_files"]),
                    "unsupported_files": to_int(raw["unsupported_files"]),
                    "format_error_files": to_int(raw["format_error_files"]),
                    "exact_rate": to_float(raw["exact_rate"]),
                    "duration_seconds": to_float(raw["duration_seconds"]),
                    "parse_duration_seconds": to_float(raw["parse_duration_seconds"]),
                    "rewrite_mode": raw.get("rewrite_mode", ""),
                    "java_format_cmd": raw.get("java_format_cmd", ""),
                    "kotlin_format_cmd": raw.get("kotlin_format_cmd", ""),
                    "target_dir": raw.get("target_dir", ""),
                    "relative_target": target.get("relative_target", ""),
                }
            )
    return rows


def render_table(rows):
    header = "| Project | Status | Files | Parsed | Failed | Parse Success | Duration | Target |\n"
    header += "| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |\n"
    lines = [header]
    for row in rows:
        success = 0.0
        if row["total_files"]:
            success = (row["parsed_files"] / row["total_files"]) * 100.0
        target = row.get("relative_target") or "-"
        lines.append(
            f"| `{row['name']}` | {row['status']} | {row['total_files']} | "
            f"{row['parsed_files']} | {row['failed_files']} | {success:.2f}% | "
            f"{row['duration_seconds']:.2f}s | `{target}` |"
        )
    return "\n".join(lines)


def render_roundtrip_table(rows):
    header = "| Project | Status | Files | Pass | Diff | Parse Failed | Unsupported | Format Error | Exact | Duration | Target |\n"
    header += "| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n"
    lines = [header]
    for row in rows:
        target = row.get("relative_target") or row.get("target_dir") or "-"
        lines.append(
            f"| `{row['name']}` | {row['status']} | {row['total_files']} | "
            f"{row['passed_files']} | {row['diff_files']} | {row['parse_failed_files']} | "
            f"{row['unsupported_files']} | {row['format_error_files']} | "
            f"{row['exact_rate']:.2f}% | {row['duration_seconds']:.2f}s | `{target}` |"
        )
    return "\n".join(lines)


def render_summary_table(language_stats):
    lines = [
        "| Language Group | Projects | OK | Timeout | Runtime Error | Files | Parsed | Failed | Parse Success | Total Time | Avg | Median | P95 |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for language in ("java", "kotlin"):
        stats = language_stats.get(language, aggregate([]))
        status_counts = stats["status_counts"]
        lines.append(
            f"| {language} | {stats['projects']} | {status_counts.get('ok', 0)} | "
            f"{status_counts.get('timeout', 0)} | {status_counts.get('runtime-error', 0)} | "
            f"{stats['total_files']} | {stats['parsed_files']} | {stats['failed_files']} | "
            f"{stats['success_rate']:.2f}% | {format_seconds(stats['total_duration'])} | "
            f"{format_seconds(stats['avg_duration'])} | {format_seconds(stats['median_duration'])} | "
            f"{format_seconds(stats['p95_duration'])} |"
        )
    return "\n".join(lines)


def render_roundtrip_summary_table(language_stats):
    lines = [
        "| Language Group | Projects | OK | Findings | Runtime Error | Timeout | Files | Pass | Diff | Parse Failed | Unsupported | Format Error | Exact | Total Time | Avg | Median | P95 |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for language in ("java", "kotlin"):
        stats = language_stats.get(language, aggregate_roundtrip([]))
        status_counts = stats["status_counts"]
        lines.append(
            f"| {language} | {stats['projects']} | {status_counts.get('ok', 0)} | "
            f"{status_counts.get('findings', 0)} | {status_counts.get('runtime-error', 0)} | "
            f"{status_counts.get('timeout', 0)} | {stats['total_files']} | "
            f"{stats['passed_files']} | {stats['diff_files']} | {stats['parse_failed_files']} | "
            f"{stats['unsupported_files']} | {stats['format_error_files']} | "
            f"{stats['exact_rate']:.2f}% | {format_seconds(stats['total_duration'])} | "
            f"{format_seconds(stats['avg_duration'])} | {format_seconds(stats['median_duration'])} | "
            f"{format_seconds(stats['p95_duration'])} |"
        )
    return "\n".join(lines)


def main():
    args = parse_args()
    summary_path = pathlib.Path(args.summary)
    output_path = pathlib.Path(args.output)
    targets = load_targets(pathlib.Path(args.out_dir))
    rows = []
    with summary_path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        for raw in reader:
            target = targets.get(raw["name"], {})
            rows.append(
                {
                    "status": raw["status"],
                    "name": raw["name"],
                    "language": raw["language"],
                    "total_files": to_int(raw["total_files"]),
                    "parsed_files": to_int(raw["parsed_files"]),
                    "failed_files": to_int(raw["failed_files"]),
                    "dependency_count": to_int(raw["dependency_count"]),
                    "unresolved_count": to_int(raw["unresolved_count"]),
                    "duration_seconds": to_float(raw["duration_seconds"]),
                    "expectation_status": raw["expectation_status"],
                    "expectation_detail": raw["expectation_detail"],
                    "relative_target": target.get("relative_target", ""),
                    "selection_mode": target.get("selection_mode", ""),
                }
            )

    by_language = defaultdict(list)
    for row in rows:
        by_language[row["language"]].append(row)

    roundtrip_rows = load_roundtrip_rows(args.roundtrip_summary, targets)
    roundtrip_by_language = defaultdict(list)
    for row in roundtrip_rows:
        roundtrip_by_language[row["language"]].append(row)

    language_stats = {language: aggregate(items) for language, items in by_language.items()}
    overall_stats = aggregate(rows)
    roundtrip_language_stats = {language: aggregate_roundtrip(items) for language, items in roundtrip_by_language.items()}
    roundtrip_overall_stats = aggregate_roundtrip(roundtrip_rows)
    top_slowest = sorted(rows, key=lambda row: row["duration_seconds"], reverse=True)[:15]
    worst_parse = sorted(
        [row for row in rows if row["total_files"] > 0],
        key=lambda row: (row["parsed_files"] / row["total_files"]),
    )[:15]
    roundtrip_findings = sorted(
        [
            row
            for row in roundtrip_rows
            if row["diff_files"] or row["parse_failed_files"] or row["unsupported_files"] or row["format_error_files"]
        ],
        key=lambda row: (row["exact_rate"], -row["total_files"], row["name"].lower()),
    )[:25]
    roundtrip_slowest = sorted(roundtrip_rows, key=lambda row: row["duration_seconds"], reverse=True)[:15]
    rewrite_modes = sorted({row["rewrite_mode"] for row in roundtrip_rows if row["rewrite_mode"]})
    java_formatters = sorted({row["java_format_cmd"] for row in roundtrip_rows if row["java_format_cmd"]})
    kotlin_formatters = sorted({row["kotlin_format_cmd"] for row in roundtrip_rows if row["kotlin_format_cmd"]})
    roundtrip_section = "Round-trip summary was not generated."
    if roundtrip_rows:
        roundtrip_section = f"""## Round-Trip Summary

- Projects checked: **{roundtrip_overall_stats['projects']}**
- Completed with exact `ok`: **{roundtrip_overall_stats['status_counts'].get('ok', 0)}**
- Projects with findings: **{roundtrip_overall_stats['status_counts'].get('findings', 0)}**
- Runtime errors/timeouts: **{roundtrip_overall_stats['status_counts'].get('runtime-error', 0) + roundtrip_overall_stats['status_counts'].get('timeout', 0)}**
- Total files checked: **{roundtrip_overall_stats['total_files']}**
- Exact pass files: **{roundtrip_overall_stats['passed_files']}**
- Diff files: **{roundtrip_overall_stats['diff_files']}**
- Parse-failed files: **{roundtrip_overall_stats['parse_failed_files']}**
- Unsupported files: **{roundtrip_overall_stats['unsupported_files']}**
- Formatter error files: **{roundtrip_overall_stats['format_error_files']}**
- Aggregate exact rate: **{roundtrip_overall_stats['exact_rate']:.2f}%**
- Rewrite mode(s): **{', '.join(rewrite_modes) if rewrite_modes else 'not recorded'}**
- Java formatter command(s): **{', '.join(java_formatters) if java_formatters else 'not configured/detected'}**
- Kotlin formatter command(s): **{', '.join(kotlin_formatters) if kotlin_formatters else 'not configured/detected'}**

## Round-Trip Duration Summary

- Total round-trip time across all projects: **{format_seconds(roundtrip_overall_stats['total_duration'])}**
- Average per project: **{format_seconds(roundtrip_overall_stats['avg_duration'])}**
- Median per project: **{format_seconds(roundtrip_overall_stats['median_duration'])}**
- P95 per project: **{format_seconds(roundtrip_overall_stats['p95_duration'])}**
- Max per project: **{format_seconds(roundtrip_overall_stats['max_duration'])}**
- Parser time inside round-trip command: **{format_seconds(roundtrip_overall_stats['total_parse_duration'])}**

## Round-Trip By Language Group

{render_roundtrip_summary_table(roundtrip_language_stats)}

## Round-Trip Findings

{render_roundtrip_table(roundtrip_findings) if roundtrip_findings else 'No round-trip findings were recorded.'}

## Slowest Round-Trip Projects

{render_roundtrip_table(roundtrip_slowest)}
"""

    generated_at = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    doc = f"""# 50 Java + 50 Kotlin Projects Parsing Result

Generated at: **{generated_at}**

## Scope

- Manifest: `{args.manifest}`
- Sources root: `{args.samples_dir}`
- Raw outputs: `{args.out_dir}`
- Input set: **50 Java-based OSS projects + 50 Kotlin-based OSS projects**
- Runner: `scripts/oss_dependency_matrix.sh` with `scripts/oss_50_projects_targets.txt`
- Target directory selection: `OSS_MATRIX_AUTO_TARGET=1` heuristic that picks the primary source root and then narrows to a representative subtree when the source root is too large (default cap: 120 source files)
- Parser mode: `go run ./cmd/jkdeps smoke-parse --fail-on-error=false`
- Round-trip mode: `jkdeps roundtrip-check --rewrite-mode lossless --json`
- Duration metric: `/usr/bin/time` wall-clock seconds for the `smoke-parse` command

## Implementation note

This report covers **real parse/analyze runs** across 100 OSS repositories and, when `roundtrip-summary.csv` is present, matching round-trip checks over the same selected target directories.

The round-trip harness currently has two modes:

- `lossless`: parse first, then write the original source bytes as the rewritten file and compare formatter-normalized output. This isolates parser coverage and file rewrite/format stability.
- `header`: rebuild only package/import headers from parsed metadata, then compare. This is useful for dependency-header validation but is stricter than the current source model can satisfy for every real-world file.
- A general AST pretty-printer for full Java/Kotlin source reconstruction is still not implemented; exact results below should therefore be interpreted as lossless round-trip coverage, not full AST serialization parity.

## Aggregate Summary

- Projects scanned: **{overall_stats['projects']}**
- Completed with `ok`: **{overall_stats['status_counts'].get('ok', 0)}**
- Timed out: **{overall_stats['status_counts'].get('timeout', 0)}**
- Runtime errors: **{overall_stats['status_counts'].get('runtime-error', 0)}**
- Total files seen: **{overall_stats['total_files']}**
- Parsed files: **{overall_stats['parsed_files']}**
- Failed files: **{overall_stats['failed_files']}**
- Aggregate parse success: **{overall_stats['success_rate']:.2f}%**
- Total dependency edges reported: **{overall_stats['dependency_count']}**
- Total unresolved imports reported: **{overall_stats['unresolved_count']}**

## Duration Summary

- Total parse time across all projects: **{format_seconds(overall_stats['total_duration'])}**
- Average per project: **{format_seconds(overall_stats['avg_duration'])}**
- Median per project: **{format_seconds(overall_stats['median_duration'])}**
- P95 per project: **{format_seconds(overall_stats['p95_duration'])}**
- Max per project: **{format_seconds(overall_stats['max_duration'])}**

{roundtrip_section}

## By Language Group

{render_summary_table(language_stats)}

## Slowest Projects

{render_table(top_slowest)}

## Lowest Parse Success Projects

{render_table(worst_parse)}

## Full Project Matrix

### Java-based projects

{render_table(sorted(by_language.get("java", []), key=lambda row: row["name"].lower()))}

### Kotlin-based projects

{render_table(sorted(by_language.get("kotlin", []), key=lambda row: row["name"].lower()))}

## Next Step

- Drive strict parse failures and round-trip findings to zero across the full selected target set.
- Replace lossless rewrite with full source serialization only after the parser metadata is rich enough to preserve all syntax and trivia.
- After round-trip correctness is stable, capture CPU and heap profiles on the slowest projects above.
- Optimize parser hot paths and allocation-heavy graph construction paths before widening the benchmark set.
"""
    output_path.write_text(doc, encoding="utf-8")


if __name__ == "__main__":
    main()
