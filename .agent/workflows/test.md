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

## 테스트 범위
| 패키지 | 테스트 내용 |
|--------|-------------|
| `domain` | 엔티티 로직, Gap 계산 |
| `service` | PriceService 상태 관리 |
| `infra` | WebSocket 파싱, 환율 조회 |
| `infra/storage` | SQLite CRUD (Create, Read, Update, Delete) |

## 주의사항
- Pure Go SQLite 드라이버 사용으로 CGO 불필요
- 테스트 시 임시 DB(`test.db`) 생성 후 자동 삭제
