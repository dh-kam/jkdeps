# 50 Java + 50 Kotlin Projects Parsing Result

Generated at: **2026-07-08 16:33 UTC**

## Scope

- Manifest: `scripts/oss_50_projects_targets.txt`
- Sources root: `/tmp/jkdeps-50-projects-sources`
- Raw outputs: `/tmp/jkdeps-50-projects-out`
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

- Projects scanned: **100**
- Completed with `ok`: **100**
- Timed out: **0**
- Runtime errors: **0**
- Total files seen: **7370**
- Parsed files: **7370**
- Failed files: **0**
- Aggregate parse success: **100.00%**
- Total dependency edges reported: **0**
- Total unresolved imports reported: **0**

## Duration Summary

- Total parse time across all projects: **1049.65s**
- Average per project: **10.50s**
- Median per project: **5.55s**
- P95 per project: **30.71s**
- Max per project: **76.33s**

## Round-Trip Summary

- Projects checked: **100**
- Completed with exact `ok`: **100**
- Projects with findings: **0**
- Runtime errors/timeouts: **0**
- Total files checked: **7370**
- Exact pass files: **7370**
- Diff files: **0**
- Parse-failed files: **0**
- Unsupported files: **0**
- Formatter error files: **0**
- Aggregate exact rate: **100.00%**
- Rewrite mode(s): **lossless**
- Java formatter command(s): **not configured/detected**
- Kotlin formatter command(s): **not configured/detected**

## Round-Trip Duration Summary

- Total round-trip time across all projects: **1568.53s**
- Average per project: **15.69s**
- Median per project: **8.07s**
- P95 per project: **53.68s**
- Max per project: **83.58s**
- Parser time inside round-trip command: **1488.45s**

## Round-Trip By Language Group

| Language Group | Projects | OK | Findings | Runtime Error | Timeout | Files | Pass | Diff | Parse Failed | Unsupported | Format Error | Exact | Total Time | Avg | Median | P95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| java | 50 | 50 | 0 | 0 | 0 | 4869 | 4869 | 0 | 0 | 0 | 0 | 100.00% | 1178.30s | 23.57s | 18.94s | 57.52s |
| kotlin | 50 | 50 | 0 | 0 | 0 | 2501 | 2501 | 0 | 0 | 0 | 0 | 100.00% | 390.24s | 7.80s | 3.94s | 22.35s |

## Round-Trip Findings

No round-trip findings were recorded.

## Slowest Round-Trip Projects

| Project | Status | Files | Pass | Diff | Parse Failed | Unsupported | Format Error | Exact | Duration | Target |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |

| `alibaba/fastjson2` | ok | 118 | 118 | 0 | 0 | 0 | 0 | 100.00% | 83.58s | `core/src/main/java/com/alibaba/fastjson2/writer` |
| `JetBrains/Exposed` | ok | 120 | 120 | 0 | 0 | 0 | 0 | 100.00% | 75.50s | `exposed-tests/src` |
| `alibaba/druid` | ok | 116 | 116 | 0 | 0 | 0 | 0 | 100.00% | 72.47s | `core/src/main/java/com/alibaba/druid/sql/dialect/oracle` |
| `FasterXML/jackson-databind` | ok | 111 | 111 | 0 | 0 | 0 | 0 | 100.00% | 59.24s | `src/main/java/tools/jackson/databind/deser` |
| `apache/shardingsphere` | ok | 1200 | 1200 | 0 | 0 | 0 | 0 | 100.00% | 55.43s | `test/it/parser/src/main` |
| `hibernate/hibernate-orm` | ok | 120 | 120 | 0 | 0 | 0 | 0 | 100.00% | 53.59s | `hibernate-core/src/main/java/org/hibernate/internal` |
| `spring-projects/spring-data-jpa` | ok | 97 | 97 | 0 | 0 | 0 | 0 | 100.00% | 53.14s | `spring-data-jpa/src/main/java/org/springframework/data/jpa/repository/query` |
| `apache/dubbo` | ok | 65 | 65 | 0 | 0 | 0 | 0 | 100.00% | 51.29s | `dubbo-common/src/main/java/org/apache/dubbo/common/utils` |
| `apache/commons-cli` | ok | 87 | 87 | 0 | 0 | 0 | 0 | 100.00% | 43.26s | `src` |
| `apache/commons-jexl` | ok | 56 | 56 | 0 | 0 | 0 | 0 | 100.00% | 41.17s | `src/main/java/org/apache/commons/jexl3/internal` |
| `google/error-prone` | ok | 113 | 113 | 0 | 0 | 0 | 0 | 100.00% | 38.17s | `core/src/main/java/com/google/errorprone/refaster` |
| `reactivestreams/reactive-streams-jvm` | ok | 30 | 30 | 0 | 0 | 0 | 0 | 100.00% | 37.32s | `tck/src` |
| `square/moshi` | ok | 64 | 64 | 0 | 0 | 0 | 0 | 100.00% | 34.35s | `moshi/src` |
| `apache/lucene` | ok | 108 | 108 | 0 | 0 | 0 | 0 | 100.00% | 33.22s | `lucene/core/src/java/org/apache/lucene/document` |
| `apache/commons-pool` | ok | 54 | 54 | 0 | 0 | 0 | 0 | 100.00% | 31.75s | `src/main` |


