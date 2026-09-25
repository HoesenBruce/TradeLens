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
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func TestSBIMarginPnLEnrichment(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "sbi.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := context.Background()
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "sbi-pnl@test", PasswordHash: "x"})
	require.NoError(t, err)
	_, err = q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: "u", Name: "SBI", Broker: "sbi", BaseCurrency: "JPY"})
	require.NoError(t, err)
	trade := ParseSBIRows([]map[string]string{
		{"約定日": "2026/09/15", "銘柄コード": "7003", "取引": "信用新規買", "約定数量": "100", "約定単価": "6000"},
		{"約定日": "2026/09/16", "銘柄コード": "7003", "取引": "信用新規買", "約定数量": "100", "約定単価": "5000"},
		{"約定日": "2026/09/17", "銘柄コード": "7003", "取引": "信用返済売", "約定数量": "100", "約定単価": "6200", "受渡金額/決済損益": "20000"},
	}, nil, "")
	_, err = Commit(ctx, q, "u", "a", sql.NullString{}, trade)
	require.NoError(t, err)
	input := []byte("信用実現損益\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2026/09/17,7003,信用返済売,100,6200,6000,20000\n")
	cp932, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), input)
	require.NoError(t, err)
	file, ok, err := ReadSBIMarginPnLCSV(cp932)
	require.NoError(t, err)
	require.True(t, ok)
	fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
	require.NoError(t, err)
	matched := MatchSBIMarginPnL(file.Parsed, fills)
	require.Equal(t, "enrichable", matched[0].Status)
	disagree := append([]SBIMarginPnLRow(nil), file.Parsed...)
	disagree = append(disagree, file.Parsed[0])
	disagree[1].Basis = 5999
	require.Equal(t, "conflict", MatchSBIMarginPnL(disagree, fills)[0].Status)
	require.Equal(t, "conflict", MatchSBIMarginPnL(disagree, fills)[1].Status)
	n, err := CommitSBIMarginPnL(ctx, q, "u", "a", matched)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	fills, err = q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
	require.NoError(t, err)
	require.Len(t, fills, 3)
	require.Equal(t, "already_enriched", MatchSBIMarginPnL(file.Parsed, fills)[0].Status)
	var details map[string]any
	require.NoError(t, json.Unmarshal([]byte(fills[2].Details.String), &details))
	require.Equal(t, 6000.0, details["broker_reported_close_basis"])
	require.Equal(t, 20000.0, details["broker_reported_realized_pnl"])
	snapshot := positions.Replay(fills, []time.Time{time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)})[0]
	require.Equal(t, 5000.0, snapshot.Accounts[0].Positions[0].AverageCost)

	conflict := append([]SBIMarginPnLRow(nil), file.Parsed...)
	conflict[0].Basis = 5999
	require.Equal(t, "conflict", MatchSBIMarginPnL(conflict, fills)[0].Status)
	wrongPnL := 20001.0
	conflict[0].Basis, conflict[0].PnL = 6000, &wrongPnL
	require.Equal(t, "conflict", MatchSBIMarginPnL(conflict, fills)[0].Status)
	short := fills[2]
	short.ID, short.Side = "short-close", "buy"
	short.Details.String = `{"lot":"sbi:margin-short","position_effect":"reduce"}`
	shortRow := file.Parsed[0]
	shortRow.Transaction = "信用返済買"
	require.Equal(t, "enrichable", MatchSBIMarginPnL([]SBIMarginPnLRow{shortRow}, []store.Execution{short})[0].Status)
	fills = append(fills, fills[2])
	fills[3].ID = "second-close"
	require.Equal(t, "ambiguous_match", MatchSBIMarginPnL(file.Parsed, fills)[0].Status)
	file.Parsed[0].Price = 6199
	require.Equal(t, "no_matching_execution", MatchSBIMarginPnL(file.Parsed, fills)[0].Status)
}
