package trades

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ex(id, side string, qty, price float64, t string, mult float64) Execution {
	ts, _ := time.Parse(time.RFC3339, t)
	return Execution{ID: id, Symbol: "AAPL", InstrumentType: "stock", Side: side,
		Quantity: qty, Price: price, ExecutedAt: ts, Multiplier: mult}
}

func TestSimpleLongRoundTrip(t *testing.T) {
	fills := []Execution{
		ex("1", "buy", 100, 10.0, "2026-01-01T10:00:00Z", 1),
		ex("2", "sell", 100, 12.0, "2026-01-01T11:00:00Z", 1),
	}
	out := Group(fills)
	require.Len(t, out, 1)
	tr := out[0]
	require.Equal(t, "long", tr.Direction)
	require.Equal(t, "closed", tr.Status)
	require.Equal(t, 100.0, tr.QtyOpened)
	require.Equal(t, 10.0, tr.AvgEntryPrice)
	require.Equal(t, 12.0, *tr.AvgExitPrice)
	require.Equal(t, 200.0, *tr.NetPnl) // (12-10)*100
	require.Equal(t, []string{"1", "2"}, tr.ExecutionIDs)
}

func TestShortRoundTrip(t *testing.T) {
	out := Group([]Execution{
		ex("1", "sell", 50, 20, "2026-01-01T10:00:00Z", 1),
		ex("2", "buy", 50, 18, "2026-01-01T12:00:00Z", 1),
	})
	require.Len(t, out, 1)
	require.Equal(t, "short", out[0].Direction)
	require.Equal(t, 100.0, *out[0].NetPnl) // (18-20)*50*-1
}

func TestScaleInAverageCost(t *testing.T) {
	out := Group([]Execution{
		ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1),
		ex("2", "buy", 100, 20, "2026-01-01T10:30:00Z", 1),
		ex("3", "sell", 200, 25, "2026-01-01T11:00:00Z", 1),
	})
	require.Len(t, out, 1)
	require.Equal(t, 15.0, out[0].AvgEntryPrice) // (10+20)/2
	require.Equal(t, 2000.0, *out[0].NetPnl)     // (25-15)*200
}

func TestPartialExitsStayOneTrade(t *testing.T) {
	out := Group([]Execution{
		ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1),
		ex("2", "sell", 40, 12, "2026-01-01T10:30:00Z", 1),
		ex("3", "sell", 60, 14, "2026-01-01T11:00:00Z", 1),
	})
	require.Len(t, out, 1)
	require.Equal(t, "closed", out[0].Status)
	// avg exit = (12*40 + 14*60)/100 = 13.2 ; pnl = (13.2-10)*100 = 320
	require.Equal(t, 320.0, *out[0].NetPnl)
}

func TestZeroCrossSplitsIntoTwoTrades(t *testing.T) {
	out := Group([]Execution{
		ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1),
		ex("2", "sell", 150, 12, "2026-01-01T11:00:00Z", 1), // closes long 100, opens short 50
		ex("3", "buy", 50, 11, "2026-01-01T12:00:00Z", 1),   // closes the short
	})
	require.Len(t, out, 2)
	require.Equal(t, "long", out[0].Direction)
	require.Equal(t, 200.0, *out[0].NetPnl) // (12-10)*100
	require.Equal(t, "short", out[1].Direction)
	require.Equal(t, 50.0, *out[1].NetPnl)   // (11-12)*50*-1
	require.Equal(t, 50.0, out[1].QtyOpened) // remainder of the crossing fill
}

func TestFuturesMultiplier(t *testing.T) {
	// ES-like: 1 point = $50
	f := func(id, side string, qty, price float64, ts string) Execution {
		tt, _ := time.Parse(time.RFC3339, ts)
		return Execution{ID: id, Symbol: "ES", InstrumentType: "future", Side: side,
			Quantity: qty, Price: price, ExecutedAt: tt, Multiplier: 50}
	}
	out := Group([]Execution{
		f("1", "buy", 2, 5000, "2026-01-01T10:00:00Z"),
		f("2", "sell", 2, 5010, "2026-01-01T11:00:00Z"),
	})
	require.Equal(t, 1000.0, *out[0].NetPnl) // (5010-5000)*2*50
}

func TestFeesReduceNetPnl(t *testing.T) {
	a := ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1)
	a.Commission = 1
	b := ex("2", "sell", 100, 12, "2026-01-01T11:00:00Z", 1)
	b.Commission = 1
	out := Group([]Execution{a, b})
	require.Equal(t, 198.0, *out[0].NetPnl) // 200 - 2
	require.Equal(t, 2.0, out[0].FeesTotal)
}

