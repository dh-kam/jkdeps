#!/usr/bin/env python3
import csv
import json
import sys
from pathlib import Path


def lookup_target(targets_path: str, project: str) -> int:
    with open(targets_path, newline="") as fh:
        reader = csv.DictReader(fh)
        for row in reader:
            if row["name"] == project:
                print(row["language"])
                print(row["target_dir"])
                return 0
    return 1


def summarize_json(raw_json: str) -> int:
    data = json.loads(raw_json)
    print(
        ",".join(
            [
                str(data.get("passed_files", 0)),
                str(data.get("diff_files", 0)),
                str(data.get("parse_failed_files", 0)),
                str(data.get("unsupported_files", 0)),
                str(data.get("format_error_files", 0)),
            ]
        )
    )
    return 0


def main(argv: list[str]) -> int:
    if len(argv) < 2:
        print("usage: roundtrip_subset_helper.py <lookup-target|summarize-json> ...", file=sys.stderr)
        return 2

    command = argv[1]
    if command == "lookup-target":
        if len(argv) != 4:
            print("usage: roundtrip_subset_helper.py lookup-target <targets.csv> <project>", file=sys.stderr)
            return 2
        return lookup_target(argv[2], argv[3])
    if command == "summarize-json":
        if len(argv) != 3:
            print("usage: roundtrip_subset_helper.py summarize-json <json>", file=sys.stderr)
            return 2
        return summarize_json(argv[2])

    print(f"unknown command: {command}", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
