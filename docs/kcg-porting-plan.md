# kotlin-compiler-golang Porting Plan

## Scope

Target: port `kotlin-compiler-embeddable` behaviors into pure Go over multiple milestones.

## Milestones

1. Bootstrap parser, AST-lite, and repository scanner in Go.
2. Port lexer/token model and stable AST layer.
3. Port symbol table, scope, and type resolver core.
4. Port diagnostics and frontend checks.
5. Validate against Kotlin compiler test data and compatibility fixtures.

## Current status

- Milestone 1 is initialized.
- `kotlin-compiler-golang` package provides parse APIs, diagnostics, top-level declarations, symbol table generation, package-level dependency graph generation, and import resolution (`resolve`) with optional multi-inventory overlay (repeatable `--inventory`).
- `kotlin-compiler-golang` supports `LenientSyntax` mode to keep analysis moving even when grammar mismatches occur; diagnostics are retained.
- `kotlin-compiler-golang` provides `graph` command to emit web-viewable dependency artifacts (`.json` + `.html`) with internal/external/unknown node typing.
- `ktcg-inventory` provides package/symbol inventory from JVM JAR and Kotlin metadata archives (including `.kotlin_builtins`, `.kjsm`, `.knm`, `.kotlin_metadata`) to guide port order and external import resolution.
- Runtime inventory scripts cover `kotlin-compiler-embeddable`, `kotlin-stdlib-js`, and `kotlinx-browser` overlays for practical JS import resolution.
- `cmd/kotlin-compiler-golang` CLI 정합성 정리(서브커맨드 플래그, help alias 처리, parse/acceptance/compile 경로 플로우)가 완료되어, 디버그/테스트 헬퍼의 샘플 경로는 `KCG_SAMPLE_ROOT`로 재매핑 가능.
- Completion gates are automated in `scripts/porting_acceptance.sh` + `scripts/verify_porting_completion.sh`:
  - strict Java parse/graph gates on Guava modules
  - Kotlin common/js/jvm/core acceptance + resolve gates (`failed_files=0`, `unresolved_imports=0`, diagnostics budgets)
  - mixed Java+Kotlin graph gate on `kotlinx-coroutines-core` root (`java>0`, `kotlin>0`, `failed=0`)
  - optional official Kotlin PSI parity gate on `kotlinx-coroutines-core/js/src` (`scripts/kcg_official_parity.sh`)
