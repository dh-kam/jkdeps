# Checkpoint 2026-03-26

## Current Focus

- Clean architecture / SRP / interface-oriented refactoring
- Keep behavior stable while reducing responsibility concentration in `cmd/jkdeps` and `kotlin-compiler-golang`
- Use small, testable refactors and verify in tmux session `jkdeps-output`

## Completed Since Last Major Milestone

- 50 Java + 50 Kotlin OSS parsing batch completed
- Parsing report generated: [50-projects-parsing-result.md](/workspace/jkdeps/docs/50-projects-parsing-result.md)
- `jkdeps roundtrip-check` added and improved
- Representative top-10 roundtrip subset mostly stabilized

## Roundtrip Status

- Current top-10 subset result file: `/tmp/jkdeps-roundtrip-top10-results.csv`
- 9 projects are `unsupported=0`
- Remaining known gap:
  - `Kotlin/kotlinx.coroutines`
  - `pass=102 diff=0 parse_failed=0 unsupported=9 format_error=0`
  - Current assessment: likely formatter-less comparison limitation, not parser failure

## Refactors Completed

### `cmd/jkdeps`

- Formatter responsibility split from `roundtrip.go`
  - [roundtrip_formatter.go](/workspace/jkdeps/cmd/jkdeps/roundtrip_formatter.go)
- Header rewrite / semantic compare split from `roundtrip.go`
  - [roundtrip_header.go](/workspace/jkdeps/cmd/jkdeps/roundtrip_header.go)
- Roundtrip evaluation policy split from `roundtrip.go`
  - [roundtrip_evaluator.go](/workspace/jkdeps/cmd/jkdeps/roundtrip_evaluator.go)

### `kotlin-compiler-golang`

- ANTLR normalization split from `compiler.go`
  - [compiler_antlr_normalize.go](/workspace/jkdeps/kotlin-compiler-golang/compiler_antlr_normalize.go)
- Rule variant candidate generation split from `compiler.go`
  - [compiler_variant_candidates.go](/workspace/jkdeps/kotlin-compiler-golang/compiler_variant_candidates.go)
- Parse timeout policy split from `compiler.go`
  - [compiler_parse_timeout.go](/workspace/jkdeps/kotlin-compiler-golang/compiler_parse_timeout.go)
- Repository parse orchestration split from `compiler.go`
  - [compiler_repository_parse.go](/workspace/jkdeps/kotlin-compiler-golang/compiler_repository_parse.go)
- Source analysis split from `compiler.go`
  - [compiler_source_analysis.go](/workspace/jkdeps/kotlin-compiler-golang/compiler_source_analysis.go)
- `runTest` trailing-lambda rewrite rule extracted to pure helper
  - [compiler.go](/workspace/jkdeps/kotlin-compiler-golang/compiler.go)

### `scripts`

- Subset runner helper split from shell script
  - [roundtrip_subset_helper.py](/workspace/jkdeps/scripts/roundtrip_subset_helper.py)
  - [run_roundtrip_subset.sh](/workspace/jkdeps/scripts/run_roundtrip_subset.sh)

## Validation Status

- Verified in tmux session `jkdeps-output` only
- Latest repeated checks passed:
  - `go test ./cmd/jkdeps`
  - `go test ./kotlin-compiler-golang`

## Remaining Priorities

1. Continue shrinking large normalization functions in `kotlin-compiler-golang/compiler.go`
2. Re-evaluate whether more `roundtrip` policy abstraction is useful or over-engineering
3. Decide whether to leave `kotlinx.coroutines unsupported=9` as a formatter limitation
4. After correctness boundary is stable, move to CPU/heap profiling for slow projects

## Likely Next Step

- Continue extracting pure helpers from trailing-lambda normalization logic
- Add focused tests per helper
- Re-run `go test ./kotlin-compiler-golang` in `jkdeps-output`
