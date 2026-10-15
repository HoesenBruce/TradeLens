package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tradermemos/api/internal/marketdata"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
	"uuid"
)

const replayMapping = `{"symbol":"symbol","side":"side","quantity":"quantity","price":"price","executed_at":"executed_at","fees":"fees"}`
const replayCSV = "symbol,side,quantity,price,executed_at,fees,unused\nAAPL,buy,10,100,2026-01-05T10:00:00Z,2,keep-this\nAAPL,sell,10,110,2026-01-05T11:00:00Z,3,keep-that\n"

type replayReport struct {
	RetainedRows, Executions, Changed, Trades int
	Fingerprint                               string
	Applied                                   bool
}

func decodeReplay(t *testing.T, rec *httptest.ResponseRecorder) replayReport {
	t.Helper()
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var wire struct {
		RetainedRows int    `json:"retained_rows"`
		Executions   int    `json:"executions"`
		Changed      int    `json:"changed"`
		Trades       int    `json:"trades"`
		Fingerprint  string `json:"fingerprint"`
		Applied      bool   `json:"applied"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wire))
	return replayReport{wire.RetainedRows, wire.Executions, wire.Changed, wire.Trades, wire.Fingerprint, wire.Applied}
}
func commitReplayCSV(t *testing.T, s *api.Server, tok, acc, csv string, extra map[string]string) []struct {
	ID            string
	RetainedRows  int  `json:"retained_rows"`
	Reprocessable bool `json:"reprocessable"`
	Filename      string
} {
	t.Helper()
	fields := map[string]string{"account_id": acc, "column_mapping": replayMapping}
	for k, v := range extra {
		fields[k] = v
	}
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "replay.csv", csv, fields))
	require.Equal(t, 200, rec.Code, rec.Body.String())
	rec = do(s, "GET", "/api/v1/imports", "", tok)
	require.Equal(t, 200, rec.Code)
	var batches []struct {
		ID            string
		RetainedRows  int  `json:"retained_rows"`
		Reprocessable bool `json:"reprocessable"`
		Filename      string
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &batches))
	return batches
}
func previewReplay(t *testing.T, s *api.Server, tok, batch string) replayReport {
	t.Helper()
	return decodeReplay(t, do(s, "POST", "/api/v1/imports/"+batch+"/reprocess", `{}`, tok))
}
func applyReplay(t *testing.T, s *api.Server, tok, batch, fp string) *httptest.ResponseRecorder {
	t.Helper()
	return do(s, "POST", "/api/v1/imports/"+batch+"/reprocess", `{"apply":true,"fingerprint":"`+fp+`"}`, tok)
}

// Simulate data written by the previous parser, without classifying it as a manual edit.
func previousParserFees(t *testing.T, f *isolationFixture) {
	t.Helper()
	_, err := f.conn.Exec("UPDATE executions SET fees=0")
	require.NoError(t, err)
	rows, err := f.conn.Query("SELECT source_id,execution_id,snapshot FROM import_source_fills")
	require.NoError(t, err)
	type entry struct{ source, id, data string }
	var entries []entry
	for rows.Next() {
		var e entry
		require.NoError(t, rows.Scan(&e.source, &e.id, &e.data))
		entries = append(entries, e)
	}
	require.NoError(t, rows.Close())
	for _, e := range entries {
		var snapshot store.Execution
		require.NoError(t, json.Unmarshal([]byte(e.data), &snapshot))
		snapshot.Fees = 0
		data, err := json.Marshal(snapshot)
		require.NoError(t, err)
		_, err = f.conn.Exec("UPDATE import_source_fills SET snapshot=? WHERE source_id=? AND execution_id=?", string(data), e.source, e.id)
		require.NoError(t, err)
	}
}

func TestReprocessRetainsDeduplicatedSourcesAndAssociations(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "replay@x.test")
	acc := accountID(t, f.s, tok)
	batches := commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)
	require.Len(t, batches, 1)
	batch := batches[0].ID
	require.Equal(t, 2, batches[0].RetainedRows)
	require.True(t, batches[0].Reprocessable)
	for i := 0; i < 2; i++ {
		commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)
	}
	var count int
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM import_source_rows").Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM import_source_contexts").Scan(&count))
	require.Equal(t, 1, count)
	var payload string
	require.NoError(t, f.conn.QueryRow("SELECT payload FROM import_source_rows ORDER BY row_number LIMIT 1").Scan(&payload))
	require.Contains(t, payload, "keep-this")
	q := store.NewForDriver(f.conn, "sqlite")
	ctx := context.Background()
	executions, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: executionsUser(t, f.conn), AccountID: acc})
	require.NoError(t, err)
	require.Len(t, executions, 2)
	uid := executions[0].UserID
	list, err := q.ListTrades(ctx, store.ListTradesParams{UserID: uid, AccountID: acc})
	require.NoError(t, err)
	require.Len(t, list, 1)
	tradeID := list[0].ID
	setup, err := q.CreateSetup(ctx, store.CreateSetupParams{ID: "replay-setup", UserID: uid, Name: "Replay setup"})
	require.NoError(t, err)
	tag, err := q.CreateTag(ctx, store.CreateTagParams{ID: "replay-tag", UserID: uid, Name: "Replay tag", Kind: "custom"})
	require.NoError(t, err)
	require.NoError(t, q.SetTradeSetup(ctx, store.SetTradeSetupParams{TradeID: tradeID, SetupID: setup.ID}))
	require.NoError(t, q.SetTradeTags(ctx, store.SetTradeTagsParams{TradeID: tradeID, TagID: tag.ID}))
	require.NoError(t, q.UpsertTradeJournal(ctx, store.UpsertTradeJournalParams{TradeID: tradeID, UserID: uid, Notes: "Keep my review", SetupID: sql.NullString{String: setup.ID, Valid: true}, InitialRisk: sql.NullFloat64{Float64: 50, Valid: true}, Confidence: sql.NullInt64{Int64: 4, Valid: true}}))
	_, err = q.InsertAttachment(ctx, store.InsertAttachmentParams{ID: "replay-attachment", UserID: uid, TradeID: tradeID, Filename: "chart.png", ContentType: "image/png", StorageKey: "chart.png"})
	require.NoError(t, err)
	_, err = q.CreateCoachReview(ctx, store.CreateCoachReviewParams{ID: "replay-review", UserID: uid, TradeID: tradeID, Model: "historical", Notes: "[]", NextAction: "Review"})
	require.NoError(t, err)
	// The stored source stays complete even when an older parser omitted fees.
	previousParserFees(t, f)
	require.NoError(t, trades.NewService(q).Regroup(ctx, uid, acc))
	before := f.databaseState(t)
	preview := previewReplay(t, f.s, tok, batch)
	require.Equal(t, 2, preview.Changed)
	require.False(t, preview.Applied)
	require.Equal(t, before, f.databaseState(t), "preview must roll back every table")
	applied := decodeReplay(t, applyReplay(t, f.s, tok, batch, preview.Fingerprint))
	require.True(t, applied.Applied)
	detail := do(f.s, "GET", "/api/v1/trades/"+tradeID, "", tok)
	require.Equal(t, 200, detail.Code)
	require.Contains(t, detail.Body.String(), "Keep my review")
	after, err := q.GetTrade(ctx, store.GetTradeParams{ID: tradeID, UserID: uid})
	require.NoError(t, err)
	require.Equal(t, 95.0, after.NetPnl.Float64)
	for _, table := range []string{"trade_journal", "trade_tags", "trade_setups", "trade_attachments", "coach_reviews"} {
		require.Equal(t, before[table], f.databaseState(t)[table], table)
	}
	updated, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: uid, AccountID: acc})
	require.NoError(t, err)
	for i, e := range updated {
		require.Equal(t, executions[i].ID, e.ID)
	}
	again := previewReplay(t, f.s, tok, batch)
	require.Zero(t, again.Changed)
	decodeReplay(t, applyReplay(t, f.s, tok, batch, again.Fingerprint))
	commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM executions").Scan(&count))
	require.Equal(t, 2, count)
}
func executionsUser(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var uid string
	require.NoError(t, conn.QueryRow("SELECT id FROM users LIMIT 1").Scan(&uid))
	return uid
}

func TestReprocessSourceRevisionAndOverlap(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "revision@x.test")
	acc := accountID(t, f.s, tok)
	batches := commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)
	oldID := batches[0].ID
	commitReplayCSV(t, f.s, tok, acc, strings.ReplaceAll(replayCSV, ",3,keep-that", ",13,keep-that"), nil)
	list := do(f.s, "GET", "/api/v1/imports", "", tok)
	var current []struct {
		ID            string
		RetainedRows  int  `json:"retained_rows"`
		Reprocessable bool `json:"reprocessable"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &current))
	var newID string
	for _, b := range current {
		if b.ID == oldID {
			require.True(t, b.Reprocessable)
		} else {
			require.Equal(t, 1, b.RetainedRows)
			require.True(t, b.Reprocessable)
			newID = b.ID
		}
	}
	require.NotEmpty(t, newID)
	p := previewReplay(t, f.s, tok, newID)
	require.Equal(t, 1, p.Changed)
	decodeReplay(t, applyReplay(t, f.s, tok, newID, p.Fingerprint))
	var count int
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM executions").Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM import_source_rows").Scan(&count))
	require.Equal(t, 3, count)
	oldPreview := previewReplay(t, f.s, tok, oldID)
	require.Equal(t, 1, oldPreview.RetainedRows)
	require.Zero(t, oldPreview.Changed)
}

