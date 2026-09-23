package trades

import (
	"sort"

	"github.com/tradermemos/api/internal/money"
)

// SBIMarginAccounting prefers the settlement result supplied by SBI. Without
// it, only a single opening can be costed without inventing a lot match.
func SBIMarginAccounting(fills []Execution) AccountingResult {
	result := AccountingResult{Trades: Group(append([]Execution(nil), fills...))}
	ordered := append([]Execution(nil), fills...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].ExecutedAt.Equal(ordered[j].ExecutedAt) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].ExecutedAt.Before(ordered[j].ExecutedAt)
	})
	var qty, entry, openFees float64
	openings := 0
	for _, fill := range ordered {
		if fill.Quantity <= 0 || fill.Price < 0 {
			continue
		}
		opening := (fill.LotKey == "sbi:margin-long" && fill.Side == "buy") ||
			(fill.LotKey == "sbi:margin-short" && fill.Side == "sell")
		if opening {
			qty += fill.Quantity
			entry = fill.Price
			openFees += fill.Fees + fill.Commission
			openings++
			continue
		}
		if fill.Quantity > qty+1e-9 || qty == 0 {
			result.UnavailableCloseIDs = append(result.UnavailableCloseIDs, fill.ID)
			continue
		}
		remaining := qty - fill.Quantity
		allocatedFees := openFees * fill.Quantity / qty
		openFees -= allocatedFees
		qty = remaining
		if fill.ConversionType == "genbiki" {
			zeroConversionTrade(result.Trades, fill.ID)
			result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
				ExecutionID: fill.ID, Date: fill.ExecutedAt, Pnl: 0,
				RemainingQty: qty, RemainingCostBasis: entry * qty, Source: "position_conversion",
			})
		} else if fill.BrokerReportedPnl != nil {
			result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
				ExecutionID: fill.ID, Date: fill.ExecutedAt, Pnl: *fill.BrokerReportedPnl,
				RemainingQty: qty, Source: "broker_reported",
			})
		} else if openings == 1 {
			sign := 1.0
			if fill.LotKey == "sbi:margin-short" {
				sign = -1
			}
			mult := fill.Multiplier
			if mult == 0 {
				mult = 1
			}
			result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
				ExecutionID: fill.ID, Date: fill.ExecutedAt,
				Pnl:          money.Round2((fill.Price-entry)*fill.Quantity*sign*mult - allocatedFees - fill.Fees - fill.Commission),
				RemainingQty: qty, RemainingCostBasis: entry * qty, Source: "calculated_single_opening",
			})
		} else {
			result.UnavailableCloseIDs = append(result.UnavailableCloseIDs, fill.ID)
		}
		if qty < 1e-9 {
			qty, openFees, openings = 0, 0, 0
		}
	}
	return result
}

func zeroConversionTrade(trades []Trade, executionID string) {
	for i := range trades {
		for _, id := range trades[i].ExecutionIDs {
			if id == executionID {
				zero := 0.0
				trades[i].GrossPnl, trades[i].NetPnl, trades[i].ReturnPct = &zero, &zero, &zero
				return
			}
		}
	}
}
