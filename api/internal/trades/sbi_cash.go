package trades

import (
	"math"
	"slices"
	"sort"
	"time"

	"github.com/tradermemos/api/internal/money"
)

var tokyo = time.FixedZone("Asia/Tokyo", 9*60*60)

// SBICashAccounting calculates broker/accounting P&L by trading date. Trades
// remain chronological review trades; their existing NetPnl semantics do not
// change when a same-day acquisition changes the broker cost basis.
func SBICashAccounting(fills []Execution) AccountingResult {
	result := AccountingResult{Trades: Group(append([]Execution(nil), fills...))}
	ordered := append([]Execution(nil), fills...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i].ExecutedAt.In(tokyo), ordered[j].ExecutedAt.In(tokyo)
		if a.Year() != b.Year() || a.YearDay() != b.YearDay() {
			return a.Before(b)
		}
		if ordered[i].Side != ordered[j].Side {
			return ordered[i].Side == "buy"
		}
		return ordered[i].ID < ordered[j].ID
	})
	var qty, cost float64
	var currentDay string
	for _, fill := range ordered {
		day := fill.ExecutedAt.In(tokyo).Format(time.DateOnly)
		if currentDay != "" && day != currentDay && qty > 0 {
			cost = math.Ceil(cost/qty-1e-9) * qty
		}
		currentDay = day
		if fill.Quantity <= 0 || fill.Price < 0 {
			continue
		}
		if fill.Side == "buy" {
			qty += fill.Quantity
			cost += fill.Price*fill.Quantity + fill.Fees + fill.Commission
			continue
		}
		if fill.Side != "sell" || fill.Quantity > qty+1e-9 {
			continue
		}
		average := math.Ceil(cost/qty - 1e-9)
		qty -= fill.Quantity
		cost = average * qty
		if qty < 1e-9 {
			qty, cost = 0, 0
		}
		result.RealizedCloses = append(result.RealizedCloses, RealizedClose{
			ExecutionID: fill.ID, Date: fill.ExecutedAt.In(tokyo),
			Pnl:          money.Round2((fill.Price-average)*fill.Quantity - fill.Fees - fill.Commission),
			RemainingQty: qty, RemainingCostBasis: cost, Source: "calculated_average_cost",
		})
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
	return result
}
