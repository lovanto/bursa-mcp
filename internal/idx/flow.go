package idx

import (
	"context"
	"fmt"
)

// flowWindows are the look-back horizons summarised by ForeignFlowTrend.
var flowWindows = []int{5, 20, 60}

// FlowWindow summarises foreign activity over the most recent N trading days.
// Foreign buy/sell in the IDX feed are share counts, so the net is in shares;
// ApproxNetValue prices each day's net at that day's close.
type FlowWindow struct {
	Days             int     `json:"days"` // window actually used (may be shorter than requested)
	NetForeignShares float64 `json:"net_foreign_shares"`
	ApproxNetValue   float64 `json:"approx_net_value_idr"`
	ForeignVolumePct float64 `json:"foreign_volume_pct"` // (buy+sell)/(2*volume), percent
	PriceChangePct   float64 `json:"price_change_pct"`   // close now vs close at window start
}

// FlowStreak is the current run of consecutive net-buy or net-sell days,
// counted from the most recent day backwards. A zero-net day ends the streak.
type FlowStreak struct {
	Direction        string  `json:"direction"` // "net_buy", "net_sell", or "none"
	Days             int     `json:"days"`
	NetForeignShares float64 `json:"net_foreign_shares"`
}

// ForeignFlowTrend is the analytic roll-up of official foreign flow for one
// stock, derived entirely from the trading-info feed (no extra endpoint).
type ForeignFlowTrend struct {
	Code        string       `json:"code"`
	Name        string       `json:"name,omitempty"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	TradingDays int          `json:"trading_days"`
	Windows     []FlowWindow `json:"windows"`
	Streak      FlowStreak   `json:"streak"`
	Notes       []string     `json:"notes,omitempty"`
}

// ForeignFlowTrend summarises foreign accumulation/distribution for `code`
// over up to `days` recent trading days (default 60, clamped like TradingInfo).
func (c *Client) ForeignFlowTrend(ctx context.Context, code string, days int) (*ForeignFlowTrend, error) {
	if days < 1 {
		days = 60
	}
	hist, err := c.TradingInfo(ctx, code, days)
	if err != nil {
		return nil, err
	}
	if len(hist) == 0 {
		return nil, fmt.Errorf("no trading data for %s", normalizeCode(code))
	}
	return buildFlowTrend(hist), nil
}

// buildFlowTrend computes the roll-up from a most-recent-first daily series.
func buildFlowTrend(hist []TradingDay) *ForeignFlowTrend {
	t := &ForeignFlowTrend{
		Code:        hist[0].StockCode,
		Name:        hist[0].StockName,
		To:          hist[0].Date,
		From:        hist[len(hist)-1].Date,
		TradingDays: len(hist),
		Notes: []string{
			"Foreign buy/sell are official IDX share counts; approx_net_value_idr prices each day's net flow at that day's close.",
		},
	}

	for _, w := range flowWindows {
		n := w
		if n > len(hist) {
			n = len(hist)
		}
		win := FlowWindow{Days: n}
		var totalVolume, foreignVolume float64
		for _, d := range hist[:n] {
			win.NetForeignShares += d.ForeignNet
			win.ApproxNetValue += d.ForeignNet * d.Close
			totalVolume += d.Volume
			foreignVolume += d.ForeignBuy + d.ForeignSell
		}
		if totalVolume > 0 {
			win.ForeignVolumePct = foreignVolume / (2 * totalVolume) * 100
		}
		// hist[n-1].Previous is the close just before the window opened.
		if base := hist[n-1].Previous; base > 0 {
			win.PriceChangePct = (hist[0].Close - base) / base * 100
		}
		t.Windows = append(t.Windows, win)
		if n < w {
			t.Notes = append(t.Notes, fmt.Sprintf("Only %d trading days available for the %d-day window.", n, w))
		}
		if n == len(hist) {
			break // longer windows would be identical
		}
	}

	t.Streak = flowStreak(hist)
	return t
}

// flowStreak measures the current consecutive net-buy/net-sell run.
func flowStreak(hist []TradingDay) FlowStreak {
	s := FlowStreak{Direction: "none"}
	if len(hist) == 0 || hist[0].ForeignNet == 0 {
		return s
	}
	positive := hist[0].ForeignNet > 0
	if positive {
		s.Direction = "net_buy"
	} else {
		s.Direction = "net_sell"
	}
	for _, d := range hist {
		if d.ForeignNet == 0 || (d.ForeignNet > 0) != positive {
			break
		}
		s.Days++
		s.NetForeignShares += d.ForeignNet
	}
	return s
}
