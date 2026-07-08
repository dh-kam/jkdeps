#!/usr/bin/env python3
import argparse
import os
import tempfile
from collections import Counter
from pathlib import Path


SKIP_DIRS = {".git", "build", "out", "target", "node_modules", ".gradle", ".idea", ".konan"}


def candidate_root(rel_path: Path) -> Path:
    parts = rel_path.parts
    try:
        src_idx = parts.index("src")
        root = list(parts[: src_idx + 1])
        idx = src_idx + 1
        while idx < len(parts) - 1 and (parts[idx].endswith("Main") or parts[idx].endswith("Test")):
            root.append(parts[idx])
            idx += 1
        if idx < len(parts) - 1 and parts[idx] in {"java", "kotlin"}:
            root.append(parts[idx])
        return Path(*root)
    except ValueError:
        parent = rel_path.parent
        return parent if str(parent) != "." else Path(".")


def path_preference(path: Path) -> tuple[int, int]:
    text = str(path).lower()
    score = 0
    preferred_markers = (
        "src/main/java",
        "src/main/kotlin",
        "src/commonmain/java",
        "src/commonmain/kotlin",
        "src/jvmmain/java",
        "src/jvmmain/kotlin",
        "src/androidmain/java",
        "src/androidmain/kotlin",
        "src/iosmain/kotlin",
        "src/jsmain/kotlin",
        "src/wasmmain/kotlin",
        "src/nativemain/kotlin",
        "src/main",
        "src/commonmain",
        "src/jvmmain",
    )
    discouraged_markers = (
        "/test/",
        "src/test",
        "srctest",
        "androidtest",
        "jvmtest",
        "commontest",
        "integrationtest",
        "integration-test",
        "integration-testing",
        "/jmh/",
        "src/jmh",
        "smoketest",
        "smoke-test",
        "benchmark",
        "samples",
        "example",
        "examples",
    )
    if any(marker in text for marker in discouraged_markers):
        score += 1000
    elif any(marker in text for marker in preferred_markers):
        score -= 1000
    return score, len(path.parts)


def path_bucket(path: Path) -> int:
    text = str(path).lower()
    preferred_markers = (
        "src/main/java",
        "src/main/kotlin",
        "src/commonmain/java",
        "src/commonmain/kotlin",
        "src/jvmmain/java",
        "src/jvmmain/kotlin",
        "src/androidmain/java",
        "src/androidmain/kotlin",
        "src/iosmain/kotlin",
        "src/jsmain/kotlin",
        "src/wasmmain/kotlin",
        "src/nativemain/kotlin",
        "src/main",
        "src/commonmain",
        "src/jvmmain",
    )
    discouraged_markers = (
        "/test/",
        "src/test",
        "srctest",
        "androidtest",
        "jvmtest",
        "commontest",
        "integrationtest",
        "integration-test",
        "integration-testing",
        "/jmh/",
        "src/jmh",
        "smoketest",
        "smoke-test",
        "benchmark",
        "samples",
        "example",
        "examples",
    )
    if any(marker in text for marker in discouraged_markers):
        return 2
    if any(marker in text for marker in preferred_markers):
        return 0
    return 1


def choose_best_root(items):
    if not items:
        return None
    bucketed = {0: [], 1: [], 2: []}
    for path, count in items:
        bucketed[path_bucket(path)].append((path, count))
    for bucket in (0, 1, 2):
        if bucketed[bucket]:
            return min(
                bucketed[bucket],
                key=lambda item: (-item[1], *path_preference(item[0]), str(item[0])),
            )
    return min(items, key=lambda item: (-item[1], *path_preference(item[0]), str(item[0])))


def pick_preferred_oversize_subtree(items):
    preferred = [(path, count) for path, count in items if path_bucket(path) == 0]
    if not preferred:
        return None
    return min(
        preferred,
        key=lambda item: (item[1], *path_preference(item[0]), str(item[0])),
    )


def collect_files(repo: Path, lang: str):
    exts = {".java"} if lang == "java" else {".kt", ".kts"}
    files = []
    counts = Counter()
    for dirpath, dirnames, filenames in os.walk(repo):
        dirnames[:] = [name for name in dirnames if name not in SKIP_DIRS]
        current_dir = Path(dirpath)
        for filename in filenames:
            path = current_dir / filename
            if path.suffix.lower() not in exts:
                continue
            rel = path.relative_to(repo)
            files.append(rel)
            counts[candidate_root(rel)] += 1
    return files, counts


def rank_items(items):
    return sorted(
        items,
        key=lambda item: (path_bucket(item[0]), -item[1], *path_preference(item[0]), str(item[0])),
    )


