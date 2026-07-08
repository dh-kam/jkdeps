# ANTLR4 Go Runtime Fixes

## Scope

This document tracks local fixes applied to the forked ANTLR4 Go runtime at [third_party/antlr4-go](/workspace/jkdeps/third_party/antlr4-go).

The goal is:

- identify real runtime bottlenecks from `pprof`
- add regression tests and microbenchmarks before changing a hotspot
- apply the smallest safe fix first
- re-run tests, benchmarks, and the real `fastjson2` parse workload

`go.mod` now points the project at the local fork:

```go
replace github.com/antlr4-go/antlr/v4 => ./third_party/antlr4-go
```

## Target Workload

Real-world measurement target:

- repository: `alibaba/fastjson2`
- path: `core/src/main/java/com/alibaba/fastjson2/writer`
- command:

```bash
/tmp/jkdeps-prof-out/jkdeps smoke-parse \
  --repo /tmp/jkdeps-prof-src/alibaba_fastjson2/core/src/main/java/com/alibaba/fastjson2/writer \
  --workers 20 \
  --java-grammar java20 \
  --max-errors 5
```

Baseline observations from `pprof` before runtime-fork fixes:

- CPU hotspots:
  - `runtime.findObject`
  - `runtime.scanobject`
  - `github.com/antlr4-go/antlr/v4.(*ParserATNSimulator).closureWork`
- memory hotspots:
  - `github.com/antlr4-go/antlr/v4.NewATNConfig`
  - `github.com/antlr4-go/antlr/v4.(*JMap...).Put`
  - `github.com/antlr4-go/antlr/v4.NewBaseSingletonPredictionContext`
  - `github.com/antlr4-go/antlr/v4.mergeArrays`

This confirmed that the main issue is allocation churn inside the parser runtime, not just top-level jkdeps orchestration.

## Fix 1: `jcollect.go`

Files:

- [jcollect.go](/workspace/jkdeps/third_party/antlr4-go/jcollect.go)
- [jcollect_benchmark_test.go](/workspace/jkdeps/third_party/antlr4-go/jcollect_benchmark_test.go)

### Reason

`pprof` showed repeated cost in `JMap.Put`, `JMap.Get`, `JStore.Put`, and the GC paths they trigger. The original implementation stored `*entry` pointers and started internal maps with tiny capacities.

### Test and Benchmark First

Added regression tests:

- `TestJMapPutGetDeleteWithHashCollisions`
- `TestJStorePutGetWithHashCollisions`

Added microbenchmarks:

- `BenchmarkJMapPut_Unique`
- `BenchmarkJMapPut_SharedBucket`
- `BenchmarkJStorePut_SharedBucket`

Baseline:

| Benchmark | Before |
| --- | ---: |
| `BenchmarkJMapPut_Unique` | `97.12 ns/op`, `368 B/op`, `4 allocs/op` |
| `BenchmarkJMapPut_SharedBucket` | `120298 ns/op`, `63 B/op`, `1 allocs/op` |
| `BenchmarkJStorePut_SharedBucket` | `117244 ns/op`, `96 B/op`, `0 allocs/op` |

### Change

- changed `JMap` bucket storage from `[]*entry` to `[]entry`
- raised initial map capacity from `1` to `8`
- fixed preallocation in `Values` and `SortedSlice` to use logical length instead of bucket count

### Result

After change:

| Benchmark | After | Delta |
| --- | ---: | ---: |
| `BenchmarkJMapPut_Unique` | `85.81 ns/op`, `360 B/op`, `3 allocs/op` | faster, `-1 alloc/op` |
| `BenchmarkJMapPut_SharedBucket` | `120931 ns/op`, `131 B/op`, `0 allocs/op` | alloc count removed |
| `BenchmarkJStorePut_SharedBucket` | `116584 ns/op`, `78 B/op`, `0 allocs/op` | less memory |

Real workload impact:

- `fastjson2` wall time improved from about `42.92s` to about `39.73s`

This was the first fork change that produced a visible end-to-end win.

## Fix 2: `atn_config.go`

Files:

- [atn_config.go](/workspace/jkdeps/third_party/antlr4-go/atn_config.go)
- [atn_config_benchmark_test.go](/workspace/jkdeps/third_party/antlr4-go/atn_config_benchmark_test.go)

### Reason

After the `jcollect` fix, the largest remaining memory hotspot was `NewATNConfig`. The next safest change was to reduce object size without changing parse semantics.

### Test and Benchmark First

Added regression tests:

- `TestNewATNConfig4CopiesParserFields`
- `TestNewATNConfig5InitializesParserFields`

Added microbenchmarks:

