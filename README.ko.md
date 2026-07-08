# jkdeps

[English README](README.md)

`jkdeps`는 Go에서 Java/Kotlin 저장소를 분석하기 위한 라이브러리와 CLI 도구입니다. 소스 파일을 파싱하고, package/import/type reference를 추출하고, 의존성 그래프와 파일 단위 의존성 리포트를 만듭니다. 다른 Go 모듈에서는 루트 패키지를 import해서 사용할 수 있습니다.

```go
import "github.com/dh-kam/jkdeps"
```

이 저장소에는 smoke parsing, 의존성 그래프 생성, Kotlin import 해석, inventory 생성, Kotlin compiler 포팅 실험을 위한 CLI도 함께 포함되어 있습니다.

## 제공 기능

- mixed Java/Kotlin 저장소 분석용 public root package: `github.com/dh-kam/jkdeps`
- Java grammar 선택: `java`, `java7`, `java8`, `java9`, `java11`, `java17`, `java20`, `java21`, `java25`
- 빠른 package/import 분석을 위한 Java `header-only` 모드
- Java signature와 일부 body reference까지 수집하는 `full` 모드
- `.kt` 파일 및 선택적 `.kts`/Gradle script 파싱
- package 단위 또는 directory 단위 의존성 그래프
- unresolved import/reference를 포함한 파일 단위 의존성 리포트
- stdlib/runtime/package 해석을 위한 external inventory 로딩
- `cmd/jkdeps`, `cmd/kotlin-compiler-golang`, `cmd/ktcg-inventory` CLI

## 상태

`jkdeps`는 아직 pre-v1입니다. 안정화 대상 API는 루트 패키지 `github.com/dh-kam/jkdeps`입니다. `internal/...` 패키지와 `cmd/...` 아래의 main package는 외부 라이브러리 API가 아닙니다.

`github.com/dh-kam/jkdeps/kotlin-compiler-golang` 패키지는 Kotlin 전용 실험과 포팅 작업을 위해 남아 있지만, 새 소비자는 특별한 이유가 없다면 루트 `jkdeps` 패키지를 사용하는 것이 좋습니다.

현재 저장소는 `third_party/antlr4-go` 아래의 로컬 ANTLR Go runtime fork를 `go.mod`의 `replace`로 사용합니다. 해당 runtime 수정이 upstream되거나 별도 모듈로 공개되기 전까지 CLI 빌드는 clone 기반 사용이 가장 안전합니다.

## 빠른 시작

```bash
git clone git@github.com:dh-kam/jkdeps.git
cd jkdeps
go test ./...
```

mixed Java/Kotlin 의존성 분석:

```bash
go run ./cmd/jkdeps deps \
  --repo /path/to/repository \
  --java-grammar java20 \
  --workers 8 \
  --group-by package \
  --json
```

브라우저에서 볼 수 있는 그래프 생성:

```bash
go run ./cmd/jkdeps graph \
  --repo /path/to/repository \
  --java-grammar java20 \
  --file-timeout 2s \
  --group-by package \
  --inventory ./kotlin-compiler-golang/inventory/runtime-index.json \
  --lenient \
  --out ./jkdeps-mixed-graph
```

parser smoke test:

```bash
go run ./cmd/jkdeps smoke-parse \
  --repo /path/to/repository \
  --java-grammar java20 \
  --workers 8 \
  --max-errors 20 \
  --file-timeout 2s
```

## Go API

다른 Go 모듈에서 루트 패키지를 사용합니다.

```bash
go get github.com/dh-kam/jkdeps
```

저장소 전체 분석 예시:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dh-kam/jkdeps"
)