def explain_selection(repo: Path, lang: str, max_files: int):
    files, counts = collect_files(repo, lang)
    if not counts:
        return {
            "selected": repo.resolve(),
            "best_root": repo.resolve(),
            "top_roots": [],
            "eligible_subtrees": [],
            "selection_source": "repo",
        }

    best_root, _ = choose_best_root(list(counts.items()))
    ranked_roots = rank_items(list(counts.items()))[:10]
    best_root_files = [
        rel for rel in files if str(rel).startswith(str(best_root) + os.sep) or rel.parent == best_root
    ]
    if len(best_root_files) <= max_files:
        return {
            "selected": (repo / best_root).resolve(),
            "best_root": (repo / best_root).resolve(),
            "top_roots": ranked_roots,
            "eligible_subtrees": [],
            "selection_source": "best_root",
        }

    subtree_counts = Counter()
    for rel in best_root_files:
        current = rel.parent
        while True:
            if current == best_root:
                break
            subtree_counts[current] += 1
            current = current.parent

    eligible = [(path, count) for path, count in subtree_counts.items() if count <= max_files]
    if eligible:
        preferred_eligible = [(path, count) for path, count in eligible if path_bucket(path) == 0]
        if preferred_eligible:
            selected, _ = choose_best_root(preferred_eligible)
            return {
                "selected": (repo / selected).resolve(),
                "best_root": (repo / best_root).resolve(),
                "top_roots": ranked_roots,
                "eligible_subtrees": rank_items(eligible)[:10],
                "selection_source": "eligible_subtree",
            }
    preferred_oversize = pick_preferred_oversize_subtree(list(subtree_counts.items()))
    if preferred_oversize is not None:
        selected, _ = preferred_oversize
        return {
            "selected": (repo / selected).resolve(),
            "best_root": (repo / best_root).resolve(),
            "top_roots": ranked_roots,
            "eligible_subtrees": [],
            "selection_source": "preferred_oversize_subtree",
        }
    if eligible:
        selected, _ = choose_best_root(eligible)
        return {
            "selected": (repo / selected).resolve(),
            "best_root": (repo / best_root).resolve(),
            "top_roots": ranked_roots,
            "eligible_subtrees": rank_items(eligible)[:10],
            "selection_source": "eligible_subtree",
        }
    return {
        "selected": (repo / best_root).resolve(),
        "best_root": (repo / best_root).resolve(),
        "top_roots": ranked_roots,
        "eligible_subtrees": [],
        "selection_source": "best_root_fallback",
    }


def select_root(repo: Path, lang: str, max_files: int) -> Path:
    return explain_selection(repo, lang, max_files)["selected"]


def touch_files(root: Path, rel_dir: str, count: int, suffix: str):
    target = root / rel_dir
    target.mkdir(parents=True, exist_ok=True)
    for idx in range(count):
        (target / f"Sample{idx}{suffix}").write_text("class Sample {}\n", encoding="utf-8")


def run_self_test() -> int:
    with tempfile.TemporaryDirectory(prefix="jkdeps-auto-target-") as tmp:
        root = Path(tmp)

        touch_files(root, "module/src/test/java/com/example", 90, ".java")
        touch_files(root, "module/src/main/java/com/example", 40, ".java")
        selected = select_root(root, "java", 120)
        expected = (root / "module/src/main").resolve()
        if selected != expected:
            print(f"FAIL prefer main over test: got {selected}, want {expected}")
            return 1

    with tempfile.TemporaryDirectory(prefix="jkdeps-auto-target-") as tmp:
        root = Path(tmp)

        touch_files(root, "lib/src/commonMain/kotlin/demo", 70, ".kt")
        touch_files(root, "lib/src/commonTest/kotlin/demo", 90, ".kt")
        selected = select_root(root, "kotlin", 120)
        expected = (root / "lib/src/commonMain/kotlin").resolve()
        if selected != expected:
            print(f"FAIL prefer commonMain over commonTest: got {selected}, want {expected}")
            return 1

    with tempfile.TemporaryDirectory(prefix="jkdeps-auto-target-") as tmp:
        root = Path(tmp)

        touch_files(root, "src/misc/java/demo/a", 30, ".java")
        touch_files(root, "src/misc/java/demo/b", 110, ".java")
        selected = select_root(root, "java", 120)
        expected = (root / "src/misc/java/demo/b").resolve()
        if selected != expected:
            print(f"FAIL pick largest eligible subtree: got {selected}, want {expected}")
            return 1

    print("self-test: PASS")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo")
    parser.add_argument("--lang", choices=("java", "kotlin"))
    parser.add_argument("--max-files", type=int, default=120)
    parser.add_argument("--explain", action="store_true")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()

    if args.self_test:
        return run_self_test()

    if not args.repo or not args.lang:
        parser.error("--repo and --lang are required unless --self-test is set")

    explanation = explain_selection(Path(args.repo), args.lang, args.max_files)
    if args.explain:
        print(f"selected={explanation['selected']}")
        print(f"selection_source={explanation['selection_source']}")
        print(f"best_root={explanation['best_root']}")
        if explanation["top_roots"]:
            print("top_roots:")
            for path, count in explanation["top_roots"]:
                print(
                    f"  - path={path} count={count} bucket={path_bucket(path)} preference={path_preference(path)}"
                )
        if explanation["eligible_subtrees"]:
            print("eligible_subtrees:")
            for path, count in explanation["eligible_subtrees"]:
                print(
                    f"  - path={path} count={count} bucket={path_bucket(path)} preference={path_preference(path)}"
                )
        return 0

    print(explanation["selected"])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
