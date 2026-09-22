package importer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenericImporterParsesAndReportsBadRows(t *testing.T) {
	rows := []map[string]string{
		{"Symbol": "AAPL", "Stock Name": "Apple", "B/S": "BUY", "Qty": "100", "Fill Price": "10.00", "Trade Date": "2026-01-01T10:00:00Z", "Commission": "1.00"},
		{"Symbol": "BADROW", "B/S": "BUY", "Qty": "notanumber", "Fill Price": "5", "Trade Date": "2026-01-01T10:00:00Z", "Commission": "0"},
	}
	mapping := map[string]string{
		"symbol": "Symbol", "stock_name": "Stock Name", "side": "B/S", "quantity": "Qty",
		"price": "Fill Price", "executed_at": "Trade Date", "commission": "Commission",
	}
	imp := NewGeneric(mapping)
	res := imp.ParseRows(rows)
	require.Len(t, res.Executions, 1)
	require.Equal(t, "buy", res.Executions[0].Side)
	require.Equal(t, "Apple", res.Executions[0].StockName)
	require.Equal(t, 100.0, res.Executions[0].Quantity)
	require.Len(t, res.Errors, 1)
	require.Equal(t, 2, res.Errors[0].Row)
}

func TestParseMoney(t *testing.T) {
	for input, want := range map[string]float64{
		"10.00": 10, "$23.145": 23.145, "$1,234.56": 1234.56,
		"($1.02)": -1.02, "-$1.25": -1.25, "$-0.40": -0.40,
		"US$400.00": 400, "USD 1.00": 1, "23.145$": 23.145,
	} {
		got, err := parseMoney(input)
		require.NoError(t, err, input)
		require.Equal(t, want, got, input)
	}
	for _, input := range []string{"", "$", "abc", "()", "(abc)"} {
		_, err := parseMoney(input)
		require.Error(t, err, input)
	}
}

func TestGenericImporterMoneyAndNonFillRows(t *testing.T) {
	mapping := map[string]string{
		"symbol": "Symbol", "side": "Action", "quantity": "Quantity",
		"price": "Price", "executed_at": "Date", "commission": "Commission",
		"fees": "Fees", "swap": "Swap",
	}
	res := NewGeneric(mapping).ParseRows([]map[string]string{
		{"Symbol": "ACME", "Action": "Buy", "Quantity": "100", "Price": "$1,234.56", "Date": "2026-09-14", "Commission": "($1.02)", "Fees": "$-0.40", "Swap": "USD 1.00"},
		{"Symbol": "ACME", "Action": "Qualified Dividend", "Quantity": "", "Price": "", "Date": "2026-09-15"},
		{"Symbol": "ACME", "Action": "UnknownAction", "Quantity": "100", "Price": "20.00", "Date": "2026-09-16"},
	})
	require.Len(t, res.Executions, 1)
	require.Equal(t, 1234.56, res.Executions[0].Price)
	require.Equal(t, 1.02, res.Executions[0].Commission)
	require.Equal(t, 1.40, res.Executions[0].Fees)
	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0].Message, "invalid side")
}

func TestGenericImporterRejectsInvalidCosts(t *testing.T) {
	mapping := map[string]string{
		"symbol": "Symbol", "side": "Side", "quantity": "Qty",
		"price": "Price", "executed_at": "Date", "fees": "Fees",
	}
	res := NewGeneric(mapping).ParseRows([]map[string]string{{
		"Symbol": "ACME", "Side": "Buy", "Qty": "1", "Price": "10",
		"Date": "2026-09-14", "Fees": "abc",
	}})
	require.Empty(t, res.Executions)
	require.Len(t, res.Errors, 1)
	require.Equal(t, "invalid fees", res.Errors[0].Message)
}

// Naive broker timestamps are the broker's wall clock, not UTC. A Friday
// 16:30 ET fill read as UTC would land on the wrong instant (and late-Friday
// fills on Saturday once bucketed) — WithSourceTZ pins the source zone.
func TestGenericImporterSourceTimezone(t *testing.T) {
	mapping := map[string]string{
		"symbol": "Symbol", "side": "Side", "quantity": "Qty",
		"price": "Price", "executed_at": "Exec Time",
	}
	row := func(ts string) []map[string]string {
		return []map[string]string{{
			"Symbol": "SPY", "Side": "BUY", "Qty": "1", "Price": "550", "Exec Time": ts,
		}}
	}

	// Offset-less time in source zone: 2026-07-10 is EDT (UTC-4).
	res := NewGeneric(mapping).WithSourceTZ("America/New_York").ParseRows(row("2026-07-10 16:30:00"))
	require.Empty(t, res.Errors)
	require.Equal(t, time.Date(2026, 7, 10, 20, 30, 0, 0, time.UTC), res.Executions[0].ExecutedAt)

	// Date-only rows land at source-zone midnight, not UTC midnight (which
	// would render as the previous evening — or Sunday — in market time).
	res = NewGeneric(mapping).WithSourceTZ("America/New_York").ParseRows(row("07/13/2026"))
	require.Empty(t, res.Errors)
	require.Equal(t, time.Date(2026, 7, 13, 4, 0, 0, 0, time.UTC), res.Executions[0].ExecutedAt)

	// An explicit RFC3339 offset always wins over the source zone.
	res = NewGeneric(mapping).WithSourceTZ("America/New_York").ParseRows(row("2026-07-10T16:30:00+08:00"))
	require.Empty(t, res.Errors)
	require.Equal(t, time.Date(2026, 7, 10, 8, 30, 0, 0, time.UTC), res.Executions[0].ExecutedAt)

	// A trailing US tz abbreviation (Webull) also wins over the source zone.
	res = NewGeneric(mapping).WithSourceTZ("America/Chicago").ParseRows(row("07/10/2026 09:31:22 EDT"))
	require.Empty(t, res.Errors)
	require.Equal(t, time.Date(2026, 7, 10, 13, 31, 22, 0, time.UTC), res.Executions[0].ExecutedAt)

	// Unknown or empty source zones keep the legacy UTC interpretation.
	res = NewGeneric(mapping).WithSourceTZ("Not/AZone").ParseRows(row("2026-07-10 16:30:00"))
	require.Empty(t, res.Errors)
	require.Equal(t, time.Date(2026, 7, 10, 16, 30, 0, 0, time.UTC), res.Executions[0].ExecutedAt)
}