func TestReprocessRejectsConflictsAndForeignUsers(t *testing.T) {
	for _, mode := range []string{"manual-edit", "missing-leg", "grouping", "stale-preview", "changed-calculation", "parser-error", "foreign-user", "reversed"} {
		t.Run(mode, func(t *testing.T) {
			f := newIsolationFixture(t)
			tok := registerAndLogin(t, f.s, mode+"@x.test")
			acc := accountID(t, f.s, tok)
			batch := commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)[0].ID
			p := previewReplay(t, f.s, tok, batch)
			status := 409
			switch mode {
			case "manual-edit":
				_, err := f.conn.Exec("UPDATE executions SET fees=99 WHERE side='buy'")
				require.NoError(t, err)
			case "missing-leg":
				_, err := f.conn.Exec("DELETE FROM executions WHERE side='buy'")
				require.NoError(t, err)
			case "grouping":
				// Current rules now interpret the source as a second buy: reject changed membership.
				rows, err := f.conn.Query("SELECT id,payload FROM import_source_rows WHERE row_number=2")
				require.NoError(t, err)
				require.True(t, rows.Next())
				var id, data string
				require.NoError(t, rows.Scan(&id, &data))
				require.NoError(t, rows.Close())
				data = strings.Replace(data, `"sell"`, `"buy"`, 1)
				_, err = f.conn.Exec("UPDATE import_source_rows SET payload=? WHERE id=?", data, id)
				require.NoError(t, err)
			case "parser-error":
				previousParserFees(t, f) // first row would change before the second row fails
				_, err := f.conn.Exec(`UPDATE import_source_rows SET payload='{"values":{"symbol":"AAPL"}}' WHERE row_number=2`)
				require.NoError(t, err)
			case "stale-preview":
				_, err := f.conn.Exec(`UPDATE import_source_contexts SET payload=replace(payload,'"source_tz":"UTC"','"source_tz":"Asia/Tokyo"')`)
				require.NoError(t, err)
			case "changed-calculation":
				_, err := f.conn.Exec("UPDATE trades SET net_pnl=999")
				require.NoError(t, err)
			case "foreign-user":
				tok = registerAndLogin(t, f.s, "other@x.test")
				status = 404
			case "reversed":
				_, err := f.conn.Exec("UPDATE import_batches SET status='reversed' WHERE id=?", batch)
				require.NoError(t, err)
			}
			before := f.databaseState(t)
			rec := applyReplay(t, f.s, tok, batch, p.Fingerprint)
			require.Equal(t, status, rec.Code, rec.Body.String())
			if mode == "stale-preview" || mode == "changed-calculation" {
				require.Contains(t, rec.Body.String(), "since preview")
			}
			require.Equal(t, before, f.databaseState(t))
		})
	}
}

