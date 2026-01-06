---
description: Go 애플리케이션 빌드
---
# /build 워크플로우

이 워크플로우는 Crypto Go 애플리케이션을 빌드합니다.

## 단계

// turbo-all
1. 의존성 동기화
```bash
go mod tidy
```

2. 애플리케이션 빌드
```bash
go build -o crypto-go.exe ./cmd/app/main.go
```

## 결과
- 빌드 성공 시 프로젝트 루트에 `crypto-go.exe` 생성
- Pure Go SQLite 드라이버 사용으로 CGO 불필요