func main() {
	report, err := jkdeps.Analyze(context.Background(), "/path/to/repository", jkdeps.Options{
		Parse: jkdeps.ParseOptions{
			Workers:       8,
			JavaGrammar:   jkdeps.JavaGrammar20,
			JavaParseMode: jkdeps.JavaParseModeHeaderOnly,
			KotlinScripts: jkdeps.KotlinScriptsRegular,
			ParseTimeout:  2 * time.Second,
		},
		Graph: jkdeps.GraphOptions{
			GroupBy: jkdeps.GroupByPackage,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("parsed %d/%d files\n", report.Repository.ParsedFiles, report.Repository.TotalFiles)
	fmt.Printf("graph: %d nodes, %d edges\n", len(report.Graph.Nodes), len(report.Graph.Edges))
	fmt.Printf("unresolved imports: %d\n", len(report.Dependencies.UnresolvedImports))
}
```

필요한 단계만 나누어 사용할 수도 있습니다.

```go
repo, err := jkdeps.ParseRepository(ctx, "/path/to/repository", jkdeps.ParseOptions{
	Workers:       8,
	JavaParseMode: jkdeps.JavaParseModeFull,
	KotlinScripts: jkdeps.KotlinScriptsAll,
})
if err != nil {
	return err
}

external, err := jkdeps.LoadExternalIndex("./kotlin-compiler-golang/inventory/runtime-index.json")
if err != nil {
	return err
}

graph := repo.BuildGraph(jkdeps.GraphOptions{
	GroupBy:  jkdeps.GroupByDir,
	External: external,
	Filter: jkdeps.GraphFilter{
		MinEdgeCount:  2,
		IncludePrefix: []string{"com.example."},
	},
})
deps := repo.BuildDependencyReport(external)
_ = graph
_ = deps
```

### Zero-Value 기본값

`jkdeps.Options{}`는 그대로 사용할 수 있습니다.

- Java grammar: `java20`
- Java parse mode: `header-only`
- Kotlin scripts: 일반 `.kts` 포함, Gradle script 제외
- Graph grouping: `package`
- Workers: 내부 기본값
- Max errors per file: 내부 기본값
- Parse timeout: 비활성화
- Syntax mode: strict

명시적인 기본값이 필요하면 다음 함수를 사용할 수 있습니다.

```go
opts := jkdeps.DefaultOptions()
opts.Parse.Workers = 8
opts.Graph.Filter = jkdeps.GraphFilter{MinEdgeCount: 2}
```

Java의 `extends`, `implements`, field type, method return/parameter, constructor parameter, `throws`, 그리고 constructor call, class literal, cast, `instanceof`, local variable type, catch type, method reference 같은 일부 body reference가 필요하면 `JavaParseModeFull`을 사용합니다.

`JavaParseModeHeaderOnly`에서 Java 파일의 `Parsed`는 package/import header 추출에 성공했다는 뜻이며, 전체 Java 소스의 syntax validation을 통과했다는 뜻은 아닙니다. mixed dependency 분석에서 Kotlin은 현재 package/import 정보를 제공하고, Java `full` 모드는 typed reference도 함께 제공합니다.

### Public Data Model

루트 패키지는 단순한 DTO 스타일 구조체를 반환합니다.

- `Repository`: 전체 parse count, duration, 정렬된 파일 목록
- `File`: path, relative path, language, package, imports, references, parse status, diagnostics, duration
- `Graph`: package 또는 directory 기준 node/weighted edge
- `DependencyReport`: 파일 단위 dependency row와 unresolved import/reference
- `ExternalIndex`: inventory JSON에서 로드하거나 코드에서 만든 package/symbol 힌트

`error`는 invalid option, 접근 불가능한 root, walk/read 실패, context cancellation 같은 운영 실패에만 사용합니다. 소스 파싱 실패는 `File.Diagnostics`와 `Repository.FailedFiles`에 기록됩니다.

## CLI 명령

### `cmd/jkdeps`

Java/Kotlin mixed repository 도구입니다.

```bash
go run ./cmd/jkdeps smoke-parse --repo /path/to/repository
go run ./cmd/jkdeps deps --repo /path/to/repository --json
go run ./cmd/jkdeps graph --repo /path/to/repository --out ./jkdeps-mixed-graph
go run ./cmd/jkdeps roundtrip-check --repo /path/to/repository --rewrite-mode lossless
```

`deps`는 빠른 import 기반 분석을 위해 Java를 기본적으로 `header-only`로 처리합니다. typed Java reference까지 필요하면 `--java-parse-mode full`을 사용합니다.

### `cmd/kotlin-compiler-golang`

Kotlin 중심 parser/resolver/compiler-porting 작업 공간입니다.

```bash
go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang symbols --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang deps --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --json
go run ./cmd/kotlin-compiler-golang graph --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --out ./jkdeps-graph
go run ./cmd/kotlin-compiler-golang acceptance --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --max-failed-files 0 --max-unresolved-imports 0
```

### `cmd/ktcg-inventory`

Kotlin runtime/stdlib 해석 흐름에서 사용하는 inventory builder입니다. 저장소에는 다음 wrapper script도 포함되어 있습니다.

```bash
./scripts/build_kcg_inventory.sh
./scripts/build_kcg_stdlib_js_inventory.sh
./scripts/build_kcg_browser_inventory.sh
./scripts/build_kcg_atomicfu_inventory.sh
./scripts/build_kcg_runtime_inventory.sh
```

## Parser 생성

ANTLR grammar는 `grammars/` 아래에 있고, 생성된 Go parser는 `internal/parsers/` 아래에 커밋됩니다.

parser 재생성:

```bash
./scripts/generate_parsers.sh
```

이 스크립트는 필요하면 `tools/antlr-4.13.2-complete.jar`를 다운로드합니다. 다운로드한 jar, 생성된 `.tokens`/`.interp` side file, 로컬 빌드 산출물은 Git에서 제외합니다.

## 저장소 구조

```text
.
├── api.go                         # public root Go API
├── cmd/jkdeps                     # mixed Java/Kotlin CLI
├── cmd/kotlin-compiler-golang     # Kotlin-focused CLI
├── cmd/ktcg-inventory             # inventory CLI
├── docs                           # 설계, 검증, 포팅 문서
├── grammars                       # Java/Kotlin ANTLR grammars
├── internal/ast                   # internal AST contracts
├── internal/mixedgraph            # internal parser/graph implementation
├── internal/parser                # internal ANTLR parser adapter
├── internal/parsers               # generated ANTLR Go parsers
├── kotlin-compiler-golang         # experimental Kotlin package
├── scripts                        # generation, inventory, validation helpers
├── testdata                       # representative Java/Kotlin samples
└── third_party/antlr4-go          # local ANTLR Go runtime fork
```

## 검증

자주 쓰는 검증 명령:

```bash
go test ./...
make kcg-acceptance
make jkdeps-graph
make oss-matrix
```

대형 sample run은 기본적으로 `/tmp/jkdeps-samples`를 사용합니다. pinned sample을 가져오거나 갱신하려면 다음을 실행합니다.

```bash
./scripts/fetch_samples.sh
```

## License

MIT. 자세한 내용은 [LICENSE](LICENSE)를 참고하세요.
