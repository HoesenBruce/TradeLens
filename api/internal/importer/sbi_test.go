package importer

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestSBITradeExecutionCSV(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sbi-trade-executions.csv")
	require.NoError(t, err)
	cp932, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), fixture)
	require.NoError(t, err)

	headers, rows, ok, err := ReadSBITradeCSV(cp932)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, IsSBI(headers))
	require.Len(t, rows, 11)

	result := ParseSBIRows(rows, "")
	require.Len(t, result.Executions, 11)
	require.Equal(t, []RowError{
		{Row: 10, Message: "invalid quantity"},
	}, result.Errors)

	cashBuy, cashSell := result.Executions[0], result.Executions[1]
	require.Equal(t, "6501", cashBuy.Symbol)
	require.Equal(t, "buy", cashBuy.Side)
	require.Equal(t, "sell", cashSell.Side)
	require.Equal(t, "stock", cashBuy.InstrumentType)
	require.Equal(t, "sbi:cash", cashBuy.LotKey)
	require.Equal(t, time.Date(2026, 1, 4, 15, 0, 0, 0, time.UTC), cashBuy.ExecutedAt)
	require.Equal(t, 10.0, cashSell.Fees)

	require.Equal(t, "sbi:margin-long", result.Executions[2].LotKey)
	require.Equal(t, "buy", result.Executions[2].Side)
	require.Equal(t, "sell", result.Executions[3].Side)
	require.Equal(t, "sbi:margin-short", result.Executions[4].LotKey)
	require.Equal(t, "sell", result.Executions[4].Side)
	require.Equal(t, "buy", result.Executions[5].Side)
	require.Equal(t, "584A", result.Executions[6].Symbol)
	require.NotEqual(t, result.Executions[6].DedupKey, result.Executions[7].DedupKey)

	marginOpen, marginClose, cashOpen := result.Executions[8], result.Executions[9], result.Executions[10]
	require.Equal(t, "buy", marginOpen.Side)
	require.Equal(t, "sbi:margin-long", marginOpen.LotKey)
	require.Equal(t, "sell", marginClose.Side)
	require.Equal(t, "sbi:margin-long", marginClose.LotKey)
	require.Equal(t, 20.0, marginClose.Fees)
	require.Equal(t, "buy", cashOpen.Side)
	require.Equal(t, "sbi:cash", cashOpen.LotKey)
	require.Zero(t, cashOpen.Fees)
	require.True(t, marginClose.ExecutedAt.Before(cashOpen.ExecutedAt))
	require.NotEqual(t, marginClose.DedupKey, cashOpen.DedupKey)
}

func TestSBICashTransactionCSV(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sbi-cash-transactions.csv")
	require.NoError(t, err)
	fixture = bytes.TrimPrefix(fixture, []byte("\xef\xbb\xbf"))
	cp932, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), fixture)
	require.NoError(t, err)

	got, ok, err := ReadSBICashCSV(cp932)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, got.Rows, 5)
	require.Len(t, got.Transactions, 4)
	require.Equal(t, []RowError{{Row: 5, Message: `invalid cash transaction date "bad-date"`}}, got.Errors)

	require.Equal(t, "deposit", got.Transactions[0].Type)
	require.Equal(t, 100000.0, got.Transactions[0].Amount)
	require.Equal(t, "JPY", got.Transactions[0].Currency)
	require.Equal(t, time.Date(2026, 1, 4, 15, 0, 0, 0, time.UTC), got.Transactions[0].OccurredAt)
	require.Equal(t, "withdrawal", got.Transactions[1].Type)
	require.Equal(t, -50000.0, got.Transactions[1].Amount)
	require.Equal(t, "dividend", got.Transactions[2].Type)
	require.Equal(t, "adjustment", got.Transactions[3].Type)
	require.Equal(t, -300.0, got.Transactions[3].Amount)
}
