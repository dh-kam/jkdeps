# 파싱 실패 사례 분석

## Kotlin 파싱 실패 분석 (kotlinx.coroutines common/src)

### 통계
- 전체: 111개 파일
- 성공: 103개 (92.79%)
- 실패: 8개 (7.21%)

### 실패 사례 상세

#### 1. EventLoop.common.kt
**에러**: `mismatched input '{'` / `extraneous input 'when'` / `extraneous input 'null'`

**원인**: Kotlin 1.6+의 `when` 식에서 guard 절(`when` 조건) 지원

**해결 방법**: 이미 구현된 Kotlin normalization으로 해결 가능

#### 2. JobSupport.kt
**에러**: `mismatched input '"'`

**원인**: 삼중 따옴표나 문자열 이스케이프 문제

**해결 방법**: 문자열 리터럴 처리 개선

#### 3. BufferedChannel.kt
**에러**: `no viable alternative at input '<'`

**원인**: 람다/제네릭 타입 파라미터 `<E>` 표기법

**해결 방법**: ANTLR Kotlin grammar 업데이트 필요

#### 4. SafeCollector.common.kt
**에러**: `no viable alternative at input '('` / `extraneous input 'fold@'`

**원인**: `@fold` 등의 함수 타입 어노테이션

**해결 방법**: Kotlin 어노테이션 스킵 처리 개선

#### 5. Select.kt
**에러**: `no viable alternative at input 'SelectInstanceInternal'`

**원인**: 내부 클래스 참조 표현법

**해결 방법**: ANTLR grammar 한계, 전용 파서 필요 시 사용

### 결론
- ANTLR Kotlin grammar는 현대 Kotlin 문법의 90% 이상 지원
- 실패 케이는 대부분 복잡한 타입 시스템이나 최신 문법
- 실무적으로는 lenient 모드로 충분히 활용 가능
