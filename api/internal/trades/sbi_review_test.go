package trades

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSBIUnresolvedOpeningRetainsMoneyWithoutLotDuration(t *testing.T) {
	for _, lot := range []string{"sbi:margin-long", "sbi:margin-short"} {
		t.Run(lot, func(t *testing.T) {
			openSide, closeSide := "buy", "sell"
			if lot == "sbi:margin-short" {
				openSide, closeSide = "sell", "buy"
			}
			fills := []Execution{
				ex("open", openSide, 40, 100, "2031-04-07T09:00:00Z", 1),
				ex("close", closeSide, 40, 110, "2031-04-08T09:00:00Z", 1),
			}
			for i := range fills {
				fills[i].LotKey = lot
			}
			fills[1].BrokerReportedPnl = f64(396)
			before := SBIMarginAccounting(fills).Trades[0]
			require.NotNil(t, before.TimeInTradeSecs)
			fills[1].OpeningMatchProvenance = "unresolved"
			after := SBIMarginAccounting(fills).Trades[0]
			require.Equal(t, before.NetPnl, after.NetPnl)
			require.Equal(t, 396.0, *after.NetPnl)
			require.Nil(t, after.TimeInTradeSecs)
			require.Contains(t, after.AccountingWarning, "opening allocation unresolved")
			// Default/IBKR accounting ignores the SBI-specific provenance marker.
			for i := range fills {
				fills[i].LotKey = "ibkr:margin"
			}
			generic := Account(fills, nil).Trades[0]
			require.NotNil(t, generic.TimeInTradeSecs)
			require.Empty(t, generic.AccountingWarning)
		})
	}
}

func TestSBIReviewDateOnlyRowPermutations(t *testing.T) {
	for _, bucket := range []string{"sbi:cash", "sbi:margin-long", "sbi:margin-short"} {
		t.Run(bucket, func(t *testing.T) {
			opening, closing, direction := "buy", "sell", "long"
			if bucket == "sbi:margin-short" {
				opening, closing, direction = "sell", "buy", "short"
			}
			base := []Execution{
				ex("close1", closing, 10, 110, "2026-09-01T00:00:00Z", 1),
				ex("open1", opening, 10, 100, "2026-09-01T00:00:00Z", 1),
				ex("open2", opening, 10, 102, "2026-09-01T00:00:00Z", 1),
				ex("close2", closing, 10, 112, "2026-09-01T00:00:00Z", 1),
			}
			for a := range 4 {
				for b := range 4 {
					for c := range 4 {
						for d := range 4 {
							if a == b || a == c || a == d || b == c || b == d || c == d {
								continue
							}
							fills := []Execution{base[a], base[b], base[c], base[d]}
							for i := range fills {
								fills[i].LotKey, fills[i].SourceTimePrecision = bucket, "date"
								fills[i].ID = fmt.Sprint(i) // ID order cannot imply a fill match either
								fills[i].ExecutedAt = fills[i].ExecutedAt.Add(time.Duration(i) * time.Microsecond)
								fills[i].Fees = 1
								if !sbiOpening(fills[i]) && bucket != "sbi:cash" {
									fills[i].BrokerReportedPnl = f64(123)
								}
							}
							var result AccountingResult
							if bucket == "sbi:cash" {
								result = SBICashAccounting(fills)
							} else {
								result = SBIMarginAccounting(fills)
							}
							require.Len(t, result.Trades, 1)
							tr := result.Trades[0]
							require.Equal(t, direction, tr.Direction)
							require.Equal(t, "closed", tr.Status)
							require.Equal(t, 101.0, tr.AvgEntryPrice)
							require.Equal(t, 111.0, *tr.AvgExitPrice)
							require.Equal(t, 20.0, tr.QtyOpened)
							require.Zero(t, tr.QtyRemaining)
							require.Equal(t, 4.0, tr.FeesTotal)
							require.Nil(t, tr.TimeInTradeSecs)
							require.Contains(t, tr.AccountingWarning, "intraday order")
							if bucket != "sbi:cash" {
								require.Equal(t, 246.0, *tr.NetPnl)
							}
						}
					}
				}
			}
		})
	}
}

func TestSBIReviewIncompleteEvidenceAndActualTimes(t *testing.T) {
	for _, bucket := range []string{"sbi:cash", "sbi:margin-long", "sbi:margin-short"} {
		for _, precision := range []string{"date", "time"} {
			for _, qty := range []float64{0, 5, 10} {
				t.Run(fmt.Sprintf("%s/%s/%g", bucket, precision, qty), func(t *testing.T) {
					openSide, closeSide, dir := "buy", "sell", "long"
					if bucket == "sbi:margin-short" {
						openSide, closeSide, dir = "sell", "buy", "short"
					}
					fills := []Execution{ex("close", closeSide, 10, 110, "2026-09-02T00:00:00Z", 1), ex("later", openSide, 10, 100, "2026-09-03T00:00:00Z", 1)}
					if qty > 0 {
						fills = append(fills, ex("prior", openSide, qty, 100, "2026-09-01T00:00:00Z", 1))
					}
					for i := range fills {
						fills[i].LotKey = bucket
						fills[i].SourceTimePrecision = precision
					}
					var result AccountingResult
					if bucket == "sbi:cash" {
						result = SBICashAccounting(fills)
					} else {
						result = SBIMarginAccounting(fills)
					}
					require.Len(t, result.Trades, 2)
					tr := result.Trades[0]
					require.Equal(t, dir, tr.Direction)
					require.Equal(t, qty, tr.QtyOpened)
					require.Zero(t, tr.QtyRemaining)
					if qty < 10 {
						require.Contains(t, tr.AccountingWarning, "Incomplete SBI opening evidence")
						require.Nil(t, tr.NetPnl)
						require.Contains(t, result.UnavailableCloseIDs, "close")
					}
					require.Equal(t, 10.0, result.Trades[1].QtyRemaining)
				})
			}
		}
	}
}

func TestSBIReviewPartialShortAndActualClosingFirst(t *testing.T) {
	open := ex("open", "sell", 70, 110, "2026-09-01T01:00:00Z", 1)
	close := ex("close", "buy", 60, 100, "2026-09-01T00:00:00Z", 1)
	open.LotKey, close.LotKey = "sbi:margin-short", "sbi:margin-short"
	open.SourceTimePrecision, close.SourceTimePrecision = "date", "date"
	result := SBIMarginAccounting([]Execution{close, open})
	require.Len(t, result.Trades, 1)
	require.Equal(t, "short", result.Trades[0].Direction)
	require.Equal(t, 70.0, result.Trades[0].QtyOpened)
	require.Equal(t, 10.0, result.Trades[0].QtyRemaining)
	require.Nil(t, result.Trades[0].NetPnl)
	open.SourceTimePrecision, close.SourceTimePrecision = "time", "time"
	result = SBIMarginAccounting([]Execution{close, open})
	require.Len(t, result.Trades, 2)
	require.Zero(t, result.Trades[0].QtyOpened)
	require.Equal(t, 70.0, result.Trades[1].QtyRemaining)
}
