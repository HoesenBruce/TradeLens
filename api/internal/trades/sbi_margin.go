package trades

import (
	"sort"

	"github.com/tradermemos/api/internal/money"
)

// SBIMarginAccounting prefers the settlement result supplied by SBI. Without
// it, only a single opening can be costed without inventing a lot match.
func SBIMarginAccounting(fills []Execution) AccountingResult {
	result := AccountingResult{Trades: group(append([]Execution(nil), fills...), false)}
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
			entry = (entry*(qty-fill.Quantity) + fill.Price*fill.Quantity) / qty
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
		if fill.SettlementType == "genwatashi" && qty > 1e-9 {
			entry = (entry*(qty+fill.Quantity) - fill.Price*fill.Quantity) / qty
		}
		if fill.ConversionType == "genbiki" || fill.SettlementType == "genwatashi" {
			// The cash leg owns disposal P&L for a settlement.
			source := "position_conversion"
			if fill.SettlementType == "genwatashi" {
				source = "position_settlement"
			}
			result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
				ExecutionID: fill.ID, Date: fill.ExecutedAt, Pnl: 0,
				RemainingQty: qty, RemainingCostBasis: entry * qty, Source: source,
			})
		} else if fill.BrokerReportedPnl != nil {
			result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
				ExecutionID: fill.ID, Date: fill.ExecutedAt, Pnl: *fill.BrokerReportedPnl, Fees: allocatedFees + fill.Fees + fill.Commission,
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
				RemainingQty: qty, RemainingCostBasis: entry * qty, Fees: allocatedFees + fill.Fees + fill.Commission, Source: "calculated_single_opening",
			})
		} else {
			result.UnavailableCloseIDs = append(result.UnavailableCloseIDs, fill.ID)
		}
		if qty < 1e-9 {
			qty, openFees, openings = 0, 0, 0
		}
	}
	reconcileReportedTrades(fills, &result)
	reconcileConversionTrades(fills, &result)
	return result
}

func reconcileConversionTrades(fills []Execution, result *AccountingResult) {
	// Only conversion-containing trades opt in; genuine partial closes survive.
	conversions := map[string]bool{}
	closes := map[string]RealizedClose{}
	unavailableIDs := map[string]bool{}
	for _, fill := range fills {
		if fill.ConversionType == "genbiki" || fill.SettlementType == "genwatashi" {
			conversions[fill.ID] = true
		}
	}
	if len(conversions) == 0 {
		return
	}
	for _, close := range result.RealizedCloses {
		closes[close.ExecutionID] = close
	}
	for _, id := range result.UnavailableCloseIDs {
		unavailableIDs[id] = true
	}
	for i := range result.Trades {
		tr := &result.Trades[i]
		conversion, unavailable := false, false
		net, fees := 0.0, 0.0
		for _, id := range tr.ExecutionIDs {
			conversion = conversion || conversions[id]
			unavailable = unavailable || unavailableIDs[id]
			net += closes[id].Pnl
			fees += closes[id].Fees
		}
		if !conversion {
			continue
		}
		if unavailable {
			tr.NetPnl, tr.GrossPnl, tr.ReturnPct = nil, nil, nil
			continue
		}
		net = money.Round2(net)
		tr.NetPnl = &net
		if len(fills) > 0 && (fills[0].LotKey == "sbi:margin-long" || fills[0].LotKey == "sbi:margin-short") {
			tr.FeesTotal = money.Round2(fees)
		}
		gross := money.Round2(net + tr.FeesTotal)
		tr.GrossPnl = &gross
		ret := 0.0
		if tr.AvgEntryPrice*tr.QtyOpened != 0 {
			ret = money.Round2(net / (tr.AvgEntryPrice * tr.QtyOpened) * 100)
		}
		tr.ReturnPct = &ret
	}
}
