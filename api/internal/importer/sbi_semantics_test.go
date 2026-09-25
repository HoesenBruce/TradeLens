package importer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/importer"
	"github.com/tradermemos/api/internal/store"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestSBIMixedPositionSemanticsPersistAndRegroup(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sbi-mixed-positions.csv")
	require.NoError(t, err)
	cp932, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), fixture)
	require.NoError(t, err)
	_, rows, ok, err := importer.ReadSBITradeCSV(cp932)
	require.NoError(t, err)
	require.True(t, ok)
	parsed := importer.ParseSBIRows(rows, nil, "")
	require.Empty(t, parsed.Errors)
	require.Len(t, parsed.Executions, 6)
	wantTypes := []string{"cash", "margin_long", "margin_short", "cash", "margin_long", "margin_short"}
	wantEffects := []string{"increase", "increase", "increase", "reduce", "reduce", "reduce"}
	for i, fill := range parsed.Executions {
		require.Equal(t, "584A", fill.Symbol)
		require.Equal(t, "匿名銘柄", fill.StockName)
		require.Equal(t, wantTypes[i], fill.PositionType)
		require.Equal(t, wantEffects[i], fill.PositionEffect)
		if i > 0 {
			require.True(t, parsed.Executions[i-1].ExecutedAt.Before(fill.ExecutedAt))
		}
	}

	conn, err := db.Open(filepath.Join(t.TempDir(), "sbi.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := context.Background()
	user, err := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.NewString(), Email: "sbi@x.com", PasswordHash: "x"})
	require.NoError(t, err)
	account, err := q.CreateAccount(ctx, store.CreateAccountParams{
		ID: uuid.NewString(), UserID: user.ID, Name: "SBI", Broker: "sbi", AccountType: "cash",
		AccountKind: "brokerage", Capabilities: `["cash","margin"]`, BaseCurrency: "JPY",
	})
	require.NoError(t, err)
	result, err := importer.Commit(ctx, q, user.ID, account.ID, sql.NullString{}, parsed)
	require.NoError(t, err)
	require.Equal(t, 6, result.Inserted)
	result, err = importer.Commit(ctx, q, user.ID, account.ID, sql.NullString{}, parsed)
	require.NoError(t, err)
	require.Zero(t, result.Inserted)
	require.Equal(t, 6, result.Skipped)

	fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: user.ID, AccountID: account.ID})
	require.NoError(t, err)
	require.Len(t, fills, 6)
	for i, fill := range fills {
		var details map[string]any
		require.NoError(t, json.Unmarshal([]byte(fill.Details.String), &details))
		require.Equal(t, wantTypes[i], details["position_type"])
		require.Equal(t, "date", details["source_time_precision"])
		require.Equal(t, wantEffects[i], details["position_effect"])
		require.Equal(t, "sbi:"+map[string]string{"cash": "cash", "margin_long": "margin-long", "margin_short": "margin-short"}[wantTypes[i]], details["lot"])
		require.Equal(t, "匿名銘柄", details["stock_name"])
		if i == 4 || i == 5 {
			require.Equal(t, "broker_reported", details["realized_pnl_source"])
			require.Equal(t, []float64{5000, 2000}[i-4], details["broker_reported_realized_pnl"])
		} else {
			require.NotContains(t, details, "broker_reported_realized_pnl")
		}
		if i >= 3 {
			require.Equal(t, 1000.0, details["broker_reported_close_basis"])
		} else {
			require.NotContains(t, details, "broker_reported_close_basis")
		}
	}
	trades, err := q.ListTrades(ctx, store.ListTradesParams{UserID: user.ID})
	require.NoError(t, err)
	require.Len(t, trades, 3)
	for _, trade := range trades {
		require.Equal(t, "closed", trade.Status)
	}

	conversion := importer.ParseSBIRows([]map[string]string{
		{"約定日": "2026/09/01", "銘柄コード": "7203", "銘柄": "匿名B", "取引": "信用新規買", "約定数量": "100", "約定単価": "1000"},
		{"約定日": "2026/09/03", "銘柄コード": "7203", "銘柄": "匿名B", "取引": "現引", "約定数量": "100", "約定単価": "1090", "手数料/諸経費等": "500"},
	}, nil, "")
	require.Empty(t, conversion.Errors)
	_, err = importer.Commit(ctx, q, user.ID, account.ID, sql.NullString{}, conversion)
	require.NoError(t, err)
	fills, err = q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: user.ID, AccountID: account.ID})
	require.NoError(t, err)
	var conversionIDs []string
	for _, fill := range fills {
		if fill.Symbol != "7203" || !fill.Details.Valid {
			continue
		}
		var details map[string]any
		require.NoError(t, json.Unmarshal([]byte(fill.Details.String), &details))
		if details["event_type"] == "position_conversion" {
			conversionIDs = append(conversionIDs, details["conversion_id"].(string))
		}
	}
	require.Len(t, conversionIDs, 2)
	require.Equal(t, conversionIDs[0], conversionIDs[1])
	trades, err = q.ListTrades(ctx, store.ListTradesParams{UserID: user.ID})
	require.NoError(t, err)
	closedConversion := false
	for _, trade := range trades {
		if trade.Symbol == "7203" && trade.Status == "closed" {
			closedConversion = true
			require.True(t, trade.NetPnl.Valid)
			require.Zero(t, trade.NetPnl.Float64)
		}
	}
	require.True(t, closedConversion)
}
