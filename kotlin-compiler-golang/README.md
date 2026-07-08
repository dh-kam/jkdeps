# kotlin-compiler-golang

`kotlin-compiler-golang`은 `kotlin-compiler-embeddable`을 순수 Go로 포팅하기 위한 작업 공간입니다.

현재 상태는 1단계 부트스트랩입니다.

- Kotlin 파일 파싱 API (`ParseFile`, `ParseRepository`)
- Kotlin 헤더(package/import) 추출
- Top-level 선언(AST-lite) 추출 및 심볼 테이블 생성
- 패키지 의존 그래프(deps) 생성
- CLI (`cmd/kotlin-compiler-golang`)
- embeddable/runtime 아카이브(JAR/KLIB) 인벤토리 도구 (`cmd/ktcg-inventory`)

## 빠른 실행

```bash
# Kotlin 파싱
 go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --json
 go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --file-timeout 2s --json
 # build.gradle.kts / settings.gradle.kts 포함
 go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --include-build-scripts --json
 # 공식 파서(embeddable) 백엔드로 파싱
 go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --parser-backend embeddable --json

# 심볼 테이블 출력
 go run ./cmd/kotlin-compiler-golang symbols --repo /path/to/repo --json

# 의존 그래프 출력
 go run ./cmd/kotlin-compiler-golang deps --repo /path/to/repo --json

# import 해석 리포트 출력
 go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --json

# 구문 오류를 허용하고 분석 진행(lenient)
 go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --lenient --json

# embeddable 인벤토리 기반 외부 import 해석
 go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/embeddable-index.json --json

# 다중 인벤토리(embeddable + stdlib-js) 병합 해석
 go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/embeddable-index.json --inventory ./kotlin-compiler-golang/inventory/stdlib-js-index.json --json

# runtime 인벤토리(embeddable + stdlib-js + kotlinx-browser) 해석
 go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --json

# acceptance 리포트 생성 + 품질 게이트
 go run ./cmd/kotlin-compiler-golang acceptance --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --max-failed-files 0 --max-unresolved-imports 0 --max-files-with-diagnostics 10 --max-total-diagnostics 70 --out ./kcg-acceptance.json
 # 공식 파서 백엔드 임베디드 모드에서 acceptance 실행
 go run ./cmd/kotlin-compiler-golang acceptance --repo /path/to/repo --parser-backend embeddable --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --max-failed-files 0 --max-unresolved-imports 0 --max-files-with-diagnostics 10 --max-total-diagnostics 70 --out ./kcg-acceptance.json

# 웹 그래프 아티팩트(HTML+JSON) 생성
 go run ./cmd/kotlin-compiler-golang graph --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --out ./jkdeps-graph

# embeddable 인벤토리 생성
 ./scripts/build_kcg_inventory.sh

# stdlib-js 인벤토리 생성
 ./scripts/build_kcg_stdlib_js_inventory.sh

# browser 인벤토리 생성
 ./scripts/build_kcg_browser_inventory.sh

# atomicfu 인벤토리 생성
 ./scripts/build_kcg_atomicfu_inventory.sh

# runtime 병합 인벤토리 생성
 ./scripts/build_kcg_runtime_inventory.sh

# Kotlin 실제 컴파일(하이브리드, JVM kotlinc 위임)
#
# 포팅 단계는 Go 파서/의미 분석 중심이며,
# 플러그인·annotation processor 실행은 kotlinc 위임으로 처리합니다.
 go run ./cmd/kotlin-compiler-golang compile --repo /path/to/repo --out ./out --include-runtime
 go run ./cmd/kotlin-compiler-golang compile \
   --repo /path/to/repo \
   --out ./out \
   --kotlinc /opt/kotlinc/bin/kotlinc \
   --plugin /path/to/compose-compiler-plugin.jar \
   --plugin /path/to/ksp-cli-plugin.jar \
   --arg-file /path/to/kcg-kotlinc.args \
   --dry-run
  # (relative paths in --plugin, --classpath, --arg-file are resolved against --repo)

# passthrough extra kotlinc args:
go run ./cmd/kotlin-compiler-golang compile \
   --repo /path/to/repo \
   --out ./out \
   -- \
   -P plugin:com.google.devtools.ksp.symbol-processing:enabled=true \
   -Xjsr305=strict

# or compile explicit sources/directories only
# (relative paths in --source / --exclude are resolved against --repo)
go run ./cmd/kotlin-compiler-golang compile \
  --repo /path/to/repo \
  --source src/main/kotlin \
  --source src/test/kotlin \
  --out ./out

# list selected sources without invoking kotlinc
go run ./cmd/kotlin-compiler-golang compile \
  --repo /path/to/repo \
  --source src/main/kotlin \
  --list-sources

# or exclude generated paths during source discovery
go run ./cmd/kotlin-compiler-golang compile \
  --repo /path/to/repo \
  --exclude /path/to/repo/build \
  --exclude /path/to/repo/.gradle \
  --out ./out
```

## 포팅 전략

- 단계 1: 파서/토큰/AST 골격을 Go로 구축
- 단계 2: 프론트엔드 의미 분석(타입/스코프/디스크립터) 포팅
- 단계 3: 진단 및 해석 단계 호환성 확장
- 단계 4: Kotlin 컴파일러 테스트셋 기반 동작 동등성 검증

`완전 동일 동작`은 대규모 작업이므로 단계적으로 커버리지를 늘려야 합니다.

## 기본 외부 패키지 해석

외부 인벤토리가 없어도 아래 루트는 기본적으로 외부 패키지로 해석합니다.

- `java.*`
- `javax.*`
- `jdk.*`
- `sun.*`
- `android.*`
- `org.codehaus.mojo.animal_sniffer.*`