- `BenchmarkNewATNConfig4`
- `BenchmarkNewATNConfig5`
- `BenchmarkATNConfigSize`

Baseline:

| Benchmark | Before |
| --- | ---: |
| `BenchmarkNewATNConfig4` | `25.33 ns/op`, `96 B/op`, `1 allocs/op` |
| `BenchmarkNewATNConfig5` | `24.51 ns/op`, `96 B/op`, `1 allocs/op` |
| `BenchmarkATNConfigSize` | `88 bytes/config` |

### Change

- reordered `ATNConfig` fields to reduce padding
- changed `cType` from `int` to `uint8`
- replaced field-by-field constructor writes with direct struct literals in parser-config constructors

### Result

After change:

| Benchmark | After | Delta |
| --- | ---: | ---: |
| `BenchmarkNewATNConfig4` | `24.16 ns/op`, `80 B/op`, `1 allocs/op` | less memory |
| `BenchmarkNewATNConfig5` | `23.33 ns/op`, `80 B/op`, `1 allocs/op` | less memory |
| `BenchmarkATNConfigSize` | `72 bytes/config` | `-16 bytes/config` |

Real workload impact:

- `fastjson2` wall time after this change was about `40.01s`

This means the object-size reduction is real and useful, but its end-to-end gain is small compared with `jcollect`. The dominant runtime cost is still deeper in parser simulation.

## Current Status

Latest `fastjson2` measurement with the forked runtime:

- parsed files: `118/118`
- failures: `0`
- `smoke-parse` wall time: about `33.63s`
- `deps` wall time on the same source set: about `46.53s`

Latest profile after the two fixes still shows these main hotspots:

- `github.com/antlr4-go/antlr/v4.(*ParserATNSimulator).closureWork`
- `github.com/antlr4-go/antlr/v4.NewATNConfig`
- `github.com/antlr4-go/antlr/v4.(*JMap...).Put` for prediction-context caching
- `github.com/antlr4-go/antlr/v4.NewBaseSingletonPredictionContext`
- GC paths caused by those allocations

## Next Candidates

Priority order:

1. `PredictionContext` allocation and caching path
2. `ParserATNSimulator.closureWork`
3. merge helpers such as `mergeArrays`

The same process should be followed for each:

1. add focused test
2. add benchmark
3. change runtime logic
4. rerun benchmark
5. rerun jkdeps parser tests
6. rerun `fastjson2`

## Notes

- Temporary GC tuning exists in jkdeps for measurement support, but it is not the primary fix strategy.
- The preferred direction is runtime logic and allocation reduction first, then GC tuning only if it still helps after structural fixes.

## Rejected Experiment: `PredictionContext` Size Reduction

Files kept for measurement:

- [prediction_context_benchmark_test.go](/workspace/jkdeps/third_party/antlr4-go/prediction_context_benchmark_test.go)

Reason:

- `NewBaseSingletonPredictionContext` and `NewArrayPredictionContext` still showed up high in `pprof`

Test and benchmark first:

- `TestSingletonBasePredictionContextCreateReturnsSharedEmpty`
- `TestNewBaseSingletonPredictionContextInitializesFields`
- `TestNewArrayPredictionContextInitializesFields`
- `BenchmarkNewBaseSingletonPredictionContext`
- `BenchmarkNewArrayPredictionContext`
- `BenchmarkPredictionContextSize`

Measured baseline:

| Benchmark | Before |
| --- | ---: |
| `BenchmarkNewBaseSingletonPredictionContext` | `24.76 ns/op`, `80 B/op`, `1 allocs/op` |
| `BenchmarkNewArrayPredictionContext` | `27.04 ns/op`, `80 B/op`, `1 allocs/op` |
| `BenchmarkPredictionContextSize` | `80 bytes/context` |

Attempted change:

- shrink internal scalar fields in `PredictionContext`

Observed result:

- object size could be reduced to `72 bytes/context`
- allocator class stayed at `80 B/op`
- real `fastjson2` wall time did not improve

Decision:

- reverted the runtime logic change
- kept the benchmark/test coverage for future experiments

This was a useful negative result: smaller struct size alone is not enough when the allocator class and merge churn stay the same.

## Fix 3: `JPCMap` Merge Cache

Files:

- [jcollect.go](/workspace/jkdeps/third_party/antlr4-go/jcollect.go)
- [jpcmap_benchmark_test.go](/workspace/jkdeps/third_party/antlr4-go/jpcmap_benchmark_test.go)

### Reason

`ParserATNSimulator` uses `mergeCache` on the hot path, and the default `JPCMap` implementation stored a nested `JMap` of `JMap`s. That created extra map objects and extra lookups for every `Put`.

### Test and Benchmark First

Added regression tests:

- `TestJPCMapPutAndGet`
- `TestJPCMapDuplicatePutDoesNotReplace`

Added microbenchmarks:

- `BenchmarkJPCMapPutDistinctPairs`
- `BenchmarkJPCMapGetHit`

Baseline:

| Benchmark | Before |
| --- | ---: |
| `BenchmarkJPCMapPutDistinctPairs` | `788.3 ns/op`, `584 B/op`, `5 allocs/op` |
| `BenchmarkJPCMapGetHit` | `15.02 ns/op`, `0 B/op`, `0 allocs/op` |

### Change

- replaced nested `JMap -> JMap` storage inside `JPCMap`
- switched to a flat bucket map keyed by `dHash(k1, k2)`
- kept the existing `JPCMap` public API so parser code did not need to change

### Result

After change:

| Benchmark | After | Delta |
| --- | ---: | ---: |
| `BenchmarkJPCMapPutDistinctPairs` | `318.4 ns/op`, `179 B/op`, `1 allocs/op` | major improvement |
| `BenchmarkJPCMapGetHit` | `8.534 ns/op`, `0 B/op`, `0 allocs/op` | faster |

Real workload impact:

- `fastjson2` wall time improved from about `39.73s` to about `33.63s`

This is the largest runtime-fork win after the initial `jcollect` change.

## Architectural Fix: Unify Product Parsing Strategy

Files:

- [internal/mixedgraph/parser.go](/workspace/jkdeps/internal/mixedgraph/parser.go)
- [internal/mixedgraph/parser_benchmark_test.go](/workspace/jkdeps/internal/mixedgraph/parser_benchmark_test.go)

### Reason

There was a structural mismatch between goals and implementation:

- `smoke-parse` used the shared Java parser path with `SLL -> LL fallback`
- `deps` and `graph` went through `mixedgraph`, which still used a separate `LL-only` Java validation path

That meant product commands were always paying the most expensive parsing strategy, even though the project already had a cheaper validated path for most files.

### Test and Benchmark First

Added comparison benchmarks in `internal/mixedgraph`:

- `BenchmarkParseJavaSourceStrategies`
- `BenchmarkParseJavaSourceStrategiesRealFile`

Measured on a representative real Java file from `fastjson2`:

| Benchmark | Current mixedgraph LL-only | Shared fallback |
| --- | ---: | ---: |
| `BenchmarkParseJavaSourceStrategiesRealFile` | `42.03s/op`, `32.86 GB/op`, `461.9M allocs/op` | `12.45s/op`, `11.73 GB/op`, `166.0M allocs/op` |

This confirmed the structural issue: for expensive files, the product path was using the wrong algorithm.

### Change

- removed the duplicate Java LL-only parse implementation from `mixedgraph`
- routed `mixedgraph` Java diagnostics through `internal/parser.ParseWithDiagnostics`
- kept existing strict/lenient semantics and unsupported-grammar diagnostics

### Result

Verification:

- `go test ./internal/mixedgraph`
- `go test ./cmd/jkdeps`

Real workload impact on the same `fastjson2/writer` target:

- `deps` wall time improved from about `48.21s` to about `46.53s`

Interpretation:

- the change is correct and high leverage for expensive individual files
- the repository-wide improvement is smaller because the workload mixes a few pathological files with many ordinary files
- this means the next iteration should optimize the remaining heavy files and the parser runtime paths they exercise most, not just the average file path

## Architectural Fix: Separate Dependency Extraction From Full Java Validation

Files:

- [internal/mixedgraph/types.go](/workspace/jkdeps/internal/mixedgraph/types.go)
- [internal/mixedgraph/parser.go](/workspace/jkdeps/internal/mixedgraph/parser.go)
- [internal/mixedgraph/parser_benchmark_test.go](/workspace/jkdeps/internal/mixedgraph/parser_benchmark_test.go)
- [cmd/jkdeps/parse_flags.go](/workspace/jkdeps/cmd/jkdeps/parse_flags.go)
- [cmd/jkdeps/deps.go](/workspace/jkdeps/cmd/jkdeps/deps.go)

### Reason

The new top-N slow file view changed the diagnosis.

For `fastjson2/writer`, the slowest files were:

- `ObjectWriterCreatorASM.java` about `46.2s`
- `ObjectWriterCreator.java` about `28.5s`
- `ObjectWriterBaseModule.java` about `22.8s`
- `FieldWriter.java` about `21.1s`

These are large Java implementation files, not header-heavy files. The important observation is that `deps` and `graph` only use:

- package name
- import list
- parse status / diagnostics summary

The dependency report itself is built from extracted headers, not from full Java bodies. That means full-compilation-unit validation in `deps`/`graph` is not the minimal algorithm for the job. It is a validation concern mixed into a dependency-extraction pipeline.

