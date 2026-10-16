package trades

import (
	"context"
	"github.com/tradermemos/api/internal/marketdata"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSBICashDailyAverageCost(t *testing.T) {
	tests := []struct {
		name string
		rows []Execution
		pnls []float64
		qty  float64
		cost float64
	}{
		{"multi day partial", []Execution{
			ex("1", "buy", 100, 1000, "2026-09-01T01:00:00Z", 1),
			ex("2", "buy", 100, 1100, "2026-09-02T01:00:00Z", 1),
			ex("3", "sell", 100, 1090, "2026-09-03T01:00:00Z", 1),
		}, []float64{4000}, 100, 105000},
		{"buy sell buy", []Execution{
			ex("1", "buy", 100, 1000, "2026-09-01T00:10:00Z", 1),
			ex("2", "sell", 100, 1100, "2026-09-01T01:30:00Z", 1),
			ex("3", "buy", 100, 1200, "2026-09-01T05:00:00Z", 1),
		}, []float64{0}, 100, 110000},
		{"prior holding sell buy", []Execution{
			ex("1", "buy", 100, 900, "2026-08-31T01:00:00Z", 1),
			ex("2", "sell", 100, 1100, "2026-09-01T00:10:00Z", 1),
			ex("3", "buy", 100, 1200, "2026-09-01T05:00:00Z", 1),
		}, []float64{5000}, 100, 105000},
		{"multiple same day", []Execution{
			ex("1", "buy", 100, 1000, "2026-09-01T00:10:00Z", 1),
			ex("2", "sell", 50, 1100, "2026-09-01T01:00:00Z", 1),
			ex("3", "buy", 100, 1200, "2026-09-01T02:00:00Z", 1),
			ex("4", "sell", 50, 1300, "2026-09-01T05:00:00Z", 1),
		}, []float64{0, 10000}, 100, 110000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i := range tc.rows {
				tc.rows[i].LotKey = "sbi:cash"
				tc.rows[i].SourceTimePrecision = "date"
			}
			result := Account(tc.rows, map[string]AccountingStrategy{"sbi:cash": SBICashAccounting})
			require.Len(t, result.RealizedCloses, len(tc.pnls))
			for i, want := range tc.pnls {
				require.Equal(t, want, result.RealizedCloses[i].Pnl)
				require.Equal(t, "calculated_average_cost", result.RealizedCloses[i].Source)
			}
			last := result.RealizedCloses[len(result.RealizedCloses)-1]
			require.Equal(t, tc.qty, last.RemainingQty)
			require.Equal(t, tc.cost, last.RemainingCostBasis)
		})
	}
}

func TestSBICashFeesRoundAcquisitionCostUp(t *testing.T) {
	buy := ex("buy", "buy", 100, 120, "2026-09-01T01:00:00Z", 1)
	buy.LotKey = "sbi:cash"
	buy.Fees = 55
	sell := ex("sell", "sell", 40, 123, "2026-09-02T01:00:00Z", 1)
	sell.LotKey = "sbi:cash"
	sell.Fees = 10
	result := SBICashAccounting([]Execution{sell, buy})
	require.Equal(t, 70.0, result.RealizedCloses[0].Pnl) // (123-121)*40-10
	require.Equal(t, 60.0, result.RealizedCloses[0].RemainingQty)
	require.Equal(t, 7260.0, result.RealizedCloses[0].RemainingCostBasis)
}

func TestConvertedCashUsesRemainingAcquisitionBasis(t *testing.T) {
	opening := ex("open", "buy", 100, 900, "2026-09-01T01:00:00Z", 1)
	sale := ex("sale", "sell", 50, 950, "2026-09-02T01:00:00Z", 1)
	conversion := ex("conversion", "buy", 100, 1005, "2026-09-03T01:00:00Z", 1)
	for _, f := range []*Execution{&opening, &sale, &conversion} {
		f.LotKey = "sbi:cash"
	}
	conversion.ConversionType = "genbiki"
	result := SBICashAccounting([]Execution{opening, sale, conversion})
	require.Equal(t, 150.0, result.Trades[0].QtyRemaining)
	require.Equal(t, 970.0, result.Trades[0].AvgEntryPrice)
	require.Equal(t, 2500.0, *result.Trades[0].NetPnl)
}