## By Language Group

| Language Group | Projects | OK | Timeout | Runtime Error | Files | Parsed | Failed | Parse Success | Total Time | Avg | Median | P95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| java | 50 | 50 | 0 | 0 | 4869 | 4869 | 0 | 100.00% | 723.97s | 14.48s | 11.84s | 36.28s |
| kotlin | 50 | 50 | 0 | 0 | 2501 | 2501 | 0 | 100.00% | 325.68s | 6.51s | 3.00s | 21.61s |

## Slowest Projects

| Project | Status | Files | Parsed | Failed | Parse Success | Duration | Target |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |

| `JetBrains/Exposed` | ok | 120 | 120 | 0 | 100.00% | 76.33s | `exposed-tests/src` |
| `alibaba/fastjson2` | ok | 118 | 118 | 0 | 100.00% | 45.76s | `core/src/main/java/com/alibaba/fastjson2/writer` |
| `alibaba/druid` | ok | 116 | 116 | 0 | 100.00% | 44.35s | `core/src/main/java/com/alibaba/druid/sql/dialect/oracle` |
| `hibernate/hibernate-orm` | ok | 120 | 120 | 0 | 100.00% | 37.01s | `hibernate-core/src/main/java/org/hibernate/internal` |
| `FasterXML/jackson-databind` | ok | 111 | 111 | 0 | 100.00% | 35.39s | `src/main/java/tools/jackson/databind/deser` |
| `apache/dubbo` | ok | 65 | 65 | 0 | 100.00% | 30.46s | `dubbo-common/src/main/java/org/apache/dubbo/common/utils` |
| `reactivestreams/reactive-streams-jvm` | ok | 30 | 30 | 0 | 100.00% | 28.90s | `tck/src` |
| `spring-projects/spring-data-jpa` | ok | 97 | 97 | 0 | 100.00% | 28.66s | `spring-data-jpa/src/main/java/org/springframework/data/jpa/repository/query` |
| `google/error-prone` | ok | 113 | 113 | 0 | 100.00% | 26.70s | `core/src/main/java/com/google/errorprone/refaster` |
| `apache/commons-jexl` | ok | 56 | 56 | 0 | 100.00% | 26.46s | `src/main/java/org/apache/commons/jexl3/internal` |
| `apache/shardingsphere` | ok | 1200 | 1200 | 0 | 100.00% | 25.75s | `test/it/parser/src/main` |
| `apache/commons-cli` | ok | 87 | 87 | 0 | 100.00% | 24.93s | `src` |
| `skydoves/Balloon` | ok | 59 | 59 | 0 | 100.00% | 23.28s | `balloon/src` |
| `square/moshi` | ok | 64 | 64 | 0 | 100.00% | 23.22s | `moshi/src` |
| `google/gson` | ok | 87 | 87 | 0 | 100.00% | 19.95s | `gson/src/main` |

## Lowest Parse Success Projects

| Project | Status | Files | Parsed | Failed | Parse Success | Duration | Target |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |

