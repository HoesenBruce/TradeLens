package importer

import (
	"database/sql"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
	"testing"
	"time"
)

func TestGenwatashiReportedPnlMatchesDisposalOnly(t *testing.T) {
	csv := []byte("約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2026/09/03,1515,現渡,100,1000,903,9100\n2026/09/03,1515,返済買,100,1000,1000,0\n")
	report, ok, err := ReadSBIMarginPnLCSV(csv)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, report.Parsed, 2)
	date, _ := time.Parse(time.RFC3339, "2026-09-03T00:00:00Z")
	cash := store.Execution{ID: "cash", Symbol: "1515", Side: "sell", Quantity: 100, Price: 1000, ExecutedAt: date, Details: sql.NullString{String: `{"lot":"sbi:cash","position_effect":"reduce","settlement_type":"genwatashi"}`, Valid: true}}
	short := cash
	short.ID = "short"
	short.Side = "buy"
	short.Details.String = `{"lot":"sbi:margin-short","position_effect":"reduce","settlement_type":"genwatashi"}`
	matches := MatchSBIMarginPnL(report.Parsed, []store.Execution{cash, short})
	require.Equal(t, "cash", matches[0].ExecutionID)
	require.Equal(t, "enrichable", matches[0].Status)
	require.Equal(t, "no_matching_execution", matches[1].Status)
}

func TestGenwatashiMappingDoesNotTreatReceiptAsPnl(t *testing.T) {
	rows := []map[string]string{{"約定日": "2026/09/03", "銘柄コード": "1515", "取引": "現渡", "約定数量": "100", "約定単価": "1000", "手数料/諸経費等": "600", "受渡金額/決済損益": "99400"}}
	parsed := ParseSBIRows(rows, nil, "")
	require.Empty(t, parsed.Errors)
	require.Len(t, parsed.Executions, 2)
	require.Nil(t, parsed.Executions[0].ReportedRealizedPnl)
	require.Equal(t, 99400.0, *parsed.Executions[0].SettlementProceeds)
	require.Equal(t, parsed.Executions[0].SettlementID, parsed.Executions[1].SettlementID)
	require.Zero(t, parsed.Executions[1].Fees)
	rows[0]["受渡金額/決済損益"] = "NaN"
	parsed = ParseSBIRows(rows, nil, "")
	require.NotEmpty(t, parsed.Errors)
	require.Empty(t, parsed.Executions)
}