func TestReprocessIdenticalFillsAndCSVFormats(t *testing.T) {
	for _, name := range []string{"identical", "day-first", "sbi", "journal", "round-trip", "genbiki", "journal-option"} {
		t.Run(name, func(t *testing.T) {
			f := newIsolationFixture(t)
			tok := registerAndLogin(t, f.s, name+"@x.test")
			acc := accountID(t, f.s, tok)
			csv := replayCSV
			extra := map[string]string{}
			switch name {
			case "identical":
				csv = "symbol,side,quantity,price,executed_at,fees\nAAPL,buy,1,10,2026-01-05T10:00:00Z,0\nAAPL,buy,1,10,2026-01-05T10:00:00Z,0\n"
			case "day-first":
				csv = strings.ReplaceAll(strings.ReplaceAll(replayCSV, "2026-01-05T10:00:00Z", "05/01/2026 10:00"), "2026-01-05T11:00:00Z", "13/01/2026 11:00")
				extra["source_tz"] = "America/New_York"
			case "sbi":
				data, err := os.ReadFile("../importer/testdata/sbi-review-direction.csv")
				require.NoError(t, err)
				csv = string(data)
				extra["column_mapping"] = `{"symbol":"銘柄コード"}`
			case "journal":
				csv = "Symbol,Open Date,Date,Side,Qty,Entry,Exit,Notes\nAAPL,2026-01-05,2026-01-06,LONG,10,100,110,source-note\n"
				extra["column_mapping"] = ""
			case "journal-option":
				csv = "Symbol,Market,Side,Qty,Entry,Exit,Open Date,Date\nTSLA,OPTION,LONG,1,1.8,2,2026-07-10T15:18:34Z,2026-07-10T15:40:17Z\n"
				extra["column_mapping"] = ""
				extra["journal_option_overrides"] = `{"1":"call"}`
			case "round-trip":
				csv = "symbol,side,quantity,open_time,open_price,close_time,close_price\nAAPL,buy,1,2026-01-05T10:00:00Z,100,2026-01-05T11:00:00Z,110\n"
				extra["column_mapping"] = `{"symbol":"symbol","side":"side","quantity":"quantity","open_time":"open_time","open_price":"open_price","close_time":"close_time","close_price":"close_price"}`
			case "genbiki":
				csv = "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/08/01,1515,信用新規買,100,1000,0\n2026/08/04,1515,現引,100,1090,500\n"
				extra["column_mapping"] = `{"symbol":"銘柄コード"}`
			}
			batches := commitReplayCSV(t, f.s, tok, acc, csv, extra)
			require.True(t, batches[0].Reprocessable)
			batch := batches[0].ID
			before := f.databaseState(t)
			p := previewReplay(t, f.s, tok, batch)
			require.Zero(t, p.Changed)
			require.Equal(t, before, f.databaseState(t))
			decodeReplay(t, applyReplay(t, f.s, tok, batch, p.Fingerprint))
			p = previewReplay(t, f.s, tok, batch)
			require.Zero(t, p.Changed)
		})
	}
}