| `google/guava` | ok | 84 | 84 | 0 | 100.00% | 18.57s | `guava/src/com/google/common/util` |
| `google/gson` | ok | 87 | 87 | 0 | 100.00% | 19.95s | `gson/src/main` |
| `square/retrofit` | ok | 57 | 57 | 0 | 100.00% | 11.77s | `retrofit/src` |
| `reactivestreams/reactive-streams-jvm` | ok | 30 | 30 | 0 | 100.00% | 28.90s | `tck/src` |
| `apache/commons-io` | ok | 69 | 69 | 0 | 100.00% | 9.39s | `src/main/java/org/apache/commons/io/input` |
| `apache/commons-codec` | ok | 87 | 87 | 0 | 100.00% | 16.87s | `src/main` |
| `apache/commons-collections` | ok | 55 | 55 | 0 | 100.00% | 1.98s | `src/main/java/org/apache/commons/collections4/functors` |
| `apache/commons-lang` | ok | 62 | 62 | 0 | 100.00% | 1.06s | `src/main/java/org/apache/commons/lang3/function` |
| `apache/commons-text` | ok | 112 | 112 | 0 | 100.00% | 14.70s | `src/main` |
| `apache/commons-csv` | ok | 55 | 55 | 0 | 100.00% | 17.09s | `src` |
| `apache/commons-compress` | ok | 101 | 101 | 0 | 100.00% | 15.08s | `src/main/java/org/apache/commons/compress/harmony/unpack200` |
| `apache/commons-cli` | ok | 87 | 87 | 0 | 100.00% | 24.93s | `src` |
| `apache/commons-configuration` | ok | 63 | 63 | 0 | 100.00% | 5.89s | `src/main/java/org/apache/commons/configuration2/builder` |
| `apache/commons-dbcp` | ok | 68 | 68 | 0 | 100.00% | 11.91s | `src/main` |
| `apache/commons-pool` | ok | 54 | 54 | 0 | 100.00% | 17.56s | `src/main` |

## Full Project Matrix

### Java-based projects

| Project | Status | Files | Parsed | Failed | Parse Success | Duration | Target |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |

