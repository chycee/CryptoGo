# Crypto Go 기술 명세서 (Technical Specifications)

**작성일**: 2026년 1월 7일
**버전**: 2.1 (Consolidated)

## 1. 시스템 아키텍처 (System Architecture)

Crypto Go는 **고성능 백엔드 엔진(Backend Engine)**과 **사용자 중심의 UI 레이어**로 구성됩니다.
상세 레이어 구조는 [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)를 참조하십시오.

### 1.1 Core Components
*   **PriceService**: 실시간 시세 데이터 중앙 관리 (In-Memory).
*   **Exchange Workers**: Upbit, Bitget(Spot/Futures) 웹소켓 연결 및 데이터 수집.
*   **Storage Layer**: SQLite 및 로컬 파일 시스템을 통한 영구 저장소.
*   **Bootstrap**: 앱 부팅 시퀀스 및 자산 동기화 오케스트레이터.

---

## 2. 도메인 및 데이터 모델 (Domain & Data)

### 2.1 핵심 엔티티 (Entities)
| 엔티티 | 설명 |
| :--- | :--- |
| `Ticker` | 거래소별 가공된 시세 데이터 객체 |
| `MarketData` | 단일 종목의 통합 정보 (김프 포함) |
| `CoinInfo` | 코인 메타데이터 (심볼, 아이콘, 즐겨찾기) |
| `AppConfig` | 사용자 설정 (Key-Value) |

### 2.2 데이터 영속성 (Persistence)
*   **DB**: `%LocalAppData%\CryptoGo\data\cryptogo.db`
*   **Assets**: `%LocalAppData%\CryptoGo\assets\icons\`

---

## 3. 부팅 및 동기화 (Startup Sequence)

### Phase 1: 초기화 (Bootstrapping)
1.  **Config Load**: `config.yaml` 로드 및 유효성 검사.
2.  **DB Check**: SQLite 파일 존재 여부 확인 및 Auto Migration.
3.  **Dir Check**: `assets/icons` 등 필수 디렉토리 생성.

### Phase 2: 자산 동기화 (Asset Sync)
1.  거래소 API로 최신 심볼 목록 Fetch.
2.  DB와 대조하여 `New`(추가), `Delisted`(비활성), `Re-listed`(재활성) 처리.
3.  아이콘 없거나 오래된 경우 병렬 다운로드.

---

## 4. UI 연동 인터페이스 (API)
*   `GetMarketData() -> []MarketData`: 모든 코인의 현재가, 김프 등 리턴.
*   `GetCoinMeta(symbol) -> CoinInfo`: 특정 코인의 아이콘 경로, 즐겨찾기 상태 리턴.
*   `ToggleFavorite(symbol)`: 즐겨찾기 상태 토글 및 DB 즉시 반영.

---

## 5. 테스트 및 모니터링 (Testing & Observability)

### 테스트 전략
*   **단위 테스트**: 에지 케이스(환율 0, 음수 가격), 정밀도(`Decimal`) 검증.
*   **통합 테스트**: WebSocket Mocking, `go test -race`로 동시성 안전 보장.

### 로깅 정책
*   **Structured Log**: `slog`를 사용하며 JSON 포맷으로 기록.
*   **Log Rotation**: `lumberjack`으로 10MB/3백업 제한, 28일 자동 삭제.

---

*Last Updated: 2026-01-07*
