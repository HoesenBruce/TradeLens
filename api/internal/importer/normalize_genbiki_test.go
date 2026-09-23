package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/positions"
	"github.com/tradermemos/api/internal/store"
)

func TestNormalizeLegacyGenbiki(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.NewForDriver(conn, "sqlite")
	ctx := context.Background()
	u, err := q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "genbiki@test", PasswordHash: "x"})
	require.NoError(t, err)
	a, err := q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: u.ID, Name: "SBI", Broker: "sbi", AccountType: "cash", BaseCurrency: "JPY"})
	require.NoError(t, err)
	row := func(date, side, qty, price, fees string) map[string]string {
		return map[string]string{"約定日": date, "銘柄コード": "1515", "取引": side, "約定数量": qty, "約定単価": price, "手数料/諸経費等": fees}
	}
	parsed := ParseSBIRows([]map[string]string{
		row("2026/08/01", "現物買", "100", "900", "0"),
		row("2026/08/02", "信用新規買", "100", "1000", "200"),
		row("2026/08/03", "信用新規買", "100", "1200", "200"),
		row("2026/08/04", "現引", "100", "1090", "500"),
		row("2026/08/05", "現物売", "200", "1300", "0"),
	}, nil, "")
	require.Empty(t, parsed.Errors)
	current := parsed
	current.Executions = append([]ParsedExecution(nil), parsed.Executions...)
	for i := range parsed.Executions {
		parsed.Executions[i].EventType = ""
		parsed.Executions[i].ConversionType = ""
		parsed.Executions[i].ConversionID = ""
	}
	_, err = Commit(ctx, q, u.ID, a.ID, sql.NullString{}, parsed)
	require.NoError(t, err)
	before, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: a.ID})
	require.NoError(t, err)
	require.True(t, legacyGenbikiPair(before[3], before[4]))
	ordinary := before[4]
	ordinary.ExecutedAt = ordinary.ExecutedAt.Add(-1) // not the parser signature
	require.False(t, legacyGenbikiPair(before[3], ordinary))
	require.NoError(t, NormalizeGenbiki(ctx, q))
	after, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: a.ID})
	require.NoError(t, err)
	for i := range after {
		require.Equal(t, before[i].ID, after[i].ID)
		require.Equal(t, before[i].DedupHash, after[i].DedupHash)
	}
	var m, c map[string]any
	require.NoError(t, json.Unmarshal([]byte(after[3].Details.String), &m))
	require.NoError(t, json.Unmarshal([]byte(after[4].Details.String), &c))
	require.Equal(t, "genbiki", m["conversion_type"])
	require.Equal(t, m["conversion_id"], c["conversion_id"])
	require.False(t, legacyGenbikiPair(after[3], after[4]))
	require.Equal(t, 110700.0, positions.ConversionBasis(after)[after[4].ID])
	trs, err := q.ListTrades(ctx, store.ListTradesParams{UserID: u.ID})
	require.NoError(t, err)
	for _, tr := range trs {
		if tr.Status == "closed" {
			require.Equal(t, 59200.0, tr.NetPnl.Float64)
			require.Equal(t, 1003.5, tr.AvgEntryPrice)
		} else {
			require.Zero(t, tr.NetPnl.Float64)
		}
	}
	require.NoError(t, NormalizeGenbiki(ctx, q))
	again, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: a.ID})
	require.NoError(t, err)
	require.Equal(t, after, again)
	result, err := Commit(ctx, q, u.ID, a.ID, sql.NullString{}, current)
	require.NoError(t, err)
	require.Zero(t, result.Inserted)
	require.Equal(t, len(after), result.Skipped)
}

