package trades

import (
	"testing"

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