### Test and Benchmark First

Added:

- `JavaParseMode` with `full` and `header-only`
- regression tests proving header-only still extracts package/imports and preserves dependency report output
- real-file benchmark comparing `parseJavaSource` modes on the slowest file

Measured on:

- `/tmp/jkdeps-prof-src/alibaba_fastjson2/core/src/main/java/com/alibaba/fastjson2/writer/ObjectWriterCreatorASM.java`

| Benchmark | Result |
| --- | ---: |
| `BenchmarkParseJavaSourceModesRealFile/full_validation` | `43.77s/op`, `33.07 GB/op`, `464.8M allocs/op` |
| `BenchmarkParseJavaSourceModesRealFile/header_only` | `3.31ms/op`, `209.9 KB/op`, `59 allocs/op` |

This is not a runtime micro-optimization. It shows that the dominant cost came from running the wrong algorithm for the product goal.

### Change

- added `--java-parse-mode full|header-only` for `deps`/`graph`
- kept `full` as the default to preserve existing strict semantics
- in `header-only` mode, Java files skip full-body validation and use header extraction only
- preserved the existing dependency-report path and JSON output shape, with `java_parse_mode` included in `deps` JSON

### Result

Real workload on the same `fastjson2/writer` target:

| Mode | Duration | Parsed | Failed | Dependencies | Unresolved |
| --- | ---: | ---: | ---: | ---: | ---: |
| `full` | `51.912317765s` | `118` | `0` | `788` | `357` |
| `header-only` | `0.006530606s` | `118` | `0` | `788` | `357` |

Verification:

- `diff -u <(jq -S "{file_dependencies, unresolved_imports}" full.json) <(jq -S "{file_dependencies, unresolved_imports}" header.json)` returned no diff
- `go test ./internal/mixedgraph`
- `go test ./cmd/jkdeps`

Interpretation:

- the main bottleneck was not only ANTLR runtime allocation
- a more fundamental issue was that `deps`/`graph` were doing full Java validation even when the downstream product only needed headers
- this gives a clean separation of concerns:
  - `smoke-parse` or explicit `full` mode for syntax validation
  - `header-only` mode for dependency extraction at production speed

### Follow-up Decision

After additional checks on Guava modules:

| Target | Full | Header-only | Dependency diff |
| --- | ---: | ---: | --- |
| `guava/src/com/google/common/base` | `6.409694239s` | `0.002896987s` | none |
| `guava/src/com/google/common/util/concurrent` | `15.563015007s` | `0.005161671s` | none |

The project now defaults `jkdeps deps` to `--java-parse-mode header-only`.

Scope:

- `deps` now optimizes for dependency extraction by default
- explicit validation remains available with `--java-parse-mode full`
- `graph` remains on `full` for now because its existing parse/fail summary semantics and tests still assume validation-oriented behavior
- `scripts/oss_dependency_matrix.sh` now passes `--java-parse-mode full` explicitly so historical parse/fail reporting stays comparable

## Follow-up: `full` Mode Now Extracts Java Signature References

Files:

- [internal/parser/java_ast_builder.go](/workspace/jkdeps/internal/parser/java_ast_builder.go)
- [internal/parser/java_type_reference.go](/workspace/jkdeps/internal/parser/java_type_reference.go)
- [internal/mixedgraph/reference_extractor.go](/workspace/jkdeps/internal/mixedgraph/reference_extractor.go)
- [internal/mixedgraph/reference_resolver.go](/workspace/jkdeps/internal/mixedgraph/reference_resolver.go)
- [internal/mixedgraph/dependency_report.go](/workspace/jkdeps/internal/mixedgraph/dependency_report.go)

Reason:

- `header-only` is the right default for fast import-based analysis.
- But richer dependency analysis needs more than imports:
  - `extends`
  - `implements`
  - field types
  - method return types
  - method parameter types
  - constructor parameter types
  - `throws` clauses

Change:

- added a Java20 listener-based AST builder for declaration/signature collection
- flattened Java declarations into typed references with explicit `reference_kind`
- taught `mixedgraph` to resolve simple type names against:
  - explicit imports
  - wildcard imports
  - `java.lang`
  - same-package fallback
- kept `header-only` unchanged; typed references are populated only in `full` mode

Verification:

- `go test ./internal/parser`
- `go test ./internal/mixedgraph`
- `go test ./internal/parser ./internal/mixedgraph ./cmd/jkdeps`

Interpretation:

- the project now has two distinct Java dependency modes
- `header-only` remains the production-speed import graph path
- `full` is now meaningfully richer, not just syntax validation
