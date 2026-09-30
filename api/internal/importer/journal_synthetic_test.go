package importer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/trades"
)

func TestSyntheticJournalPreviewAndNetPnl(t *testing.T) {
	// Invented fills exercise stock and option multipliers without a broker export.
	rows := []map[string]string{
		{"Date": "2026-01-02T11:00:00Z", "Open Date": "2026-01-02T10:00:00Z", "Symbol": "TEST", "Market": "STOCK", "Side": "LONG", "Qty": "2", "Entry": "10", "Exit": "12", "Return ($)": "4"},
		{"Date": "2026-01-03T11:00:00Z", "Open Date": "2026-01-03T10:00:00Z", "Symbol": "TEST260116C00010000", "Market": "OPTION", "Side": "LONG", "Qty": "1", "Entry": "1.5", "Exit": "1", "Return ($)": "-50"},
	}

	summary, samples := BuildJournalPreview(rows)
	require.Equal(t, 2, summary.RowCount)
	require.Equal(t, 2, summary.TradeCount)
	require.Equal(t, 4, summary.ExecutionCount)
	require.Equal(t, 1, summary.StockTrades)
	require.Equal(t, 1, summary.OptionTrades)
	require.InDelta(t, -46, summary.NetPnl, 0.01)
	require.Len(t, samples, 2)

	parsed := NewJournal().ParseRows(rows)
	require.Empty(t, parsed.Errors)
	var pnl float64
	for i := 0; i < len(parsed.Executions); i += 2 {
		fills := parsed.Executions[i : i+2]
		grouped := trades.Group([]trades.Execution{
			{Symbol: fills[0].Symbol, InstrumentType: fills[0].InstrumentType, Side: fills[0].Side, Quantity: fills[0].Quantity, Price: fills[0].Price, ExecutedAt: fills[0].ExecutedAt, Multiplier: fills[0].Multiplier, LotKey: fills[0].LotKey},
			{Symbol: fills[1].Symbol, InstrumentType: fills[1].InstrumentType, Side: fills[1].Side, Quantity: fills[1].Quantity, Price: fills[1].Price, ExecutedAt: fills[1].ExecutedAt, Multiplier: fills[1].Multiplier, LotKey: fills[1].LotKey},
		})
		require.Len(t, grouped, 1)
		pnl += *grouped[0].NetPnl
	}
	require.InDelta(t, -46, pnl, 0.01)
}
