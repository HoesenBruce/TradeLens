package api

import (
	"math"
	"strings"

	"github.com/tradermemos/api/internal/store"
)

// Read only the reconstructed inventory, whose units already include approved
// splits and whose trade boundaries already handle reversals/conversions.
type tradeQuantitiesDTO struct {
	EntryQty      *float64 `json:"entry_qty"`
	ExitQty       *float64 `json:"exit_qty"`
	OpenQty       *float64 `json:"open_qty"`
	RemainingPct  *float64 `json:"remaining_pct"`
	Lifecycle     string   `json:"lifecycle"`
	UnknownReason string   `json:"unknown_reason,omitempty"`
}

func tradeQuantities(t store.Trade) tradeQuantitiesDTO {
	unknown := func(reason string) tradeQuantitiesDTO {
		return tradeQuantitiesDTO{Lifecycle: "unknown", UnknownReason: reason}
	}
	if strings.Contains(t.AccountingWarning, "Incomplete SBI opening evidence") || t.QtyOpened <= 0 {
		return unknown("incomplete_opening")
	}
	if strings.Contains(t.AccountingWarning, "Corporate-action boundary:") {
		return unknown("unverified_share_units")
	}
	entry, open := t.QtyOpened, t.QtyRemaining
	if math.IsNaN(entry) || math.IsNaN(open) || math.IsInf(entry, 0) || math.IsInf(open, 0) || open < 0 || open > entry ||
		(t.Status == "closed" && open != 0) || (t.Status == "open" && open == 0) || (t.Status != "open" && t.Status != "closed") {
		return unknown("inconsistent_inventory")
	}
	exit, pct := entry-open, open/entry*100
	lifecycle := "open"
	if open == 0 {
		lifecycle = "closed"
	} else if exit > 0 {
		lifecycle = "partially_closed"
	}
	return tradeQuantitiesDTO{EntryQty: &entry, ExitQty: &exit, OpenQty: &open, RemainingPct: &pct, Lifecycle: lifecycle}
}