func TestOpenTradeHasNoPnl(t *testing.T) {
	out := Group([]Execution{ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1)})
	require.Len(t, out, 1)
	require.Equal(t, "open", out[0].Status)
	require.Nil(t, out[0].NetPnl)
	require.Nil(t, out[0].ClosedAt)
	require.Equal(t, 100.0, out[0].QtyRemaining)
}

func TestPartialOpenKeepsRemainingQty(t *testing.T) {
	out := Group([]Execution{
		ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1),
		ex("2", "sell", 40, 12, "2026-01-01T10:30:00Z", 1),
	})
	require.Len(t, out, 1)
	require.Equal(t, "open", out[0].Status)
	require.Equal(t, 100.0, out[0].QtyOpened)
	require.Equal(t, 60.0, out[0].QtyRemaining)
}

func TestReversalFeesAreProportionalAndConserved(t *testing.T) {
	for _, side := range []string{"buy", "sell"} {
		t.Run(side, func(t *testing.T) {
			opposite := "sell"
			if side == "sell" {
				opposite = "buy"
			}
			a := ex("1", side, 100, 10, "2026-01-01T10:00:00Z", 1)
			a.Fees, a.Commission = 1, 2
			b := ex("2", opposite, 150, 12, "2026-01-01T11:00:00Z", 1)
			b.Fees, b.Commission = 4, 2
			fills := []Execution{a, b}
			out := Group(fills)
			require.Len(t, out, 2)
			require.Equal(t, 7.0, out[0].FeesTotal) // 3 opening + 4 closing
			require.Equal(t, 2.0, out[1].FeesTotal) // 50/150 of reversal cost
			require.Equal(t, 50.0, out[1].QtyRemaining)
			require.Nil(t, out[1].NetPnl)
			require.Equal(t, 9.0, out[0].FeesTotal+out[1].FeesTotal)
			require.Equal(t, b, fills[1]) // execution costs themselves are untouched
			gross := 200.0
			if side == "sell" {
				gross = -200
			}
			require.Equal(t, gross-7, *out[0].NetPnl)
			c := ex("3", side, 50, 11, "2026-01-01T12:00:00Z", 1)
			c.Commission = 1
			closed := Group(append(fills, c))
			require.Equal(t, 3.0, closed[1].FeesTotal)
			require.Equal(t, 10.0, closed[0].FeesTotal+closed[1].FeesTotal)
		})
	}
}

func TestReversalFeeRoundingConservesCents(t *testing.T) {
	a := ex("1", "buy", 1, 10, "2026-01-01T10:00:00Z", 1)
	b := ex("2", "sell", 2, 10, "2026-01-01T11:00:00Z", 1)
	b.Fees, b.Commission = .01, .02
	out := Group([]Execution{a, b})
	require.Equal(t, .02, out[0].FeesTotal)
	require.Equal(t, .01, out[1].FeesTotal)
	require.InDelta(t, .03, out[0].FeesTotal+out[1].FeesTotal, 1e-9)
}

func TestNonReversalFeesAndPartialCloseSemantics(t *testing.T) {
	a := ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1)
	a.Fees = 2
	b := ex("2", "sell", 40, 12, "2026-01-01T11:00:00Z", 1)
	b.Commission = 3
	out := Group([]Execution{a, b})
	require.Len(t, out, 1)
	require.Nil(t, out[0].NetPnl)
	require.Equal(t, 5.0, out[0].FeesTotal)
	b.Quantity = 100
	out = Group([]Execution{a, b})
	require.Equal(t, 5.0, out[0].FeesTotal)
	require.Equal(t, 195.0, *out[0].NetPnl)
}

func TestSBIReviewTradeFeesRemainUnchanged(t *testing.T) {
	for _, lot := range []string{"sbi:cash", "sbi:margin-long", "sbi:margin-short"} {
		t.Run(lot, func(t *testing.T) {
			side, opposite := "buy", "sell"
			if lot == "sbi:margin-short" {
				side, opposite = opposite, side
			}
			a := ex("1", side, 100, 10, "2026-01-01T10:00:00Z", 1)
			b := ex("2", opposite, 150, 12, "2026-01-01T11:00:00Z", 1)
			a.LotKey, b.LotKey, b.Commission = lot, lot, 6
			strategy := SBIMarginAccounting
			if lot == "sbi:cash" {
				strategy = SBICashAccounting
			}
			out := Account([]Execution{a, b}, map[string]AccountingStrategy{lot: strategy}).Trades
			require.Len(t, out, 2)
			require.Equal(t, 6.0, out[0].FeesTotal)
			require.Zero(t, out[1].FeesTotal)
		})
	}
}
