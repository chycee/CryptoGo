---
description: Go 애플리케이션 실행
---
# /run 워크플로우

이 워크플로우는 Crypto Go 애플리케이션을 즉시 실행합니다.

## 단계

// turbo-all
1. 애플리케이션 실행
```bash
go run ./cmd/app/main.go
```

## 사전 요구사항
- `_workspace/secrets/` 하위에 API 키 설정 파일(`demo.yaml` 등)이 위치해야 합니다.
- 실행 시 `_workspace/data` 및 `_workspace/logs`가 자동 생성됩니다.

## 동작 설명
실행 시 다음 작업이 자동으로 수행됩니다:
1. **Bootstrap**: 설정 로드 → `_workspace/data/events.db` 연결 → `_workspace/logs/app.log` 초기화
2. **Asset Sync**: 심볼 목록 동기화 및 아이콘 다운로드 (`_workspace/data` 저장)
3. **Workers**: Upbit/Bitget WebSocket 연결

## 종료
- `Ctrl+C`로 Graceful Shutdown (모든 연결 정리 후 종료)
