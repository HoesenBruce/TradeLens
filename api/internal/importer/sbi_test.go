package importer

import (
	"bytes"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/trades"
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

	result := ParseSBIRows(rows, nil, "")
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
	require.Nil(t, result.Executions[2].ReportedRealizedPnl)
	require.Equal(t, "buy", result.Executions[2].Side)
	require.Equal(t, "sell", result.Executions[3].Side)
	require.Equal(t, 90.0, *result.Executions[3].ReportedRealizedPnl)
	require.Equal(t, "sbi:margin-short", result.Executions[4].LotKey)
	require.Equal(t, "sell", result.Executions[4].Side)
	require.Equal(t, "buy", result.Executions[5].Side)
	require.Equal(t, 388.0, *result.Executions[5].ReportedRealizedPnl)
	require.Equal(t, "584A", result.Executions[6].Symbol)
	require.NotEqual(t, result.Executions[6].DedupKey, result.Executions[7].DedupKey)

	marginOpen, marginClose, cashOpen := result.Executions[8], result.Executions[9], result.Executions[10]
	require.Equal(t, "buy", marginOpen.Side)
	require.Equal(t, "sbi:margin-long", marginOpen.LotKey)
	require.Equal(t, "sell", marginClose.Side)
	require.Equal(t, "sbi:margin-long", marginClose.LotKey)
	require.Equal(t, 20.0, marginClose.Fees)
	require.Nil(t, marginClose.ReportedRealizedPnl) // 現引 field is transfer amount
	require.Equal(t, "buy", cashOpen.Side)
	require.Equal(t, "sbi:cash", cashOpen.LotKey)
	require.Zero(t, cashOpen.Fees)
	require.True(t, marginClose.ExecutedAt.Before(cashOpen.ExecutedAt))
	require.NotEqual(t, marginClose.DedupKey, cashOpen.DedupKey)
	require.Equal(t, "position_conversion", marginClose.EventType)
	require.Equal(t, "genbiki", marginClose.ConversionType)
	require.Equal(t, marginClose.ConversionID, cashOpen.ConversionID)
}

func TestSBIRowsUseConfirmedMapping(t *testing.T) {
	result := ParseSBIRows([]map[string]string{{
		"約定日": "2026/01/04", "銘柄コード": "5401", "銘柄": "既定名", "確認した銘柄名": "日本製鉄",
		"取引": "株式現物買", "約定数量": "100", "約定単価": "3250",
	}}, map[string]string{"stock_name": "確認した銘柄名"}, "")
	require.Empty(t, result.Errors)
	require.Len(t, result.Executions, 1)
	require.Equal(t, "日本製鉄", result.Executions[0].StockName)
}

func TestSBIDateOnlyRowsCashAccountingKeepsMarginSeparate(t *testing.T) {
	rows := []map[string]string{
		{"約定日": "2026/09/01", "銘柄コード": "5401", "銘柄": "A", "取引": "株式現物買", "約定数量": "100", "約定単価": "1000"},
		{"約定日": "2026/09/01", "銘柄コード": "5401", "銘柄": "A", "取引": "株式現物売", "約定数量": "100", "約定単価": "1100"},
		{"約定日": "2026/09/01", "銘柄コード": "5401", "銘柄": "A", "取引": "株式現物買", "約定数量": "100", "約定単価": "1200"},
		{"約定日": "2026/09/01", "銘柄コード": "5401", "銘柄": "A", "取引": "信用新規買", "約定数量": "100", "約定単価": "500"},
	}
	parsed := ParseSBIRows(rows, nil, "")
	require.Empty(t, parsed.Errors)
	var cash []trades.Execution
	for i, fill := range parsed.Executions {
		if fill.LotKey != "sbi:cash" {
			continue
		}
		cash = append(cash, trades.Execution{
			ID: strconv.Itoa(i + 1), Symbol: fill.Symbol, InstrumentType: fill.InstrumentType,
			Side: fill.Side, Quantity: fill.Quantity, Price: fill.Price, ExecutedAt: fill.ExecutedAt,
			LotKey: fill.LotKey,
		})
	}
	result := trades.Account(cash, map[string]trades.AccountingStrategy{"sbi:cash": trades.SBICashAccounting})
	require.Equal(t, 0.0, result.RealizedCloses[0].Pnl)
	require.Equal(t, 100.0, result.RealizedCloses[0].RemainingQty)
	require.Equal(t, 110000.0, result.RealizedCloses[0].RemainingCostBasis)
}

func TestSBIMarginSettlementMissingZeroAndInvalid(t *testing.T) {
	row := map[string]string{
		"約定日": "2026/09/01", "銘柄コード": "5401", "銘柄": "A", "取引": "信用返済買",
		"約定数量": "100", "約定単価": "900", "受渡金額/決済損益": "--",
	}
	result := ParseSBIRows([]map[string]string{row}, nil, "")
	require.Empty(t, result.Errors)
	require.Nil(t, result.Executions[0].ReportedRealizedPnl)
	row["受渡金額/決済損益"] = "0"
	result = ParseSBIRows([]map[string]string{row}, nil, "")
	require.Equal(t, 0.0, *result.Executions[0].ReportedRealizedPnl)
	row["受渡金額/決済損益"] = "oops"
	result = ParseSBIRows([]map[string]string{row}, nil, "")
	require.Equal(t, []RowError{{Row: 1, Message: "invalid margin settlement P&L"}}, result.Errors)
	require.Empty(t, result.Executions)
}

func TestSBISemanticLabels(t *testing.T) {
	cases := []struct{ label, position, effect string }{
		{"株式現物買", "cash", "increase"}, {"現物買", "cash", "increase"},
		{"株式現物売", "cash", "reduce"}, {"現物売", "cash", "reduce"},
		{"信用新規買", "margin_long", "increase"}, {"信用返済売", "margin_long", "reduce"},
		{"信用新規売", "margin_short", "increase"}, {"信用返済買", "margin_short", "reduce"},
	}
	for _, tc := range cases {
		position, effect, ok := sbiSemantics(tc.label)
		require.True(t, ok, tc.label)
		require.Equal(t, tc.position, position)
		require.Equal(t, tc.effect, effect)
	}
	_, _, ok := sbiSemantics("unknown")
	require.False(t, ok)
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