func TestReprocessUnavailableForLegacyAndUnsupported(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "legacy@x.test")
	acc := accountID(t, f.s, tok)
	batches := commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)
	batch := batches[0].ID
	_, err := f.conn.Exec("DELETE FROM import_source_rows")
	require.NoError(t, err)
	before := f.databaseState(t)
	rec := do(f.s, "POST", "/api/v1/imports/"+batch+"/reprocess", `{}`, tok)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, before, f.databaseState(t))
	batches = commitReplayCSV(t, f.s, tok, acc, sbiCashCSV, nil)
	for _, b := range batches {
		require.False(t, b.Reprocessable)
	}
}

func TestReprocessPostgres(t *testing.T) {
	databaseURL := os.Getenv("TM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TM_TEST_DATABASE_URL for PostgreSQL replay acceptance")
	}
	conn, err := db.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, db.Migrate(conn, db.DriverPostgres))
	q := store.NewForDriver(conn, db.DriverPostgres)
	jwt := auth.NewJWT("local-pg-replay")
	s := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q)})
	tok := registerAndLogin(t, s, "replay-"+uuid.New().String()+"@pg.test")
	acc := accountID(t, s, tok)
	batch := commitReplayCSV(t, s, tok, acc, replayCSV, nil)[0].ID
	p := previewReplay(t, s, tok, batch)
	require.Equal(t, 2, p.RetainedRows)
	require.Zero(t, p.Changed)
	decodeReplay(t, applyReplay(t, s, tok, batch, p.Fingerprint))
	commitReplayCSV(t, s, tok, acc, strings.ReplaceAll(replayCSV, ",3,keep-that", ",13,keep-that"), nil)
	rec := do(s, "GET", "/api/v1/imports", "", tok)
	var list []struct {
		ID            string
		Reprocessable bool `json:"reprocessable"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	var revised string
	for _, b := range list {
		if b.ID != batch && b.Reprocessable {
			revised = b.ID
		}
	}
	require.NotEmpty(t, revised)
	p = previewReplay(t, s, tok, revised)
	require.Equal(t, 1, p.Changed)
	decodeReplay(t, applyReplay(t, s, tok, revised, p.Fingerprint))
	p = previewReplay(t, s, tok, revised)
	require.Zero(t, p.Changed)
	accountID(t, s, tok) // the product requires keeping at least one account
	rec = do(s, "DELETE", "/api/v1/accounts/"+acc, "", tok)
	require.Equal(t, 204, rec.Code, rec.Body.String())
}

func TestReprocessPreviewIncludesAccountingOnlyChanges(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "accounting-replay@x.test")
	acc := accountID(t, f.s, tok)
	batch := commitReplayCSV(t, f.s, tok, acc, replayCSV, nil)[0].ID
	_, err := f.conn.Exec("UPDATE trades SET net_pnl=999")
	require.NoError(t, err)
	before := f.databaseState(t)
	rec := do(f.s, "POST", "/api/v1/imports/"+batch+"/reprocess", `{}`, tok)
	p := decodeReplay(t, rec)
	require.Zero(t, p.Changed)
	var wire struct {
		TradesChanged int `json:"trades_changed"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wire))
	require.Equal(t, 1, wire.TradesChanged)
	require.Equal(t, before, f.databaseState(t))
	decodeReplay(t, applyReplay(t, f.s, tok, batch, p.Fingerprint))
	var pnl float64
	require.NoError(t, f.conn.QueryRow("SELECT net_pnl FROM trades").Scan(&pnl))
	require.Equal(t, 95.0, pnl)
}

