package idx

import (
	"strings"
	"testing"
)

// flowDay builds a minimal TradingDay for flow tests.
func flowDay(date string, close, prev, volume, fbuy, fsell float64) TradingDay {
	return TradingDay{
		StockCode: "TEST", Date: date, Close: close, Previous: prev,
		Volume: volume, ForeignBuy: fbuy, ForeignSell: fsell, ForeignNet: fbuy - fsell,
	}
}

func TestBuildFlowTrend(t *testing.T) {
	// Most recent first: 3 net-buy days, then a net-sell day.
	hist := []TradingDay{
		flowDay("2026-07-03", 110, 105, 1000, 400, 100), // net +300
		flowDay("2026-07-02", 105, 100, 1000, 300, 100), // net +200
		flowDay("2026-07-01", 100, 95, 1000, 200, 100),  // net +100
		flowDay("2026-06-30", 95, 100, 1000, 100, 300),  // net -200
	}
	tr := buildFlowTrend(hist)

	if tr.From != "2026-06-30" || tr.To != "2026-07-03" || tr.TradingDays != 4 {
		t.Errorf("range = %s..%s (%d days)", tr.From, tr.To, tr.TradingDays)
	}

	// Only one window: the 5-day window is truncated to all 4 days, and
	// longer windows (20/60) would be identical so they are dropped.
	if len(tr.Windows) != 1 {
		t.Fatalf("got %d windows, want 1: %+v", len(tr.Windows), tr.Windows)
	}
	w := tr.Windows[0]
	if w.Days != 4 {
		t.Errorf("window days = %d, want 4", w.Days)
	}
	if w.NetForeignShares != 400 { // 300+200+100-200
		t.Errorf("net shares = %v, want 400", w.NetForeignShares)
	}
	// 300*110 + 200*105 + 100*100 - 200*95 = 33000+21000+10000-19000 = 45000
	if w.ApproxNetValue != 45000 {
		t.Errorf("approx net value = %v, want 45000", w.ApproxNetValue)
	}
	// foreign volume = sum(buy+sell) = 500+400+300+400 = 1600; 2*total volume = 8000 -> 20%
	if w.ForeignVolumePct != 20 {
		t.Errorf("foreign volume pct = %v, want 20", w.ForeignVolumePct)
	}
	// price change: base = Previous of oldest day (100) -> close 110 = +10%
	if w.PriceChangePct != 10 {
		t.Errorf("price change = %v, want 10", w.PriceChangePct)
	}

	// Streak: 3 consecutive net-buy days, sum +600.
	if tr.Streak.Direction != "net_buy" || tr.Streak.Days != 3 || tr.Streak.NetForeignShares != 600 {
		t.Errorf("streak = %+v, want net_buy/3/600", tr.Streak)
	}

	if !strings.Contains(strings.Join(tr.Notes, " "), "Only 4 trading days") {
		t.Errorf("expected truncation note, got %v", tr.Notes)
	}
}

func TestFlowStreakZeroAndSell(t *testing.T) {
	// Zero net on the latest day: no streak.
	s := flowStreak([]TradingDay{flowDay("d", 100, 100, 10, 5, 5)})
	if s.Direction != "none" || s.Days != 0 {
		t.Errorf("streak = %+v, want none", s)
	}

	// Net-sell streak of 2, ended by a zero day.
	s = flowStreak([]TradingDay{
		flowDay("d3", 100, 100, 10, 1, 4),
		flowDay("d2", 100, 100, 10, 2, 3),
		flowDay("d1", 100, 100, 10, 5, 5),
		flowDay("d0", 100, 100, 10, 9, 1),
	})
	if s.Direction != "net_sell" || s.Days != 2 || s.NetForeignShares != -4 {
		t.Errorf("streak = %+v, want net_sell/2/-4", s)
	}
}

func TestBuildFlowTrendFullWindows(t *testing.T) {
	// 25 identical net-buy days: expect 5-day and 20-day windows plus the
	// truncated 60-day window (= all 25 days).
	var hist []TradingDay
	for i := 0; i < 25; i++ {
		hist = append(hist, flowDay("d", 100, 100, 1000, 200, 100))
	}
	tr := buildFlowTrend(hist)
	if len(tr.Windows) != 3 {
		t.Fatalf("got %d windows, want 3", len(tr.Windows))
	}
	if tr.Windows[0].Days != 5 || tr.Windows[1].Days != 20 || tr.Windows[2].Days != 25 {
		t.Errorf("window sizes = %d/%d/%d, want 5/20/25",
			tr.Windows[0].Days, tr.Windows[1].Days, tr.Windows[2].Days)
	}
	if tr.Windows[1].NetForeignShares != 2000 { // 20 * +100
		t.Errorf("20-day net = %v, want 2000", tr.Windows[1].NetForeignShares)
	}
}