func TestSBICashCloseTotalsAndReportedPrecedence(t *testing.T) {
	buy := ex("open", "buy", 2, 100, "2026-09-01T01:00:00Z", 1)
	a := ex("a", "sell", 1, 130, "2026-09-02T01:00:00Z", 1)
	b := ex("b", "sell", 1, 90, "2026-09-03T01:00:00Z", 1)
	buy.Fees, a.Fees, b.Fees = 2, 7, 5
	for _, f := range []*Execution{&buy, &a, &b} {
		f.LotKey = "sbi:cash"
	}
	a.BrokerReportedPnl = f64(120)
	b.BrokerReportedPnl = f64(-30)
	a.BrokerReportedCloseBasis = f64(99)
	b.BrokerReportedCloseBasis = f64(99)
	partial := SBICashAccounting([]Execution{buy, a})
	require.Nil(t, partial.Trades[0].NetPnl)
	require.Equal(t, 120.0, partial.RealizedCloses[0].Pnl)
	require.Equal(t, 99.0, partial.RealizedCloses[0].RemainingCostBasis)
	result := SBICashAccounting([]Execution{buy, a, b})
	require.Equal(t, 90.0, *result.Trades[0].NetPnl)
	require.Equal(t, 104.0, *result.Trades[0].GrossPnl)
	require.Equal(t, "broker_reported", result.RealizedCloses[0].Source)
}

func TestSBICashRoundingAndActualReentry(t *testing.T) {
	a := ex("a", "buy", 100, 100.1, "2026-09-01T01:00:00Z", 1)
	b := ex("b", "buy", 100, 101.2, "2026-09-02T01:00:00Z", 1)
	c := ex("c", "sell", 200, 110, "2026-09-03T01:00:00Z", 1)
	for _, f := range []*Execution{&a, &b, &c} {
		f.LotKey = "sbi:cash"
	}
	result := SBICashAccounting([]Execution{a, b, c})
	require.Equal(t, 1800.0, result.RealizedCloses[0].Pnl)
	require.Equal(t, 1800.0, *result.Trades[0].NetPnl)
	require.Contains(t, result.Trades[0].AccountingWarning, "estimate")
	b.Side, b.Quantity, b.Price = "sell", 100, 110
	b.ExecutedAt = a.ExecutedAt.Add(time.Hour)
	c.Side, c.Quantity, c.Price = "buy", 100, 120
	c.ExecutedAt = b.ExecutedAt.Add(time.Hour)
	result = SBICashAccounting([]Execution{c, b, a})
	require.Len(t, result.Trades, 2)
	require.Equal(t, 900.0, *result.Trades[0].NetPnl)
	require.Nil(t, result.Trades[1].NetPnl)
	require.Equal(t, 120.0, result.Trades[1].AvgEntryPrice)
}

func TestSBICashReportedSplitAndNonSBIIsolation(t *testing.T) {
	a := ex("a", "buy", 100, 1000, "2026-09-01T01:00:00Z", 1)
	b := ex("b", "sell", 100, 150, "2026-09-03T01:00:00Z", 1)
	for _, f := range []*Execution{&a, &b} {
		f.LotKey = "sbi:cash"
	}
	b.BrokerReportedPnl = f64(120)
	result := SBICashAccounting([]Execution{a, b})
	svc := &Service{GetBars: func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{Timezone: "Asia/Tokyo", CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2026-09-02", CandidateType: "stock_split", SuspectedRatio: 7}}}, nil
	}}
	require.NoError(t, svc.checkSplitBoundaries(context.Background(), []Execution{a, b}, &result))
	require.Equal(t, 120.0, *result.Trades[0].NetPnl)
	require.Nil(t, result.Trades[0].ReturnPct)
	// Generic/IBKR buckets continue using the default engine, including open partial closes.
	for _, lot := range []string{"", "ibkr:cash", "unknown:cash"} {
		a.LotKey, b.LotKey = lot, lot
		b.Quantity = 50
		got := Account([]Execution{a, b}, map[string]AccountingStrategy{"sbi:cash": SBICashAccounting})
		require.Equal(t, Group([]Execution{a, b}), got.Trades)
		require.Empty(t, got.RealizedCloses)
		require.Nil(t, got.Trades[0].NetPnl)
	}
}
