# 성능 분석 및 최적화 계획

## 현재 성능 벤치마크 (34.156s)

### Java 파싱 성능
| Grammar | Time/op | Bytes/op | Allocs/op | 비고 |
|---------|---------|----------|-----------|------|
| Java8   | 30.3ms  | 35.4MB   | 443K      | 기준 |
| Java9   | 41.2ms  | 48.7MB   | 624K      | 36% 느림 |
| Java20  | 30.1ms  | 34.3MB   | 445K      | 최고 |

**결론**: Java20이 가장 빠르고 효율적

### Kotlin 파싱 성능
| Test | Time/op | Bytes/op | Allocs/op |
|------|---------|----------|-----------|
| Simple | 13.0ms  | 14.1MB   | 178K      |
| With Normalization | 8.1ms  | 8.7MB    | 109K      |

**놀라운 점**: Normalization이 실제로 성능을 개선함!

## 최적화 기회 분석

### 1. 메모리 할당 감소 (HIGH IMPACT)
- 현재: 35-49 MB/op, 443-624K allocs/op
- 목표: 50% 감소
- 방법:
  - strings.Builder 풀링
  - sync.Pool 활용
  - 불필요한 문자열 복사 제거

### 2. Java9 grammar 제거 (MEDIUM IMPACT)
- Java9가 36% 느리고 48% 더 많은 메모리 사용
- 현재 사용되지 않으면 제거 고려

### 3. 병렬 처리 개선 (MEDIUM IMPACT)
- 현재 고정된 worker 수
- 동적 worker 풀 크기 조정

### 4. Lexer/Parser 캐싱 (LOW IMPACT)
- 반복되는 토큰 스트림 생성
- Grammar 직렬화 캐시
