package importer

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func TestSBICashReportedResultMatching(t *testing.T) {
	input := []byte("実現損益詳細\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2026/09/03,1515,売却,100,1000,900,9990\n2026/09/03,1515,返済売,100,1000,900,9990\n合計,,,,,,19980\n")
	report, ok, err := ReadSBIMarginPnLCSV(input)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, report.Parsed, 2)
	at, _ := time.Parse(time.RFC3339, "2026-09-03T00:00:00Z")
	cash := store.Execution{ID: "cash", InstrumentType: "stock", Multiplier: 1, Symbol: "1515", Side: "sell", Quantity: 100, Price: 1000, ExecutedAt: at, Details: sql.NullString{String: `{"lot":"sbi:cash","position_effect":"reduce"}`, Valid: true}}
	margin := cash
	margin.ID = "margin"
	margin.Details.String = `{"lot":"sbi:margin-long","position_effect":"reduce"}`
	rows := MatchSBIMarginPnL(report.Parsed, []store.Execution{margin, cash})
	require.Equal(t, "cash", rows[0].ExecutionID)
	require.Equal(t, "margin", rows[1].ExecutionID)
	require.Equal(t, "enrichable", rows[0].Status)
	for _, details := range []string{`{}`, `{"broker":"SBI","position_type":"cash","position_effect":"reduce"}`, `{"lot":"ibkr:cash","position_effect":"reduce"}`, `{"lot":"sbi:cash","position_effect":"increase"}`} {
		nonSBI := cash
		nonSBI.Details.String = details
		require.Equal(t, "no_matching_execution", MatchSBIMarginPnL(report.Parsed[:1], []store.Execution{nonSBI})[0].Status)
	}
	// One source row cannot guess which identical fill it describes.
	second := cash
	second.ID = "second"
	require.Equal(t, "ambiguous_match", MatchSBIMarginPnL(report.Parsed[:1], []store.Execution{cash, second})[0].Status)
	repeated := []SBIMarginPnLRow{report.Parsed[0], report.Parsed[0]}
	matches := MatchSBIMarginPnL(repeated, []store.Execution{second, cash})
	require.Equal(t, "cash", matches[0].ExecutionID)
	require.Equal(t, "second", matches[1].ExecutionID)
	require.Equal(t, "enrichable", matches[1].Status)
	repeated[1].Basis = 901
	require.Equal(t, "ambiguous_match", MatchSBIMarginPnL(repeated, []store.Execution{cash, second})[0].Status)
	require.Equal(t, "conflict", MatchSBIMarginPnL(repeated, []store.Execution{cash})[0].Status)
	cash.Details.String = `{"lot":"sbi:cash","position_effect":"reduce","broker_reported_close_basis":900,"broker_reported_realized_pnl":9990}`
	require.Equal(t, "already_enriched", MatchSBIMarginPnL(report.Parsed[:1], []store.Execution{cash})[0].Status)
}
