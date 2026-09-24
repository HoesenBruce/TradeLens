package trades

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
	"testing"
)

func TestSplitSettlementAccounting(t *testing.T) {
	for _, tc := range []struct {
		name, lot                string
		reported, partial, split bool
		want                     *float64
	}{
		{"SBI IHI", "sbi:margin-long", true, false, true, f64(22362)},
		{"missing settlement", "sbi:margin-long", false, false, true, nil},
		{"partial", "sbi:margin-long", true, true, true, nil},
		{"non SBI default", "", true, false, false, f64(-3026438)},
		{"non SBI split", "", true, false, true, nil},
		{"SBI no split", "sbi:margin-long", true, false, false, f64(22362)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fills := []Execution{
				ex("a", "buy", 100, 17980, "2025-09-25T01:00:00Z", 1),
				ex("b", "buy", 100, 17510, "2025-09-26T01:00:00Z", 1),
				ex("c", "sell", 100, 2614.5, "2025-09-29T01:00:00Z", 1),
				ex("d", "sell", 100, 2611.5, "2025-09-29T02:00:00Z", 1),
			}
			for i := range fills {
				fills[i].LotKey = tc.lot
				fills[i].Fees = 9.5
			}
			if tc.reported {
				fills[2].BrokerReportedPnl = f64(11331)
				fills[3].BrokerReportedPnl = f64(11031)
			}
			if tc.partial {
				fills = fills[:3]
			}
			result := Account(fills, map[string]AccountingStrategy{"sbi:margin-long": SBIMarginAccounting})
			svc := &Service{GetBars: func(context.Context, marketdata.Request) (marketdata.Response, error) {
				r := marketdata.Response{Timezone: "Asia/Tokyo"}
				if tc.split {
					r.Bars = []marketdata.Bar{{Time: 1759104000, MarketDate: "2025-09-29", SplitRatio: 7}}
				}
				return r, nil
			}}
			require.NoError(t, svc.checkSplitBoundaries(context.Background(), fills, &result))
			require.Len(t, result.Trades, 1)
			tr := result.Trades[0]
			if tc.want == nil {
				require.Nil(t, tr.NetPnl)
			} else {
				require.Equal(t, *tc.want, *tr.NetPnl)
				require.Equal(t, *tc.want+38, *tr.GrossPnl)
			}
			if tc.split {
				require.Contains(t, tr.AccountingWarning, "2025-09-29")
				require.Nil(t, tr.ReturnPct)
			} else {
				require.Empty(t, tr.AccountingWarning)
			}
			require.Equal(t, 17980.0, fills[0].Price)
			if tc.partial {
				require.Equal(t, 100.0, tr.QtyRemaining)
			}
		})
	}
}

func TestBoundaryMarketDateAndRejectedEvidence(t *testing.T) {
	fills := []Execution{ex("a", "buy", 1, 700, "2025-09-28T23:00:00Z", 1), ex("b", "sell", 1, 100, "2025-09-29T05:00:00Z", 1)}
	for _, status := range []string{"unconfirmed", "rejected"} {
		result := Account(fills, nil)
		svc := &Service{GetBars: func(context.Context, marketdata.Request) (marketdata.Response, error) {
			return marketdata.Response{Timezone: "Asia/Tokyo", CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2025-09-29", Status: status, CandidateType: "stock_split", SuspectedRatio: 7}}}, nil
		}}
		require.NoError(t, svc.checkSplitBoundaries(context.Background(), fills, &result))
		require.NotNil(t, result.Trades[0].NetPnl) // both fills are on the post-split local day
		require.Empty(t, result.Trades[0].AccountingWarning)
	}
}

func TestReportedZeroAndShortSettlement(t *testing.T) {
	for _, tc := range []struct {
		name     string
		short    bool
		reported float64
	}{
		{"zero is evidence", false, 0}, {"short settlement", true, 1234},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := ex("a", "buy", 100, 1000, "2026-01-01T01:00:00Z", 1), ex("b", "sell", 100, 1100, "2026-01-02T01:00:00Z", 1)
			a.LotKey, b.LotKey = "sbi:margin-long", "sbi:margin-long"
			if tc.short {
				a.Side, b.Side = "sell", "buy"
				a.LotKey, b.LotKey = "sbi:margin-short", "sbi:margin-short"
			}
			a.Fees, b.Fees = 10, 20
			b.BrokerReportedPnl = &tc.reported
			result := SBIMarginAccounting([]Execution{a, b})
			require.Equal(t, tc.reported, *result.Trades[0].NetPnl)
			require.Equal(t, tc.reported+30, *result.Trades[0].GrossPnl)
		})
	}
}

func TestRejectedProviderBoundaryOverridesDetection(t *testing.T) {
	fills := []Execution{ex("a", "buy", 1, 700, "2025-09-25T01:00:00Z", 1), ex("b", "sell", 1, 100, "2025-09-29T05:00:00Z", 1)}
	result := Account(fills, nil)
	svc := &Service{GetBars: func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2025-09-29", CandidateType: "stock_split", Status: "rejected"}}, Bars: []marketdata.Bar{{MarketDate: "2025-09-29", SplitRatio: 7}}}, nil
	}}
	require.NoError(t, svc.checkSplitBoundaries(context.Background(), fills, &result))
	require.Empty(t, result.Trades[0].AccountingWarning)
	require.Equal(t, -600.0, *result.Trades[0].NetPnl)
}
