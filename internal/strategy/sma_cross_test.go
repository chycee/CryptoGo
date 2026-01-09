package strategy_test

import (
	"crypto_go/internal/domain"
	"crypto_go/internal/strategy"
	"crypto_go/pkg/quant"
	"testing"
)

func TestSMACrossStrategy(t *testing.T) {
	// Setup: Short=3, Long=5
	strat := strategy.NewSMACrossStrategy("BTC", 3, 5)

	// Helper to push price and check action
	push := func(price int64) []domain.Order {
		state := domain.MarketState{
			Symbol:      "BTC",
			PriceMicros: quant.PriceMicros(price),
		}
		out := make([]domain.Order, 1)
		count := strat.OnMarketUpdate(state, out)
		return out[:count]
	}

	// Sequence:
	// T1: 100 -> [100] (Not enough)
	// T2: 100 -> [100, 100]
	// T3: 100 -> [100, 100, 100] (S=100)
	// T4: 100 -> [100, 100, 100, 100] (S=100)
	// T5: 100 -> [..., 100] (S=100, L=100). Prev=0. Actions=[]
	//
	// T6: 200 -> [100, 100, 100, 100, 200]
	//    Short(3) = (100+100+200)/3 = 133
	//    Long(5)  = (100+100+100+100+200)/5 = 120
	//    Prev(S=100, L=100) -> Curr(S=133 > L=120) => GOLDEN CROSS (BUY)

	// T1-T5: All 100
	for i := 0; i < 5; i++ {
		orders := push(100)
		if len(orders) > 0 {
			t.Errorf("T%d: Expected no orders, got %v", i, orders)
		}
	}

	// T6: Price jumps to 200
	orders := push(200)
	if len(orders) != 1 {
		t.Fatalf("T6: Expected 1 order (BUY), got %d", len(orders))
	}
	if orders[0].Side != "BUY" {
		t.Errorf("T6: Expected BUY, got %s", orders[0].Side)
	}

	// T7: Price drops to 50
	// Prices: [100, 100, 100, 200, 50]
	// Short(3) = (100+200+50)/3 = 350/3 = 116
	// Long(5)  = (100+100+100+200+50)/5 = 550/5 = 110
	// Prev(S=133, L=120) -> Curr(S=116 > L=110)
	// Still above, no cross.
	orders = push(50)
	if len(orders) != 0 {
		t.Errorf("T7: Expected no orders, got %v", orders)
	}

	// T8: Price drops to 0
	// Prices: [100, 100, 200, 50, 0]
	// Short(3) = (200+50+0)/3 = 83
	// Long(5)  = 450/5 = 90
	// Prev(S=116, L=110) -> Curr(S=83 < L=90) => DEAD CROSS (SELL)
	orders = push(0)
	if len(orders) != 1 {
		t.Fatalf("T8: Expected 1 order (SELL), got %d", len(orders))
	}
	if orders[0].Side != "SELL" {
		t.Errorf("T8: Expected SELL, got %s", orders[0].Side)
	}
}
