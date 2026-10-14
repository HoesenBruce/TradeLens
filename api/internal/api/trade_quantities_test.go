package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func TestTradeQuantitiesReadModel(t *testing.T) {
	for _, tc := range []struct {
		name, status, warning, lifecycle string
		entry, open, exit, pct           float64
	}{
		{"partial long", "open", "", "partially_closed", 500, 100, 400, 20},
		{"pyramiding", "open", "", "partially_closed", 700, 300, 400, 300.0 / 700 * 100},
		{"closed", "closed", "", "closed", 500, 0, 500, 0},
		{"partial short", "open", "", "partially_closed", 70, 10, 60, 10.0 / 70 * 100},
		{"reversal remainder", "open", "", "open", 200, 200, 0, 100},
		{"open only", "open", "", "open", 500, 500, 0, 100},
		{"normalized split", "open", "Stock split adjusted: 2:1.", "partially_closed", 400, 50, 350, 12.5},
		{"normalized reverse split", "open", "Stock split adjusted: 0.5:1.", "partially_closed", 100, 25, 75, 25},
		{"date only", "open", "SBI date-level reconciliation: intraday order unknown.", "partially_closed", 70, 10, 60, 10.0 / 70 * 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := store.Trade{Status: tc.status, QtyOpened: tc.entry, QtyRemaining: tc.open, AccountingWarning: tc.warning}
			before := row
			got := toTradeDTO(row, nil).Quantities
			require.Equal(t, tc.entry, *got.EntryQty)
			require.Equal(t, tc.open, *got.OpenQty)
			require.Equal(t, tc.exit, *got.ExitQty)
			require.InDelta(t, tc.pct, *got.RemainingPct, 1e-9)
			require.Equal(t, tc.lifecycle, got.Lifecycle)
			require.Equal(t, before, row)
		})
	}
}

func TestTradeQuantitiesUnknownIsNullNotZero(t *testing.T) {
	for _, row := range []store.Trade{
		{Status: "closed", QtyOpened: 0},
		{Status: "closed", QtyOpened: 5, AccountingWarning: "Incomplete SBI opening evidence: excess disposal."},
		{Status: "open", QtyOpened: 200, QtyRemaining: 50, AccountingWarning: "Corporate-action boundary: stock_split; units not adjusted."},
		{Status: "open", QtyOpened: 500},
		{Status: "open", QtyOpened: 100, QtyRemaining: 101},
	} {
		got := toTradeDTO(row, nil)
		require.Equal(t, "unknown", got.Quantities.Lifecycle)
		require.NotEmpty(t, got.Quantities.UnknownReason)
		require.Nil(t, got.Quantities.EntryQty)
		require.Nil(t, got.Quantities.ExitQty)
		require.Nil(t, got.Quantities.OpenQty)
		require.Nil(t, got.Quantities.RemainingPct)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		require.Contains(t, string(raw), `"open_qty":null`)
		require.Contains(t, string(raw), `"qty_opened":`)
	}
}
