package trades

import (
	"sort"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/money"
)

const sbiDateWarning = "SBI date-level reconciliation: intraday order, fill pairing and exact elapsed holding time are unknown; calendar holding days use market-local dates."
const sbiMissingWarning = "Incomplete SBI opening evidence: disposal/close exceeds supported holdings; acquisition basis and P&L are unavailable."

func sbiOpening(f Execution) bool {
	if f.LotKey == "sbi:margin-short" {
		return f.Side == "sell"
	}
	return f.Side == "buy"
}

// Reconcile uncertain days as a whole; storage offsets are not fill chronology.
// Days with actual times only keep their execution order.
func sbiReviewOrder(fills []Execution) []Execution {
	ordered := append([]Execution(nil), fills...)
	uncertain := map[string]bool{}
	for _, f := range fills {
		if f.SourceTimePrecision == "date" {
			uncertain[f.ExecutedAt.In(tokyo).Format(time.DateOnly)] = true
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		dayA, dayB := a.ExecutedAt.In(tokyo).Format(time.DateOnly), b.ExecutedAt.In(tokyo).Format(time.DateOnly)
		if dayA != dayB {
			return dayA < dayB
		}
		if uncertain[dayA] && sbiOpening(a) != sbiOpening(b) {
			return sbiOpening(a)
		}
		// Within a role retain the storage order for stable historical IDs, not lot pairing.
		if !a.ExecutedAt.Equal(b.ExecutedAt) {
			return a.ExecutedAt.Before(b.ExecutedAt)
		}
		return a.ID < b.ID
	})
	return ordered
}

// Explicit SBI buckets cannot reverse into the opposite position. An unmatched
// closing record remains reviewable, without inventing an acquisition.
func sbiReviewTrades(fills []Execution) []Trade {
	var result []Trade
	var cur *openState
	var latest time.Time
	missing, uncertain, inferred, unresolved := false, false, false, false
	finish := func(closed bool) {
		var tr Trade
		if closed {
			tr = cur.finalize(latest)
		} else {
			tr = cur.finalizeOpen()
		}
		if uncertain {
			tr.AccountingWarning = sbiDateWarning
			tr.TimeInTradeSecs = nil
		}
		if inferred {
			tr.TimeInTradeSecs = nil
			tr.AccountingWarning = strings.TrimSpace(tr.AccountingWarning + " fifo_inferred: TradeLens equivalent-lot allocation; SBI repayment order and lot-level holding duration are unknown.")
		}
		if unresolved {
			tr.TimeInTradeSecs = nil
			tr.AccountingWarning = strings.TrimSpace(tr.AccountingWarning + " SBI opening allocation unresolved: broker-reported close result retained; opening lots and lot-level holding duration are unknown.")
		}
		if missing {
			tr.TimeInTradeSecs = nil
			tr.AccountingWarning = strings.TrimSpace(tr.AccountingWarning + " " + sbiMissingWarning)
			tr.NetPnl, tr.GrossPnl, tr.ReturnPct = nil, nil, nil
		}
		result = append(result, tr)
		cur, missing, uncertain, inferred, unresolved = nil, false, false, false, false
	}
	for _, f := range sbiReviewOrder(fills) {
		if f.Quantity <= 0 || f.Price < 0 {
			continue
		}
		inferred = inferred || f.OpeningMatchProvenance == "fifo_inferred"
		unresolved = unresolved || f.OpeningMatchProvenance == "unresolved"
		if cur == nil || f.ExecutedAt.After(latest) {
			latest = f.ExecutedAt
		}
		if sbiOpening(f) {
			signed := f.Quantity
			if f.LotKey == "sbi:margin-short" {
				signed = -signed
			}
			if cur == nil {
				cur = newOpen(f, signed, f.Multiplier, true)
			} else {
				cur.scaleIn(f, signed)
				if f.ExecutedAt.Before(cur.openedAt) {
					cur.openedAt = f.ExecutedAt
				}
			}
			uncertain = uncertain || f.SourceTimePrecision == "date"
			continue
		}
		if cur == nil {
			direction := "long"
			if f.LotKey == "sbi:margin-short" {
				direction = "short"
			}
			warning := sbiMissingWarning
			if f.SourceTimePrecision == "date" {
				warning += " " + sbiDateWarning
			}
			at := f.ExecutedAt
			result = append(result, Trade{Symbol: f.Symbol, InstrumentType: f.InstrumentType,
				Direction: direction, Status: "closed", OpenedAt: at, ClosedAt: &at,
				AvgExitPrice: f64(f.Price), FeesTotal: money.Round2(f.Fees + f.Commission),
				AccountingWarning: warning, ExecutionIDs: []string{f.ID}})
			unresolved = false
			continue
		}
		uncertain = uncertain || f.SourceTimePrecision == "date"
		missing = missing || f.Quantity > abs(cur.position)+1e-9
		cur.reduce(f, min(f.Quantity, abs(cur.position)), f.Multiplier)
		if abs(cur.position) < 1e-9 {
			finish(true)
		}
	}
	if cur != nil {
		finish(false)
	}
	return result
}

func protectIncompleteSBITrades(result *AccountingResult) {
	for i := range result.Trades {
		tr := &result.Trades[i]
		if strings.Contains(tr.AccountingWarning, sbiMissingWarning) {
			tr.NetPnl, tr.GrossPnl, tr.ReturnPct = nil, nil, nil
		}
	}
}