func TestReprocessArchiveBulkChunkingAndDedup(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "bulk-replay@x.test")
	acc := accountID(t, f.s, tok)
	var csv strings.Builder
	csv.WriteString("symbol,side,quantity,price,executed_at,fees\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&csv, "AAPL,buy,1,%d,2026-01-05T10:00:00Z,0\n", i+1)
	}
	batch := commitReplayCSV(t, f.s, tok, acc, csv.String(), nil)[0]
	require.Equal(t, 1000, batch.RetainedRows)
	commitReplayCSV(t, f.s, tok, acc, csv.String(), nil)
	var n int
	for _, table := range []string{"executions", "import_source_rows", "import_source_fills"} {
		require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
		require.Equal(t, 1000, n, table)
	}
	require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM import_source_contexts").Scan(&n))
	require.Equal(t, 1, n)
}

func TestReprocessArchiveFailureRollsBackImport(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "archive-failure@x.test")
	acc := accountID(t, f.s, tok)
	_, err := f.conn.Exec("CREATE TRIGGER fail_source BEFORE INSERT ON import_source_rows BEGIN SELECT RAISE(ABORT,'test archive failure'); END;")
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "replay.csv", replayCSV, map[string]string{"account_id": acc, "column_mapping": replayMapping}))
	require.Equal(t, 500, rec.Code)
	for _, table := range []string{"executions", "trades", "import_source_contexts", "import_source_rows", "import_source_fills"} {
		var n int
		require.NoError(t, f.conn.QueryRow("SELECT count(*) FROM "+table).Scan(&n))
		require.Zero(t, n, table)
	}
}

func TestReprocessDoesNotExposeProviderErrors(t *testing.T) {
	f := newIsolationFixture(t)
	tok := registerAndLogin(t, f.s, "provider-replay@x.test")
	acc := accountID(t, f.s, tok)
	csv := strings.Replace(replayCSV, "2026-01-05T11:00:00Z", "2026-01-06T11:00:00Z", 1)
	batch := commitReplayCSV(t, f.s, tok, acc, csv, nil)[0].ID
	q := store.NewForDriver(f.conn, "sqlite")
	jwt := auth.NewJWT("disposable-isolation-test-secret")
	failed := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q, func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{}, errors.New("SYNTHETIC-private-provider-detail")
	})})
	before := f.databaseState(t)
	rec := do(failed, "POST", "/api/v1/imports/"+batch+"/reprocess", `{}`, tok)
	require.Equal(t, 500, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "SYNTHETIC-private-provider-detail")
	require.Equal(t, before, f.databaseState(t))
}
