package trades

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountingSelectionPreservesDefault(t *testing.T) {
	base := []Execution{
		ex("1", "buy", 100, 10, "2026-01-01T10:00:00Z", 1),
		ex("2", "buy", 100, 20, "2026-01-01T11:00:00Z", 1),
		ex("3", "sell", 200, 25, "2026-01-01T12:00:00Z", 1),
	}
	called := false
	strategies := map[string]AccountingStrategy{"sbi:cash": func([]Execution) AccountingResult {
		called = true
		return AccountingResult{}
	}}
	got := Account(base, strategies)
	require.False(t, called)
	require.Equal(t, 2000.0, *got.Trades[0].NetPnl)
	require.Empty(t, got.RealizedCloses)

	partial := base[:1]
	partial = append(partial, ex("4", "sell", 40, 12, "2026-01-01T12:00:00Z", 1))
	got = Account(partial, strategies)
	require.Equal(t, "open", got.Trades[0].Status)
	require.Nil(t, got.Trades[0].NetPnl)
	require.Equal(t, 60.0, got.Trades[0].QtyRemaining)

	partial[0].LotKey, partial[1].LotKey = "sbi:cash", "sbi:cash"
	Account(partial, strategies)
	require.True(t, called)
}
