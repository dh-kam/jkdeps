# kotlin-compiler-golang Porting Completion (Baseline)

Validated on: **February 24, 2026**

## Baseline artifacts

- Output directory: `/tmp/jkdeps-porting-final13`
- Sample repository refs:
  - `guava`: `9857e70cf51a341ebb41dd2f0b8d3354f6a9d869`
  - `kotlinx.coroutines`: `b11abdf01d4d5db85247ab365abc72efc7b95062`
- Full run command:
  - `RUN_GUAVA_STRESS_STRICT=1 RUN_KOTLIN_JVM=1 RUN_KOTLIN_CORE=1 RUN_KOTLIN_OFFICIAL_PARITY=1 KOTLIN_OFFICIAL_PARITY_MAX_PARSE_STATUS_MISMATCH=0 ./scripts/porting_acceptance.sh /tmp/jkdeps-samples /tmp/jkdeps-porting-final13`
- Completion check command:
  - `./scripts/verify_porting_completion.sh /tmp/jkdeps-porting-final13`
- Aggregate audit command:
  - `PORTING_OUT_DIR=/tmp/jkdeps-porting-final13 BASELINE_FILE_PATH=./docs/porting-baseline.json make porting-audit`

## Result

- `Porting completion check PASSED`
- Machine-readable regression baseline: `docs/porting-baseline.json`
  - includes pinned `sample_refs` and `runtime_inventory.sha256`
- Baseline comparison report outputs: `porting-baseline-compare.md`, `porting-baseline-compare.json`
- Aggregate audit report output: `porting-audit.json` (`porting_audit.sh` / `make porting-audit`)
- Run metadata artifact: `porting-run-metadata.json` (sample refs + inventory hash)
- Optional verify JSON output: `porting-completion-verify.json` (`verify_porting_completion.sh --json-out`)
- Baseline refresh command: `./scripts/update_porting_baseline.sh /tmp/jkdeps-porting-final13 ./docs/porting-baseline.json`

## Key metrics snapshot

- `kotlin-common-acceptance.json`
  - parse: `total=111 parsed=111 failed=0`
  - diagnostics: `diag_files=0 total=0`
  - resolve: `unresolved_imports=0`
- `kotlin-js-acceptance-lenient.json`
  - parse: `total=7 parsed=7 failed=0`
  - diagnostics: `diag_files=0 total=0`
  - resolve: `unresolved_imports=0`
- `kotlin-jvm-acceptance-lenient.json`
  - parse: `total=51 parsed=51 failed=0`
  - diagnostics: `diag_files=0 total=0`
  - resolve: `unresolved_imports=0`
- `kotlin-core-acceptance-lenient.json`
  - parse: `total=698 parsed=698 failed=0`
  - diagnostics: `diag_files=12 total=70`
  - resolve: `unresolved_imports=0`
- `kotlin_core_mixed_graph_lenient.log`
  - mixed parse: `total=699 java=1 kotlin=698`
  - parse status: `parsed=699 failed=0`
- `kotlin_core_mixed_dir_graph_lenient.log`
  - mixed parse: `total=699 java=1 kotlin=698`
  - parse status: `parsed=699 failed=0`
  - graph: `nodes=141 edges=2902`
- `kotlin-official-parity.json`
  - status: `PASSED`
  - compared: `official=7`, `go=7`, `files_compared=7`
  - mismatches: `parse=0` (allowed `<=0`), `package=0`, `imports=0`, `declarations=0`

## Gate summary

- Guava strict parse/graph gates: passed
- Kotlin common/js/jvm/core acceptance + resolve gates: passed
- Kotlin core mixed Java+Kotlin package/dir graph gates: passed (`java>0`, `kotlin>0`, `failed=0`)
- Kotlin official PSI parity gate: passed (strict zero-mismatch default)
