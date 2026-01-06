# CryptoGo: Indie Quant Architecture Design

**Role**: Indie Quant Developer (Deterministic, Agile, Pragmatic).  
**MINDSET**: "Backtest is Reality." / "Complexity is the Enemy."

---

## 1. DATA Principles

| Rule | Implementation |
|------|---------------|
| Money: `int64` only | `PriceMicros` (×10⁶), `QtySats` (×10⁸) |
| Time: `int64` only | `TimeStamp` (Unix Microseconds) |
| No `float32`/`float64` | Internal logic is float-free |
| SafeMath | `pkg/safe`: Add/Sub/Mul/Div with Panic-on-Overflow |
| Unit Suffix | All int64 fields have explicit suffix (e.g., `PriceMicros`, `UnixM`) |

> **Boundary Exception**: `float64` is allowed ONLY for external API parsing. Immediately convert to fixed-point types at the boundary.

---

## 2. ARCH Principles

| Rule | Implementation |
|------|---------------|
| Single-Goroutine Hotpath | `Sequencer.Run()` - one goroutine owns all state |
| No Mutex in Hotpath | Mutex used ONLY for I/O workers (WebSocket) |
| Bounded Channels | `make(chan Event, size)` with explicit overflow policy |
| Overflow Policy | Drop with warning log (non-blocking send) |

```
[Upbit/Bitget WS] → [Gateway Worker] → [Bounded Chan] → [Sequencer] → [State]
                          ↓                                   ↓
                      (Mutex OK)                         (NO Mutex)

[Yahoo Finance] -.환율 조회.-> [UI Layer]  (Hotpath 외부)
```

---

## 3. PERF Principles

| Rule | Implementation |
|------|---------------|
| Zero-Alloc Hotpath | Minimize heap allocations in Sequencer loop |
| sync.Pool | Profile-first: apply only after benchmarking |
| Cache-Line Friendly | Struct fields ordered by access pattern |
| No float math | Bit-shift for backoff: `1 << uint(retryCount)` |

---

## 4. DB Principles

| Rule | Implementation |
|------|---------------|
| SQLite WAL Mode | `PRAGMA journal_mode=WAL` |
| WAL-first Pattern | Persist event → Update state (never reverse) |
| Synchronous | `PRAGMA synchronous=NORMAL` |
| Optimistic Lock | Ready for `version` column when multi-writer needed |

---

## 5. BOUNDARY Principles

| Rule | Implementation |
|------|---------------|
| Sequence Gap = Halt | Panic immediately on gap detection |
| Idempotent Replay | `ReplayEvent()` produces identical state |
| Clock Sync | Tag external events with internal monotonic time |
| External Determinism | Float → int64 conversion at boundary only |

---

## 6. TEST Principles

| Rule | Implementation |
|------|---------------|
| Replay = Live | `backtest/replayer.go` - deterministic replay engine |
| Race Detection | `go test -race ./...` (requires CGO; Linux/CI recommended) |
| Fuzz Testing | `pkg/safe/math_fuzz_test.go`, `pkg/quant/types_fuzz_test.go` |
| Gap Detection Test | `sequencer_test.go` - panic verification |


---

## 7. DONE Criteria

| Criterion | Status |
|-----------|--------|
| DESIGN.md | ✅ This document |
| Panic → State Dump | ✅ `Sequencer.DumpState()` on panic |
| Balance Invariant | ✅ `domain/balance.go` - VerifyInvariant() |

---

## System Structure

```
pkg/
├── quant/          # Fixed-point types (PriceMicros, QtySats, TimeStamp)
└── safe/           # SafeMath (panic-on-overflow)

internal/
├── domain/         # Domain entities (Ticker, Alert, MarketData)
├── engine/         # Sequencer (single-threaded event processor)
├── event/          # Event definitions (MarketUpdate, OrderUpdate)
├── storage/        # SQLite WAL event store
└── infra/          # I/O workers (Upbit, Bitget WebSocket)

backtest/
└── replayer.go     # Deterministic replay engine
```

---

*Pragmatic Integrity. Deterministic Future.*
