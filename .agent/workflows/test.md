---
description: Go 테스트 실행
---
# /test 워크플로우

이 워크플로우는 Crypto Go의 전체 테스트를 실행합니다.

## 단계

// turbo-all
1. 전체 테스트 실행 (상세 출력)
```bash
go test -v ./...
```

2. 동시성 안전 검사 (Race Condition)
```bash
go test -race ./...
```

3. Fuzz 테스트 실행 (SafeMath/Parser 경계값 탐색)
```bash
go test -fuzz=FuzzSafeAdd -fuzztime=10s ./pkg/safe
go test -fuzz=FuzzSafeMul -fuzztime=10s ./pkg/safe
```

4. 통합 테스트 실행 (비트겟 테스트넷 실거래 로직 검증)
```bash
# _workspace/secrets/demo.yaml 설정 필요
go run ./cmd/integration/main.go
```

## 테스트 범위
| 패키지 | 테스트 내용 |
|--------|-------------|
| `pkg/safe` | SafeMath Panic 테스트, Fuzz 테스트 |
| `pkg/quant` | 타입 변환, Fuzz 테스트 |
| `domain` | 엔티티 로직, Gap 계산 |
| `engine` | Sequencer 상태 관리, Gap Detection |
| `infra` | WebSocket 파싱, 환율 조회, Panic Recovery |
| `integration` | Bitget API 연동, 주문/취소 실무 검증 |

## 주의사항
- Pure Go SQLite 드라이버 사용으로 CGO 불필요
- 테스트 시 `_workspace/data`에 임시 DB 생성 가능
- Fuzz 테스트는 CI에서 시간 제한(10s)으로 실행
- `-race` 테스트는 CGO 필요 (Windows: MSVC/GCC 필요, Linux/CI 권장)
- **보안**: 통합 테스트 시 사용하는 키 파일은 반드시 `_workspace/secrets/`에 위치해야 하며, Git에 커밋되지 않도록 주의해야 합니다. (Pre-commit Hook 작동 중)