| `alibaba/druid` | ok | 116 | 116 | 0 | 100.00% | 44.35s | `core/src/main/java/com/alibaba/druid/sql/dialect/oracle` |
| `alibaba/fastjson2` | ok | 118 | 118 | 0 | 100.00% | 45.76s | `core/src/main/java/com/alibaba/fastjson2/writer` |
| `apache/commons-beanutils` | ok | 46 | 46 | 0 | 100.00% | 4.71s | `src/main/java/org/apache/commons/beanutils2/converters` |
| `apache/commons-cli` | ok | 87 | 87 | 0 | 100.00% | 24.93s | `src` |
| `apache/commons-codec` | ok | 87 | 87 | 0 | 100.00% | 16.87s | `src/main` |
| `apache/commons-collections` | ok | 55 | 55 | 0 | 100.00% | 1.98s | `src/main/java/org/apache/commons/collections4/functors` |
| `apache/commons-compress` | ok | 101 | 101 | 0 | 100.00% | 15.08s | `src/main/java/org/apache/commons/compress/harmony/unpack200` |
| `apache/commons-configuration` | ok | 63 | 63 | 0 | 100.00% | 5.89s | `src/main/java/org/apache/commons/configuration2/builder` |
| `apache/commons-csv` | ok | 55 | 55 | 0 | 100.00% | 17.09s | `src` |
| `apache/commons-dbcp` | ok | 68 | 68 | 0 | 100.00% | 11.91s | `src/main` |
| `apache/commons-io` | ok | 69 | 69 | 0 | 100.00% | 9.39s | `src/main/java/org/apache/commons/io/input` |
| `apache/commons-jexl` | ok | 56 | 56 | 0 | 100.00% | 26.46s | `src/main/java/org/apache/commons/jexl3/internal` |
| `apache/commons-lang` | ok | 62 | 62 | 0 | 100.00% | 1.06s | `src/main/java/org/apache/commons/lang3/function` |
| `apache/commons-pool` | ok | 54 | 54 | 0 | 100.00% | 17.56s | `src/main` |
| `apache/commons-text` | ok | 112 | 112 | 0 | 100.00% | 14.70s | `src/main` |
| `apache/commons-validator` | ok | 73 | 73 | 0 | 100.00% | 9.18s | `src/main` |
| `apache/dubbo` | ok | 65 | 65 | 0 | 100.00% | 30.46s | `dubbo-common/src/main/java/org/apache/dubbo/common/utils` |
| `apache/flink` | ok | 110 | 110 | 0 | 100.00% | 16.48s | `flink-runtime/src/main/java/org/apache/flink/runtime/jobmaster` |
| `apache/httpcomponents-client` | ok | 46 | 46 | 0 | 100.00% | 4.04s | `httpclient5/src/main/java/org/apache/hc/client5/http/entity` |
| `apache/httpcomponents-core` | ok | 112 | 112 | 0 | 100.00% | 13.42s | `httpcore5/src/main/java/org/apache/hc/core5/http/nio` |
| `apache/kafka` | ok | 71 | 71 | 0 | 100.00% | 9.85s | `clients/src/main/java/org/apache/kafka/common/security/oauthbearer` |
| `apache/lucene` | ok | 108 | 108 | 0 | 100.00% | 17.36s | `lucene/core/src/java/org/apache/lucene/document` |
| `apache/rocketmq` | ok | 62 | 62 | 0 | 100.00% | 2.80s | `remoting/src/main/java/org/apache/rocketmq/remoting/protocol/body` |
| `apache/shardingsphere` | ok | 1200 | 1200 | 0 | 100.00% | 25.75s | `test/it/parser/src/main` |
| `bazelbuild/bazel` | ok | 116 | 116 | 0 | 100.00% | 2.35s | `third_party/java/proguard/proguard6.2.2/src/proguard/classfile/attribute` |
| `checkstyle/checkstyle` | ok | 66 | 66 | 0 | 100.00% | 10.42s | `src/main/java/com/puppycrawl/tools/checkstyle/checks/coding` |
| `dropwizard/dropwizard` | ok | 68 | 68 | 0 | 100.00% | 6.51s | `dropwizard-jersey/src/main` |
| `elastic/elasticsearch` | ok | 111 | 111 | 0 | 100.00% | 14.32s | `server/src/main/java/org/elasticsearch/injection/guice` |
| `FasterXML/jackson-annotations` | ok | 63 | 63 | 0 | 100.00% | 2.84s | `src` |
| `FasterXML/jackson-core` | ok | 27 | 27 | 0 | 100.00% | 3.97s | `src/main/java/tools/jackson/core/util` |
| `FasterXML/jackson-databind` | ok | 111 | 111 | 0 | 100.00% | 35.39s | `src/main/java/tools/jackson/databind/deser` |
| `FasterXML/jackson-dataformats-text` | ok | 20 | 20 | 0 | 100.00% | 8.63s | `csv/src/main` |
| `google/error-prone` | ok | 113 | 113 | 0 | 100.00% | 26.70s | `core/src/main/java/com/google/errorprone/refaster` |
| `google/gson` | ok | 87 | 87 | 0 | 100.00% | 19.95s | `gson/src/main` |
| `google/guava` | ok | 84 | 84 | 0 | 100.00% | 18.57s | `guava/src/com/google/common/util` |
| `grpc/grpc-java` | ok | 80 | 80 | 0 | 100.00% | 10.72s | `xds/src/main/java/io/grpc/xds/internal` |
| `hibernate/hibernate-orm` | ok | 120 | 120 | 0 | 100.00% | 37.01s | `hibernate-core/src/main/java/org/hibernate/internal` |
| `junit-team/junit5` | ok | 86 | 86 | 0 | 100.00% | 11.52s | `jupiter-tests/src/test/java/org/junit/jupiter/api` |
| `micrometer-metrics/micrometer` | ok | 28 | 28 | 0 | 100.00% | 5.05s | `micrometer-core/src/main/java/io/micrometer/core/instrument/binder/jvm` |
| `mockito/mockito` | ok | 41 | 41 | 0 | 100.00% | 0.31s | `mockito-core/src/main/java/org/mockito/exceptions` |
| `netty/netty` | ok | 79 | 79 | 0 | 100.00% | 13.32s | `codec-http/src/main/java/io/netty/handler/codec/http/websocketx` |
| `openzipkin/zipkin` | ok | 59 | 59 | 0 | 100.00% | 9.64s | `zipkin-server/src/main` |
| `quarkusio/quarkus` | ok | 67 | 67 | 0 | 100.00% | 1.66s | `independent-projects/arc/tests/src/test/java/io/quarkus/arc/test/decorators` |
| `reactivestreams/reactive-streams-jvm` | ok | 30 | 30 | 0 | 100.00% | 28.90s | `tck/src` |
| `resilience4j/resilience4j` | ok | 72 | 72 | 0 | 100.00% | 7.19s | `resilience4j-spring6/src/main` |
| `spring-projects/spring-boot` | ok | 101 | 101 | 0 | 100.00% | 12.29s | `core/spring-boot/src/main/java/org/springframework/boot/context/properties` |
| `spring-projects/spring-data-jpa` | ok | 97 | 97 | 0 | 100.00% | 28.66s | `spring-data-jpa/src/main/java/org/springframework/data/jpa/repository/query` |
| `spring-projects/spring-framework` | ok | 44 | 44 | 0 | 100.00% | 5.97s | `spring-test/src/main/java/org/springframework/mock` |
| `spring-projects/spring-security` | ok | 46 | 46 | 0 | 100.00% | 3.23s | `web/src/main/java/org/springframework/security/web/server/authentication` |
| `square/retrofit` | ok | 57 | 57 | 0 | 100.00% | 11.77s | `retrofit/src` |