func TestGenbikiTransferredBasisEndToEnd(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		cash, second, converted   string
		basis, average, remaining float64
	}{
		{"full", "0", "0", "100", 100500, 1005, 0},
		{"existing cash", "100", "0", "100", 100500, 952.5, 0},
		{"multiple opening fills", "0", "100", "200", 220500, 1102.5, 0},
		{"partial", "0", "100", "100", 110500, 1105, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			require.NoError(t, err)
			defer conn.Close()
			require.NoError(t, db.Migrate(conn))
			q := store.NewForDriver(conn, "sqlite")
			ctx := context.Background()
			_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "basis@test", PasswordHash: "x"})
			require.NoError(t, err)
			_, err = q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: "u", Name: "SBI", Broker: "sbi", AccountType: "cash", BaseCurrency: "JPY"})
			require.NoError(t, err)
			row := func(date, side, qty, price, fees string) map[string]string {
				return map[string]string{"約定日": date, "銘柄コード": "1515", "取引": side, "約定数量": qty, "約定単価": price, "手数料/諸経費等": fees}
			}
			rows := []map[string]string{}
			if tc.cash != "0" {
				rows = append(rows, row("2026/08/01", "現物買", tc.cash, "900", "0"))
			}
			rows = append(rows, row("2026/08/02", "信用新規買", "100", "1000", "0"))
			if tc.second != "0" {
				rows = append(rows, row("2026/08/03", "信用新規買", tc.second, "1200", "0"))
			}
			rows = append(rows, row("2026/08/04", "現引", tc.converted, "3159", "500"))
			parsed := ParseSBIRows(rows, nil, "")
			require.Empty(t, parsed.Errors)
			_, err = Commit(ctx, q, "u", "a", sql.NullString{}, parsed)
			require.NoError(t, err)
			fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
			require.NoError(t, err)
			cash := fills[len(fills)-1]
			require.Equal(t, 3159.0, cash.Price)
			require.Equal(t, tc.basis, positions.ConversionBasis(fills)[cash.ID])
			trs, err := q.ListTrades(ctx, store.ListTradesParams{UserID: "u"})
			require.NoError(t, err)
			for _, tr := range trs {
				if tr.ClosedAt.Valid || tr.QtyRemaining == tc.remaining && tc.remaining > 0 {
					require.Zero(t, tr.NetPnl.Float64)
				} else {
					require.Equal(t, tc.average, tr.AvgEntryPrice)
				}
			}
			before := fills
			require.NoError(t, NormalizeGenbiki(ctx, q))
			fills, err = q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
			require.NoError(t, err)
			require.Equal(t, before, fills)
		})
	}
}

func TestLegacyGenbikiDoesNotInferOrdinaryTrades(t *testing.T) {
	margin := store.Execution{ID: "m", UserID: "u", AccountID: "a", Symbol: "1515", InstrumentType: "stock", Side: "sell", Quantity: 100, Price: 1090, Multiplier: 1, Details: sql.NullString{String: `{"lot":"sbi:margin-long","position_type":"margin_long","position_effect":"reduce"}`, Valid: true}}
	margin.ExecutedAt, _ = time.Parse(time.RFC3339Nano, "2026-08-03T15:00:00.000004Z")
	cash := margin
	cash.ID = "c"
	cash.Side = "buy"
	cash.Details.String = `{"lot":"sbi:cash","position_type":"cash","position_effect":"increase"}`
	cash.ExecutedAt = margin.ExecutedAt.Add(2 * time.Microsecond)
	require.False(t, legacyGenbikiPair(margin, cash), "ordinary rows occupy even microseconds")
	cash.ExecutedAt = margin.ExecutedAt.Add(time.Microsecond)
	require.True(t, legacyGenbikiPair(margin, cash))
	for _, mutate := range []func(*store.Execution){
		func(e *store.Execution) { e.AccountID = "other" },
		func(e *store.Execution) { e.ImportBatchID = sql.NullString{String: "other", Valid: true} },
		func(e *store.Execution) { e.Symbol = "other" },
		func(e *store.Execution) { e.Quantity = 200 },
		func(e *store.Execution) { e.Fees = 1 },
		func(e *store.Execution) {
			e.Details.String = `{"lot":"ibkr:cash","position_type":"cash","position_effect":"increase"}`
		},
	} {
		changed := cash
		mutate(&changed)
		require.False(t, legacyGenbikiPair(margin, changed))
	}
}
