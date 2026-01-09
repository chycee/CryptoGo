# ADR 0006: HFT Readiness & Hyper-Optimization

## Context (배경)

CryptoGo의 목표인 "p99 < 1ms" 레이턴시와 100% 결정론적 정합성을 달성하기 위해, 기존 구조에서 발견된 미세한 성능 병목점(인메모리 할당, 부동소수점 연산, 캐시 효율)을 해결해야 했습니다.

1. **힙 할당(Heap Allocation)**: 핫패스 내의 빈번한 객체 생성은 GC(Garbage Collection) 지터를 유발하여 레이턴시 스파이크의 원인이 됨.
2. **부동소수점(Float64)**: 외부 API에서 전송되는 `float64`는 금융 데이터 정합성에 위험요소이며, 연산 오버헤드 존재.
3. **캐시 비효율**: 구조체 필드 배치에 따른 메모리 패딩(Padding)으로 인해 CPU 캐시 라인 활용도가 저하되는 문제.

## Decision (결정)

다음의 4가지 핵심 최적화를 도입하여 HFT(High-Frequency Trading) 수준의 실행 환경을 구축합니다.

### 1. Zero-Alloc Hotpath Strategy
- **Interface Refactor**: `OnMarketUpdate` 인터페이스가 슬라이스를 반환하는 대신, 호출자가 제공한 고정 버퍼(`out []domain.Order`)에 결과를 쓰도록 변경.
- **Result**: 시그널 발생 시 추가적인 힙 할당을 0으로 억제.

### 2. Floating-Point Free Ingestion
- **json.Number Usage**: 모든 인프라 게이트웨이(Upbit, Bitget, FX)에서 `float64` 파싱을 폐기하고 `json.Number`를 사용하여 문자열로 수신.
- **Strict Integer Parsing**: 수신 즉시 `strconv.ParseFloat` 조차 호출하지 않는 고유의 **Fixed-Point String Parser**를 통해 `int64` (PriceMicros, QtySats)로 직접 변환. 연산의 모든 단계에서 부동소수점 오버헤드와 오차 가능성을 0%로 배제.

### 3. Cache-Line Alignment
- **Field Reordering**: `MarketState`, `CoinInfo` 등 핫패스 구조체의 필드를 8바이트 단위로 내림차순 정렬하여 메모리 패딩 제거.
- **Performance**: 구조체 크기를 최소화하고 CPU 캐시 적중률 향상.

### 4. Event Pooling (Strict Management)
- **sync.Pool**: 모든 시장 데이터 이벤트(`MarketUpdateEvent`)를 `sync.Pool`에서 관리.
- **Warump**: 애플리케이션 시작 시 `event.Warmup()`을 통해 객체 풀을 미리 채워 런타임 할당 최소화.

## Consequences (결과 및 영향)

- **Performance**: 벤치마크 결과 핫패스 연산 속도가 **~16ns/op**로 단축되었으며, 할당량이 **0 B/op**로 고정됨.
- **Stability**: GC 지터가 사라져 p99 레이턴시의 안정성이 비약적으로 향상됨.
- **Auditability**: 모든 데이터가 정수형으로 관리되어 재설명(Replay) 시 데이터 불일치 가능성 차단.
- **Complexity**: 인터페이스 변경으로 인해 테스트 코드 작성이 다소 복잡해졌으나(버퍼 관리 필요), 성능 이득이 이를 압도함.