### Kotlin-based projects

| Project | Status | Files | Parsed | Failed | Parse Success | Duration | Target |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |

| `adrielcafe/voyager` | ok | 24 | 24 | 0 | 100.00% | 0.43s | `voyager-core/src/commonMain/kotlin` |
| `airbnb/mavericks` | ok | 43 | 43 | 0 | 100.00% | 5.52s | `mvrx/src` |
| `android/architecture-components-samples` | ok | 84 | 84 | 0 | 100.00% | 3.32s | `GithubBrowserSample/app/src` |
| `android/architecture-samples` | ok | 47 | 47 | 0 | 100.00% | 3.37s | `app/src` |
| `android/compose-samples` | ok | 59 | 59 | 0 | 100.00% | 4.52s | `JetNews/app/src` |
| `android/nowinandroid` | ok | 34 | 34 | 0 | 100.00% | 1.50s | `core/data/src` |
| `android/sunflower` | ok | 61 | 61 | 0 | 100.00% | 3.06s | `app/src` |
| `android/uamp` | ok | 15 | 15 | 0 | 100.00% | 2.83s | `common/src` |
| `apollographql/apollo-kotlin` | ok | 59 | 59 | 0 | 100.00% | 7.27s | `libraries/apollo-api/src/commonMain/kotlin` |
| `arkivanov/Decompose` | ok | 92 | 92 | 0 | 100.00% | 2.28s | `decompose/src/commonMain/kotlin` |
| `arkivanov/Essenty` | ok | 12 | 12 | 0 | 100.00% | 0.58s | `state-keeper/src/commonMain/kotlin` |
| `arrow-kt/arrow` | ok | 38 | 38 | 0 | 100.00% | 4.98s | `arrow-libs/core/arrow-core/src/commonMain/kotlin` |
| `badoo/Reaktive` | ok | 91 | 91 | 0 | 100.00% | 2.06s | `reaktive/src/commonMain/kotlin/com/badoo/reaktive/observable` |
| `cashapp/molecule` | ok | 6 | 6 | 0 | 100.00% | 0.61s | `molecule-runtime/src/commonMain/kotlin` |
| `cashapp/redwood` | ok | 51 | 51 | 0 | 100.00% | 16.07s | `redwood-yoga/src/commonMain/kotlin` |
| `cashapp/sqldelight` | ok | 14 | 14 | 0 | 100.00% | 0.91s | `runtime/src/commonMain/kotlin` |
| `cashapp/turbine` | ok | 7 | 7 | 0 | 100.00% | 0.80s | `src/commonMain/kotlin` |
| `cashapp/zipline` | ok | 36 | 36 | 0 | 100.00% | 2.43s | `zipline/src/commonMain/kotlin` |
| `chrisbanes/tivi` | ok | 54 | 54 | 0 | 100.00% | 1.63s | `domain/src/commonMain/kotlin` |
| `coil-kt/coil` | ok | 69 | 69 | 0 | 100.00% | 4.63s | `coil-core/src/commonMain/kotlin` |
| `detekt/detekt` | ok | 98 | 98 | 0 | 100.00% | 11.43s | `detekt-rules-style/src/main` |
| `DroidKaigi/conference-app-2024` | ok | 55 | 55 | 0 | 100.00% | 2.07s | `core/data/src/commonMain/kotlin` |
| `google/accompanist` | ok | 14 | 14 | 0 | 100.00% | 0.84s | `sample/src` |
| `InsertKoinIO/koin` | ok | 74 | 74 | 0 | 100.00% | 2.59s | `projects/core/koin-core/src/commonMain/kotlin` |
| `JakeWharton/timber` | ok | 1 | 1 | 0 | 100.00% | 1.05s | `timber/src/androidMain/kotlin` |
| `JetBrains/Exposed` | ok | 120 | 120 | 0 | 100.00% | 76.33s | `exposed-tests/src` |
| `kizitonwose/Calendar` | ok | 41 | 41 | 0 | 100.00% | 2.37s | `compose-multiplatform/library/src/commonMain/kotlin` |
| `Kotest/Kotest` | ok | 52 | 52 | 0 | 100.00% | 1.65s | `kotest-framework/kotest-framework-engine/src/commonMain/kotlin/io/kotest/core/spec` |
| `Kotlin/kotlin-wrappers` | ok | 78 | 78 | 0 | 100.00% | 3.20s | `kotlin-node/karakum/src/jsMain/kotlin/wrappersgenerator/node/plugins` |
| `Kotlin/kotlinx-atomicfu` | ok | 8 | 8 | 0 | 100.00% | 0.47s | `atomicfu/src/nativeMain/kotlin` |
| `Kotlin/kotlinx-datetime` | ok | 55 | 55 | 0 | 100.00% | 7.83s | `core/common/src` |
| `Kotlin/kotlinx.cli` | ok | 7 | 7 | 0 | 100.00% | 1.87s | `core/commonMain/src` |
| `Kotlin/kotlinx.coroutines` | ok | 111 | 111 | 0 | 100.00% | 14.83s | `kotlinx-coroutines-core/common/src` |
| `Kotlin/kotlinx.serialization` | ok | 50 | 50 | 0 | 100.00% | 2.95s | `core/commonMain/src` |
| `ktorio/ktor` | ok | 92 | 92 | 0 | 100.00% | 5.16s | `ktor-server/ktor-server-core/common/src` |
| `mockk/mockk` | ok | 67 | 67 | 0 | 100.00% | 3.95s | `modules/mockk/src/commonMain/kotlin` |
| `patrykandpatrick/vico` | ok | 80 | 80 | 0 | 100.00% | 16.23s | `vico/compose/src/commonMain/kotlin/com/patrykandpatrick/vico/compose/cartesian` |
| `raamcosta/compose-destinations` | ok | 82 | 82 | 0 | 100.00% | 2.24s | `compose-destinations/src` |
| `RBusarow/Dispatch` | ok | 21 | 21 | 0 | 100.00% | 0.95s | `dispatch-core/src` |
| `realm/realm-kotlin` | ok | 79 | 79 | 0 | 100.00% | 19.64s | `packages/library-base/src/commonMain/kotlin/io/realm/kotlin/internal` |
| `skydoves/Balloon` | ok | 59 | 59 | 0 | 100.00% | 23.28s | `balloon/src` |
| `skydoves/landscapist` | ok | 33 | 33 | 0 | 100.00% | 3.10s | `landscapist-core/src/commonMain/kotlin` |
| `skydoves/Pokedex` | ok | 16 | 16 | 0 | 100.00% | 0.64s | `app/src` |
| `slackhq/circuit` | ok | 28 | 28 | 0 | 100.00% | 3.85s | `circuit-foundation/src/commonMain/kotlin` |
| `slackhq/compose-lints` | ok | 63 | 63 | 0 | 100.00% | 5.59s | `compose-lint-checks/src` |
| `slackhq/paparazzi` | ok | 88 | 88 | 0 | 100.00% | 15.23s | `paparazzi/src/main` |
| `square/moshi` | ok | 64 | 64 | 0 | 100.00% | 23.22s | `moshi/src` |
| `square/okio` | ok | 41 | 41 | 0 | 100.00% | 3.77s | `okio/src/jvmMain/kotlin` |
| `touchlab/Kermit` | ok | 10 | 10 | 0 | 100.00% | 0.25s | `kermit-core/src/commonMain/kotlin` |
| `touchlab/SKIE` | ok | 18 | 18 | 0 | 100.00% | 0.30s | `SKIE/runtime/kotlin/src/commonMain/kotlin` |

## Next Step

- Drive strict parse failures and round-trip findings to zero across the full selected target set.
- Replace lossless rewrite with full source serialization only after the parser metadata is rich enough to preserve all syntax and trivia.
- After round-trip correctness is stable, capture CPU and heap profiles on the slowest projects above.
- Optimize parser hot paths and allocation-heavy graph construction paths before widening the benchmark set.
