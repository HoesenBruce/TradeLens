package trades

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/money"
)

var tokyo = time.FixedZone("Asia/Tokyo", 9*60*60)

// SBICashAccounting calculates broker/accounting P&L by trading date. Trades
// use date-level reconciliation only when execution times are unavailable.
func SBICashAccounting(fills []Execution) AccountingResult {
	result := AccountingResult{Trades: sbiReviewTrades(fills)}
	ordered := sbiReviewOrder(fills)
	var qty, cost float64
	for _, fill := range ordered {
		if fill.Quantity <= 0 || fill.Price < 0 {
			continue
		}
		if fill.Side == "buy" {
			qty += fill.Quantity
			cost += fill.Price*fill.Quantity + fill.Fees + fill.Commission
			continue
		}
		if fill.Side != "sell" || fill.Quantity > qty+1e-9 {
			result.UnavailableCloseIDs = append(result.UnavailableCloseIDs, fill.ID)
			if fill.BrokerReportedPnl != nil {
				result.RealizedCloses = append(result.RealizedCloses, RealizedClose{ExecutionID: fill.ID, Date: fill.ExecutedAt.In(tokyo), Pnl: *fill.BrokerReportedPnl, Fees: fill.Fees + fill.Commission, Source: "broker_reported"})
			}
			// Consume supported holdings; a later purchase cannot repair this sale.
			if fill.Side == "sell" {
				qty, cost = 0, 0
			}
			continue
		}
		average := math.Ceil(cost/qty - 1e-9)
		if fill.BrokerReportedCloseBasis != nil {
			average = *fill.BrokerReportedCloseBasis
		}
		qty -= fill.Quantity
		cost = average * qty
		if qty < 1e-9 {
			qty, cost = 0, 0
		}
		pnl := money.Round2((fill.Price-average)*fill.Quantity - fill.Fees - fill.Commission)
		source := "calculated_average_cost"
		if fill.BrokerReportedPnl != nil {
			pnl, source = *fill.BrokerReportedPnl, "broker_reported"
		}
		if fill.SettlementResult != nil {
			pnl, source = fill.SettlementResult.RealizedPnL, fill.SettlementResult.Source
		}
		result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
			ExecutionID: fill.ID, Date: fill.ExecutedAt.In(tokyo),
			Pnl: pnl, Fees: fill.Fees + fill.Commission,
			RemainingQty: qty, RemainingCostBasis: cost, Source: source,
		})
	}
	reconcileReportedTrades(fills, &result, true)
	estimates := map[string]bool{}
	for _, close := range result.RealizedCloses {
		if close.Source == "calculated_average_cost" {
			estimates[close.ExecutionID] = true
		}
	}
	for i := range result.Trades {
		tr := &result.Trades[i]
		for _, id := range tr.ExecutionIDs {
			if estimates[id] {
				tr.AccountingWarning = strings.TrimSpace(tr.AccountingWarning + " SBI cash P&L uses an average-cost estimate where broker-reported pretax results are missing.")
				break
			}
		}
	}
	reconcileConversionTrades(fills, &result)
	if qty > 0 {
		for i := range result.Trades {
			tr := &result.Trades[i]
			if tr.Status != "open" {
				continue
			}
			for _, fill := range fills {
				if fill.ConversionType != "genbiki" {
					continue
				}
				if slices.Contains(tr.ExecutionIDs, fill.ID) {
					tr.AvgEntryPrice = money.Round2(cost / qty)
					break
				}
			}
		}
	}
	protectIncompleteSBITrades(&result)
	return result
}
