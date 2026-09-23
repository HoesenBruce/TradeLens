package trades

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSBIMarginAccountingReportedAndFallback(t *testing.T) {
	first := ex("first", "buy", 100, 1000, "2026-09-01T01:00:00Z", 1)
	second := ex("second", "buy", 100, 1100, "2026-09-02T01:00:00Z", 1)
	close := ex("close", "sell", 100, 1090, "2026-09-03T01:00:00Z", 1)
	for _, fill := range []*Execution{&first, &second, &close} {
		fill.LotKey = "sbi:margin-long"
	}
	pnl := 8993.0
	close.BrokerReportedPnl = &pnl
	result := Account([]Execution{first, second, close}, map[string]AccountingStrategy{"sbi:margin-long": SBIMarginAccounting})
	require.Equal(t, pnl, result.RealizedCloses[0].Pnl)
	require.Equal(t, "broker_reported", result.RealizedCloses[0].Source)
	close.BrokerReportedPnl = nil
	result = SBIMarginAccounting([]Execution{first, second, close})
	require.Empty(t, result.RealizedCloses)
	require.Equal(t, []string{"close"}, result.UnavailableCloseIDs)
	result = SBIMarginAccounting([]Execution{first, close})
	require.Equal(t, 9000.0, result.RealizedCloses[0].Pnl)
	require.Equal(t, "calculated_single_opening", result.RealizedCloses[0].Source)
}
