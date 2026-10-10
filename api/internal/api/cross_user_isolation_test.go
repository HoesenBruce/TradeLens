package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/accountvalue"
	"github.com/tradermemos/api/internal/alerts"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/backup"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/flexsync"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/storage"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

type isolationFixture struct {
	s     *api.Server
	conn  *sql.DB
	files string
}
type isolationUser struct {
	session, id, pat, patID                                                                                                 string
	accounts                                                                                                                []string
	trade, execution, cash, note, setup, tag, news, asset, prediction, attachment, media, channel, share, shareToken, batch string
	marker, symbol                                                                                                          string
}

func newIsolationFixture(t *testing.T) *isolationFixture {
	t.Helper()
	root := t.TempDir()
	conn, err := db.Open(filepath.Join(root, "isolation.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, db.Migrate(conn))
	q := store.NewForDriver(conn, "sqlite")
	jwt := auth.NewJWT("disposable-isolation-test-secret")
	provider := marketdata.NewYahooProvider()
	// Fixed FX transport: the isolation suite never reaches a live provider.
	provider.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
		quote := "150"
		if strings.Contains(r.URL.Path, "JPYUSD") {
			quote = "0.006666666666666667"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[{"timestamp":[1780000000],"indicators":{"quote":[{"close":[` + quote + `]}]}}]}}`))}, nil
	})}
	f := &isolationFixture{conn: conn, files: filepath.Join(root, "files")}
	f.s = api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q),
		Storage: storage.NewLocalDisk(f.files), AttachMaxBytes: 1 << 20, ImportMaxBytes: 1 << 20,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Market: marketdata.NewService(q, provider),
		AccountValue: accountvalue.NewService(accountValueBars(map[string]map[string]float64{
			"1306": {"2026-01-05": 100, "2026-01-06": 100}, "ISOA": {"2026-01-05": 110, "2026-01-06": 110}, "ISOB": {"2026-01-05": 987654, "2026-01-06": 987654},
		})), ShareLinksEnabled: true, Backup: backup.New(conn, backup.Config{Driver: "sqlite", Dir: filepath.Join(root, "backups"), Keep: 2}, nil),
	})
	return f
}
func isolationJSON(t *testing.T, body any) string {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	return string(b)
}
func isolationObject(t *testing.T, s *api.Server, method, path, body, token string, status int) map[string]any {
	t.Helper()
	rec := do(s, method, "/api/v1"+path, body, token)
	// Creation responses may contain token secrets: never print their bodies.
	require.Equal(t, status, rec.Code, "%s %s", method, path)
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}
func (f *isolationFixture) seed(t *testing.T, name, symbol string, amount float64) isolationUser {
	t.Helper()
	u := isolationUser{session: registerAndLogin(t, f.s, name+"@isolation.invalid"), marker: "PRIVATE-" + name, symbol: symbol}
	u.id = isolationObject(t, f.s, "GET", "/me", "", u.session, 200)["id"].(string)
	for i := 0; i < 2; i++ {
		obj := isolationObject(t, f.s, "POST", "/accounts", isolationJSON(t, map[string]any{"name": fmt.Sprintf("%s-%d", u.marker, i), "base_currency": "JPY", "user_id": "forged"}), u.session, 201)
		require.Equal(t, u.id, obj["user_id"])
		u.accounts = append(u.accounts, obj["id"].(string))
		seedShareClosedTrade(t, f.s, u.session, u.accounts[i], symbol, "2026-01-05", 100, 100+amount)
	}
	ex := isolationObject(t, f.s, "POST", "/executions", isolationJSON(t, map[string]any{"account_id": u.accounts[0], "symbol": symbol, "side": "buy", "quantity": 2, "price": 100, "executed_at": "2026-01-06T01:00:00Z"}), u.session, 201)
	u.execution = ex["execution_id"].(string)
	rec := do(f.s, "GET", "/api/v1/trades?account_id="+u.accounts[0]+"&status=closed", "", u.session)
	require.Equal(t, 200, rec.Code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
	u.trade = list[0]["id"].(string)
	u.cash = isolationObject(t, f.s, "POST", "/cash-transactions", isolationJSON(t, map[string]any{"account_id": u.accounts[0], "type": "deposit", "amount": amount * 1000, "currency": "JPY", "occurred_at": "2026-01-05T00:00:00Z", "note": u.marker}), u.session, 201)["id"].(string)
	u.note = isolationObject(t, f.s, "POST", "/notes", isolationJSON(t, map[string]any{"occurred_at": "2026-01-05", "title": u.marker, "body": u.marker}), u.session, 201)["id"].(string)
	u.setup = isolationObject(t, f.s, "POST", "/setups", isolationJSON(t, map[string]any{"name": u.marker, "thesis": u.marker, "symbol": symbol, "direction": "short", "target_price": 123, "stop_price": 140}), u.session, 201)["id"].(string)
	u.tag = createTag(t, f.s, u.session, u.marker)
	isolationObject(t, f.s, "PATCH", "/trades/"+u.trade, isolationJSON(t, map[string]any{"notes": u.marker, "setup_id": u.setup, "tag_ids": []string{u.tag}, "initial_risk": 10, "target_price": 123, "stop_price": 140}), u.session, 200)
	n := isolationObject(t, f.s, "POST", "/news", isolationJSON(t, map[string]any{"title": u.marker, "source": "synthetic", "published_at": "2026-01-05T00:00:00Z", "notes": u.marker, "assets": []any{map[string]any{"asset_type": "stock", "symbol": symbol, "market": "JP"}}}), u.session, 201)
	u.news = n["id"].(string)
	u.asset = n["assets"].([]any)[0].(map[string]any)["id"].(string)
	u.prediction = isolationObject(t, f.s, "POST", "/news/"+u.news+"/predictions", isolationJSON(t, map[string]any{"news_asset_id": u.asset, "direction": "bullish", "horizons": []int{1}, "reasoning": u.marker}), u.session, 201)["id"].(string)
	require.Equal(t, 200, do(f.s, "POST", "/api/v1/news/"+u.news+"/predictions/"+u.prediction+"/validate", `{}`, u.session).Code)
	for _, path := range []string{"/trades/" + u.trade + "/attachments", "/media"} {
		r := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(r, imageUploadReq(t, "/api/v1"+path, u.session, []byte("\x89PNG\r\n\x1a\n"+u.marker)))
		require.Equal(t, 201, r.Code)
		var obj map[string]any
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &obj))
		if path == "/media" {
			u.media = obj["id"].(string)
		} else {
			u.attachment = obj["id"].(string)
		}
	}
	p := isolationObject(t, f.s, "POST", "/access-tokens", isolationJSON(t, map[string]any{"name": u.marker}), u.session, 201)
	u.pat, u.patID = p["token"].(string), p["id"].(string)
	sh := isolationObject(t, f.s, "POST", "/share-links", isolationJSON(t, map[string]any{"account_id": u.accounts[0], "show_amounts": false}), u.session, 201)
	u.share, u.shareToken = sh["id"].(string), sh["token"].(string)
	u.channel = isolationObject(t, f.s, "POST", "/settings/alert-channels", isolationJSON(t, map[string]any{"kind": "webhook", "target": "https://example.invalid/" + u.marker, "label": u.marker}), u.session, 200)["id"].(string)
	isolationObject(t, f.s, "PUT", "/accounts/"+u.accounts[0]+"/flex-sync", isolationJSON(t, map[string]any{"query_id": u.marker, "token": "synthetic-unused-credential", "enabled": false}), u.session, 200)
	isolationObject(t, f.s, "PUT", "/accounts/"+u.accounts[0]+"/prop-settings", `{"profit_target":100,"max_drawdown":50}`, u.session, 200)
	r := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(r, multipartFileReq(t, "/api/v1/imports/commit", u.session, u.marker+".html", mt5StatementHTML, map[string]string{"account_id": u.accounts[1]}))
	require.Equal(t, 200, r.Code)
	r = do(f.s, "GET", "/api/v1/imports", "", u.session)
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &list))
	require.Len(t, list, 1)
	u.batch = list[0]["id"].(string)
	for _, tok := range []string{u.session, u.pat} {
		require.Equal(t, u.id, isolationObject(t, f.s, "GET", "/me", "", tok, 200)["id"])
		for _, acc := range u.accounts {
			require.Equal(t, u.id, isolationObject(t, f.s, "GET", "/accounts/"+acc, "", tok, 200)["user_id"])
		}
	}
	for _, path := range []string{"/accounts/" + u.accounts[0] + "/prop-status", "/accounts/" + u.accounts[0] + "/prop-settings", "/accounts/" + u.accounts[0] + "/flex-sync", "/news/" + u.news + "/predictions/" + u.prediction + "/validations"} {
		require.Equal(t, 200, do(f.s, "GET", "/api/v1"+path, "", u.session).Code)
	}
	summary := isolationObject(t, f.s, "GET", "/analytics/summary", "", u.session, 200)
	require.Greater(t, summary["total_trades"].(float64), float64(0))

	return u
}

// Hash complete rows so a rejected write cannot hide behind a safe response,
// and assertion output never prints credentials from the disposable database.
func (f *isolationFixture) databaseState(t *testing.T) map[string][32]byte {
	t.Helper()
	rows, err := f.conn.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	require.NoError(t, err)
	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	state := map[string][32]byte{}
	for _, name := range tables {
		// PAT authentication records usage even on denied requests.
		if name == "access_token_uses" {
			continue
		}
		r, err := f.conn.Query(`SELECT * FROM "` + strings.ReplaceAll(name, `"`, `""`) + `"`)
		require.NoError(t, err)
		cols, err := r.Columns()
		require.NoError(t, err)
		var records []string
		for r.Next() {
			values := make([]any, len(cols))
			dest := make([]any, len(cols))
			for i := range values {
				dest[i] = &values[i]
			}
			require.NoError(t, r.Scan(dest...))
			if name == "access_tokens" {
				for i, col := range cols {
					if col == "last_used_at" {
						values[i] = nil
					}
				}
			}
			records = append(records, isolationJSON(t, values))
		}
		require.NoError(t, r.Err())
		require.NoError(t, r.Close())
		sort.Strings(records)
		state[name] = sha256.Sum256([]byte(strings.Join(records, "\n")))
	}
	return state
}
func (f *isolationFixture) fileState(t *testing.T) map[string][32]byte {
	t.Helper()
	state := map[string][32]byte{}
	err := filepath.WalkDir(f.files, func(path string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		state[path] = sha256.Sum256(b)
		return nil
	})
	require.NoError(t, err)
	return state
}
func TestCrossUserIsolationIDOR(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	require.NotEqual(t, a.id, b.id)
	for _, pair := range [][2]isolationUser{{a, b}, {b, a}} {
		caller, owner := pair[0], pair[1]
		acc := "/accounts/" + owner.accounts[0]
		trade := "/trades/" + owner.trade
		news := "/news/" + owner.news
		cases := []struct {
			method, path, body string
			denied             int
		}{
			{"GET", acc, "", 404}, {"PUT", acc, `{"name":"tampered"}`, 404}, {"DELETE", acc, "", 404}, {"DELETE", acc + "/trades", "", 404},
			{"GET", acc + "/prop-settings", "", 404}, {"GET", acc + "/prop-status", "", 404}, {"PUT", acc + "/prop-settings", `{"profit_target":1}`, 404}, {"DELETE", acc + "/prop-settings", "", 404},
			{"GET", acc + "/flex-sync", "", 404}, {"PUT", acc + "/flex-sync", `{"query_id":"x","token":"fake"}`, 404}, {"DELETE", acc + "/flex-sync", "", 404}, {"POST", acc + "/flex-sync/run", "", 404},
			{"POST", "/trades/regroup", isolationJSON(t, map[string]any{"account_id": owner.accounts[0]}), 500},
			{"GET", trade, "", 404}, {"PATCH", trade, `{"notes":"tampered","initial_risk":999}`, 404}, {"DELETE", trade, "", 404},
			{"POST", trade + "/coach", `{}`, 404}, {"POST", trade + "/coach/stream", `{}`, 404}, {"GET", trade + "/coach/reviews", "", 404}, {"POST", trade + "/excursion", `{}`, 404},
			{"PATCH", "/executions/" + owner.execution, `{"side":"buy","quantity":1,"price":1,"executed_at":"2026-01-06T01:00:00Z"}`, 404}, {"DELETE", "/executions/" + owner.execution, "", 404},
			{"POST", "/executions", isolationJSON(t, map[string]any{"account_id": owner.accounts[0], "user_id": owner.id, "symbol": "INJECTED", "side": "buy", "quantity": 1, "price": 1, "executed_at": "2026-01-06T01:00:00Z"}), 404},
			{"POST", "/cash-transactions", isolationJSON(t, map[string]any{"account_id": owner.accounts[0], "type": "deposit", "amount": 1, "occurred_at": "2026-01-05T00:00:00Z"}), 404},
			{"PUT", "/cash-transactions/" + owner.cash, `{"type":"deposit","amount":1,"occurred_at":"2026-01-05T00:00:00Z"}`, 404}, {"DELETE", "/cash-transactions/" + owner.cash, "", 404},
			{"POST", "/cash-transactions", isolationJSON(t, map[string]any{"account_id": caller.accounts[0], "trade_id": owner.trade, "type": "deposit", "amount": 1, "occurred_at": "2026-01-05T00:00:00Z"}), 400},
			{"GET", "/notes/" + owner.note, "", 404}, {"PATCH", "/notes/" + owner.note, `{"body":"tampered","occurred_at":"2026-01-05"}`, 404}, {"DELETE", "/notes/" + owner.note, "", 404},
			{"GET", "/setups/" + owner.setup, "", 404}, {"PATCH", "/setups/" + owner.setup, `{"name":"tampered"}`, 404}, {"DELETE", "/setups/" + owner.setup, "", 404},
			{"PATCH", "/tags/" + owner.tag, `{"name":"tampered"}`, 404}, {"DELETE", "/tags/" + owner.tag, "", 404},
			{"PATCH", "/trades/" + caller.trade, isolationJSON(t, map[string]any{"setup_id": owner.setup}), 400},
			{"PATCH", "/trades/" + caller.trade, isolationJSON(t, map[string]any{"setup_ids": []string{caller.setup, owner.setup}}), 400},
			{"PATCH", "/trades/" + caller.trade, isolationJSON(t, map[string]any{"notes": "tampered", "tag_ids": []string{owner.tag}}), 400},
			{"GET", news, "", 404}, {"PATCH", news, `{"title":"tampered","source":"fake","published_at":"2026-01-05T00:00:00Z"}`, 404}, {"DELETE", news, "", 404},
			{"POST", news + "/assets", `{"asset_type":"stock","symbol":"X"}`, 404}, {"PATCH", news + "/assets/" + owner.asset, `{"asset_type":"stock","symbol":"X"}`, 404}, {"DELETE", news + "/assets/" + owner.asset, "", 404},
			{"PATCH", "/news/" + caller.news + "/assets/" + owner.asset, `{"asset_type":"stock","symbol":"X"}`, 404},
			{"POST", news + "/predictions", isolationJSON(t, map[string]any{"news_asset_id": owner.asset, "direction": "bullish", "horizons": []int{1}}), 404},
			{"POST", "/news/" + caller.news + "/predictions", isolationJSON(t, map[string]any{"news_asset_id": owner.asset, "direction": "bullish", "horizons": []int{1}}), 404},
			{"PATCH", news + "/predictions/" + owner.prediction, `{"direction":"bearish","horizons":[1]}`, 404}, {"DELETE", news + "/predictions/" + owner.prediction, "", 404},
			{"PATCH", "/news/" + caller.news + "/predictions/" + owner.prediction, `{"direction":"bearish","horizons":[1]}`, 404},
			{"POST", news + "/predictions/" + owner.prediction + "/validate", `{}`, 404}, {"GET", news + "/predictions/" + owner.prediction + "/validations", "", 404},
			{"POST", news + "/analyze", `{}`, 404}, {"POST", news + "/analysis/accept", `{}`, 404}, {"GET", news + "/export", "", 404},
			{"GET", trade + "/attachments", "", 404}, {"POST", trade + "/attachments", "", 404}, {"GET", "/attachments/" + owner.attachment + "/file", "", 404}, {"DELETE", "/attachments/" + owner.attachment, "", 404},
			{"GET", "/media/" + owner.media + "/file", "", 404}, {"DELETE", "/media/" + owner.media, "", 404},
			{"GET", "/access-tokens/" + owner.patID + "/uses", "", 200}, {"DELETE", "/access-tokens/" + owner.patID, "", 404},
			{"DELETE", "/share-links/" + owner.share, "", 404}, {"POST", "/share-links", isolationJSON(t, map[string]any{"account_id": owner.accounts[0]}), 404},
			{"POST", "/imports/" + owner.batch + "/commit", "", 404}, {"DELETE", "/imports/" + owner.batch, "", 404},
			{"PATCH", "/settings/alert-channels/" + owner.channel, `{"enabled":false}`, 404}, {"DELETE", "/settings/alert-channels/" + owner.channel, "", 404},
		}
		for _, tc := range cases {
			t.Run(caller.marker+"/"+tc.method+tc.path, func(t *testing.T) {
				for _, token := range []string{caller.session, caller.pat} {
					before, files := f.databaseState(t), f.fileState(t)
					rec := do(f.s, tc.method, "/api/v1"+tc.path, tc.body, token)
					require.Equal(t, tc.denied, rec.Code, "%s", rec.Body.String())
					require.Equal(t, before, f.databaseState(t), "denied request changed rows")
					require.Equal(t, files, f.fileState(t), "denied request changed files")
					require.NotContains(t, rec.Body.String(), owner.marker)
					require.False(t, strings.Contains(rec.Body.String(), owner.pat), "response leaked a token")
				}
				require.Equal(t, 401, do(f.s, tc.method, "/api/v1"+tc.path, tc.body, "").Code)
			})
		}
		for _, path := range []string{acc, trade, news, "/notes/" + owner.note, "/setups/" + owner.setup, "/attachments/" + owner.attachment + "/file", "/media/" + owner.media + "/file"} {
			require.Equal(t, 200, do(f.s, "GET", "/api/v1"+path, "", owner.session).Code, "owner read-back %s", path)
		}
		for _, path := range []string{acc, trade, news, "/notes/" + owner.note, "/setups/" + owner.setup} {
			missing := path[:strings.LastIndex(path, "/")+1] + "missing"
			require.Equal(t, do(f.s, "GET", "/api/v1"+missing, "", caller.session).Body.String(), do(f.s, "GET", "/api/v1"+path, "", caller.session).Body.String())
		}
	}
}

func TestCrossUserIsolationAggregatesAndLists(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	paths := []string{"/accounts", "/trades", "/trades?status=open", "/trades?status=closed", "/cash-transactions", "/notes", "/setups", "/tags", "/news", "/news/performance", "/imports", "/flex-sync", "/share-links", "/settings/alert-channels", "/alerts/events?limit=1"}
	for _, name := range []string{"summary", "r-summary", "equity-curve", "daily", "breakdown", "compliance", "behavior", "execution-score", "montecarlo"} {
		paths = append(paths, "/analytics/"+name+"?seed=244&paths=10&horizon=10&target_currency=JPY&by=symbol")
	}
	for _, by := range []string{"setup", "tag", "mistake", "trade_quality"} {
		paths = append(paths, "/analytics/breakdown?by="+by+"&target_currency=JPY")
	}
	paths = append(paths, "/analytics/account-value?from=2026-01-05&to=2026-01-06")
	baseline := map[string]string{}
	for _, path := range paths {
		rec := do(f.s, "GET", "/api/v1"+path, "", a.session)
		require.Equal(t, 200, rec.Code, "%s: %s", path, rec.Body.String())
		baseline[path] = rec.Body.String()
	}
	b := f.seed(t, "b", "ISOB", 9876)
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			sep := "?"
			if strings.Contains(path, "?") {
				sep = "&"
			}
			suffixes := []string{"", sep + "account_id=" + a.accounts[0] + "," + a.accounts[1], sep + "account_id=" + a.accounts[0] + "&account_id=" + a.accounts[1], sep + "user_id=" + b.id + "&account_id=" + a.accounts[0] + "," + a.accounts[1] + "," + b.accounts[0], sep + "account_id=" + a.accounts[0] + "&account_id=" + a.accounts[1] + "&account_id=" + b.accounts[1], sep + "sort=desc&limit=1&offset=1&q=" + b.marker}
			for _, suffix := range suffixes {
				rec := do(f.s, "GET", "/api/v1"+path+suffix, "", a.session)
				strictScope := strings.HasPrefix(path, "/trades") || strings.HasPrefix(path, "/analytics/") && !strings.HasPrefix(path, "/analytics/account-value")
				if (strings.Contains(suffix, b.accounts[0]) || strings.Contains(suffix, b.accounts[1])) && strictScope {
					require.Equal(t, 400, rec.Code)
					require.JSONEq(t, `{"error":{"code":"unknown_currency","message":"account scope currency could not be resolved"}}`, rec.Body.String())
					continue
				}
				require.Equal(t, 200, rec.Code, "%s", rec.Body.String())
				require.NotContains(t, rec.Body.String(), b.marker)
				require.NotContains(t, rec.Body.String(), b.symbol)
				require.NotContains(t, rec.Body.String(), b.accounts[0])
				require.NotContains(t, rec.Body.String(), b.id)
				// Mixed IDs must have the same aggregates as both owned accounts.
				if suffix == "" || strings.Contains(suffix, "account_id=") {
					require.JSONEq(t, baseline[path], rec.Body.String(), "%s", path)
				}
			}
			require.Equal(t, 401, do(f.s, "GET", "/api/v1"+path, "", "").Code)
		})
	}
	for _, account := range []string{a.accounts[0], b.accounts[0], "missing"} {
		for _, path := range []string{"/trades", "/cash-transactions", "/executions", "/analytics/summary", "/analytics/account-value"} {
			extra := ""
			if strings.Contains(path, "account-value") {
				extra = "&from=2026-01-05&to=2026-01-06"
			}
			rec := do(f.s, "GET", "/api/v1"+path+"?account_id="+account+extra, "", a.session)
			if account != a.accounts[0] && (path == "/trades" || path == "/analytics/summary") {
				require.Equal(t, 400, rec.Code)
				require.JSONEq(t, `{"error":{"code":"unknown_currency","message":"account scope currency could not be resolved"}}`, rec.Body.String())
				continue
			}
			require.Equal(t, 200, rec.Code)
			require.NotContains(t, rec.Body.String(), b.symbol)
			if path == "/trades" || path == "/cash-transactions" || path == "/executions" {
				var rows []any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rows))
				if account != a.accounts[0] {
					require.Empty(t, rows)
				} else {
					require.NotEmpty(t, rows)
				}
			}
			if path == "/analytics/summary" {
				var obj map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &obj))
				if account == a.accounts[0] {
					require.Equal(t, float64(1), obj["total_trades"])
					require.Equal(t, float64(20), obj["net_pnl"])
				} else {
					require.Equal(t, float64(0), obj["total_trades"])
				}
			}
		}
	}
	// Currency conversion, date and symbol filters must narrow only A's rows.
	for _, query := range []string{"target_currency=USD", "symbol=" + b.symbol, "from=2026-01-06T00:00:00Z&to=2026-01-07T00:00:00Z"} {
		rec := do(f.s, "GET", "/api/v1/analytics/summary?"+query, "", a.session)
		require.Equal(t, 200, rec.Code)
		var obj map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &obj))
		if query == "target_currency=USD" {
			var original map[string]any
			require.NoError(t, json.Unmarshal([]byte(baseline["/analytics/summary?seed=244&paths=10&horizon=10&target_currency=JPY&by=symbol"]), &original))
			require.InDelta(t, original["net_pnl"].(float64)/150, obj["net_pnl"], .011)
		} else {
			require.Equal(t, float64(0), obj["total_trades"])
		}
	}
}

func TestCrossUserIsolationInstanceSettings(t *testing.T) {
	f := newIsolationFixture(t)
	owner := registerAndLogin(t, f.s, "owner@isolation.invalid")
	member := registerAndLogin(t, f.s, "member@isolation.invalid")
	for _, name := range []string{"ocr", "coach"} {
		path := "/settings/" + name
		isolationObject(t, f.s, "PUT", path, `{"enabled":false,"base_url":"https://example.invalid","model":"synthetic","api_key":"synthetic-key","custom_prompt":"owner-private-prompt"}`, owner, 200)
		require.Equal(t, 200, do(f.s, "GET", "/api/v1"+path, "", owner).Code)
		for _, tc := range []struct{ method, suffix string }{{"GET", ""}, {"PUT", ""}, {"POST", "/test"}, {"POST", "/models"}} {
			before := f.databaseState(t)
			rec := do(f.s, tc.method, "/api/v1"+path+tc.suffix, `{"base_url":"https://example.invalid","model":"synthetic"}`, member)
			require.Equal(t, 403, rec.Code)
			require.Equal(t, before, f.databaseState(t))
			require.NotContains(t, rec.Body.String(), "owner-private-prompt")
			require.Equal(t, 401, do(f.s, tc.method, "/api/v1"+path+tc.suffix, `{}`, "").Code)
		}
	}
}

func TestCrossUserIsolationFilesImportsExports(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	before, files := f.databaseState(t), f.fileState(t)
	csv := "symbol,side,quantity,price,executed_at\nISOC,buy,1,100,2026-01-05T01:00:00Z\n"
	mapping := `{"symbol":"symbol","side":"side","quantity":"quantity","price":"price","executed_at":"executed_at"}`
	for _, path := range []string{"/imports", "/imports/commit", "/imports/" + b.batch + "/commit"} {
		r := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(r, multipartReq(t, "/api/v1"+path, a.session, csv, map[string]string{"account_id": b.accounts[0], "column_mapping": mapping}))
		require.Equal(t, 404, r.Code)
		require.Equal(t, before, f.databaseState(t))
		require.Equal(t, files, f.fileState(t))
	}
	r := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(r, imageUploadReq(t, "/api/v1/trades/"+b.trade+"/attachments", a.session, []byte("\x89PNG\r\n\x1a\nPRIVATE-upload")))
	require.Equal(t, 404, r.Code)
	require.Equal(t, files, f.fileState(t))
	// Raw storage keys, original filenames and encoded traversal never bypass the ID lookup.
	for _, path := range []string{"/attachments/chart.png/file", "/media/chart.png/file", "/attachments/..%2F" + b.id + "%2F" + b.attachment + "/file", "/media/..%2F" + b.id + "%2Fmedia%2F" + b.media + "/file", "/" + b.id + "/" + b.attachment, "/exports/" + b.batch} {
		require.Equal(t, 404, do(f.s, "GET", "/api/v1"+path, "", a.session).Code)
	}
	for _, u := range []isolationUser{a, b} {
		foreign := b
		if u.id == b.id {
			foreign = a
		}
		for _, route := range []string{"/exports", "/exports/trades"} {
			for _, format := range []string{"json", "csv", "zip"} {
				path := "/api/v1" + route + "?format=" + format + "&account_id=" + u.accounts[0]
				rec := do(f.s, "GET", path, "", u.session)
				require.Equal(t, 200, rec.Code)
				bodies := []string{rec.Body.String()}
				if format == "zip" {
					bodies = nil
					zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
					require.NoError(t, err)
					require.NotEmpty(t, zr.File)
					for _, entry := range zr.File {
						rc, err := entry.Open()
						require.NoError(t, err)
						data, err := io.ReadAll(rc)
						require.NoError(t, err)
						require.NoError(t, rc.Close())
						bodies = append(bodies, string(data))
						require.NotContains(t, entry.Name, "..")
						require.NotContains(t, entry.Name, foreign.id)
					}
				}
				found := false
				for _, body := range bodies {
					found = found || strings.Contains(body, u.symbol)
					for _, secret := range []string{foreign.symbol, foreign.marker, foreign.id, foreign.trade, foreign.accounts[0], foreign.attachment} {
						require.NotContains(t, body, secret)
					}
				}
				require.True(t, found, "export must contain owner data")
				require.Equal(t, 404, do(f.s, "GET", "/api/v1"+route+"?format="+format+"&account_id="+foreign.accounts[0], "", u.session).Code)
				require.Equal(t, 401, do(f.s, "GET", path, "", "").Code)
			}
		}
		rec := do(f.s, "GET", "/api/v1/news/export", "", u.session)
		require.Equal(t, 200, rec.Code)
		require.Contains(t, rec.Body.String(), u.marker)
		require.NotContains(t, rec.Body.String(), foreign.marker)
		for _, query := range []string{"id=" + foreign.news, "id=" + u.news + "&id=" + foreign.news} {
			require.Equal(t, 404, do(f.s, "GET", "/api/v1/news/export?"+query, "", u.session).Code)
		}
		require.Equal(t, 200, do(f.s, "GET", "/api/v1/news/"+u.news+"/export", "", u.session).Code)
	}
	require.Equal(t, files, f.fileState(t), "exports must not leave generated files")
	// Positive preview and commit prove foreign rejection was not a broken parser.
	for _, path := range []string{"/imports", "/imports/commit"} {
		r := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(r, multipartReq(t, "/api/v1"+path, a.session, csv, map[string]string{"account_id": a.accounts[0], "column_mapping": mapping}))
		require.Equal(t, 200, r.Code, r.Body.String())
	}
	require.Equal(t, files, f.fileState(t), "CSV uploads are not persisted as retrievable files")
}

func TestCrossUserIsolationTokensAndSharing(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	for _, u := range []isolationUser{a, b} {
		foreign := b
		if u.id == b.id {
			foreign = a
		}
		rec := do(f.s, "GET", "/api/v1/access-tokens", "", u.pat)
		require.Equal(t, 200, rec.Code)
		require.NotContains(t, rec.Body.String(), foreign.patID)
		require.False(t, strings.Contains(rec.Body.String(), u.pat), "list leaked a token")
		rec = do(f.s, "GET", "/api/v1/access-tokens/"+u.patID+"/uses", "", u.session)
		require.Equal(t, 200, rec.Code)
		var uses []any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &uses))
		require.NotEmpty(t, uses)
		require.Equal(t, "[]\n", do(f.s, "GET", "/api/v1/access-tokens/"+foreign.patID+"/uses", "", u.session).Body.String())
		// The bearer link deliberately exposes a scoped aggregate, never private details.
		rec = do(f.s, "GET", "/api/v1/public/share/"+u.shareToken, "", "")
		require.Equal(t, 200, rec.Code)
		for _, secret := range []string{u.marker, foreign.marker, u.id, u.trade, u.accounts[0], foreign.symbol, "net_pnl"} {
			require.NotContains(t, rec.Body.String(), secret)
		}
		var pub map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pub))
		require.Equal(t, float64(1), pub["summary"].(map[string]any)["total_trades"])
		// URL query parameters cannot expand the scope encoded by the owner.
		require.JSONEq(t, rec.Body.String(), do(f.s, "GET", "/api/v1/public/share/"+u.shareToken+"?account_id="+foreign.accounts[0]+"&show_amounts=true", "", "").Body.String())
		require.Equal(t, 204, do(f.s, "DELETE", "/api/v1/share-links/"+u.share, "", u.session).Code)
		revoked := do(f.s, "GET", "/api/v1/public/share/"+u.shareToken, "", "")
		require.Equal(t, 404, revoked.Code)
		require.Equal(t, do(f.s, "GET", "/api/v1/public/share/missing", "", "").Body.String(), revoked.Body.String())
		require.Equal(t, 204, do(f.s, "DELETE", "/api/v1/access-tokens/"+u.patID, "", u.session).Code)
		rec = do(f.s, "GET", "/api/v1/me", "", u.pat)
		require.Equal(t, 401, rec.Code)
		require.False(t, strings.Contains(rec.Body.String(), u.pat), "error leaked a token")
	}
	// Expiration is a deterministic fixture change, not a wall-clock sleep.
	obj := isolationObject(t, f.s, "POST", "/share-links", `{}`, a.session, 201)
	_, err := f.conn.Exec(`UPDATE share_links SET expires_at=? WHERE id=?`, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), obj["id"])
	require.NoError(t, err)
	require.Equal(t, 404, do(f.s, "GET", "/api/v1/public/share/"+obj["token"].(string), "", "").Code)
	obj = isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"expired"}`, a.session, 201)
	_, err = f.conn.Exec(`UPDATE access_tokens SET expires_at=? WHERE id=?`, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), obj["id"])
	require.NoError(t, err)
	require.Equal(t, 401, do(f.s, "GET", "/api/v1/me", "", obj["token"].(string)).Code)
}

func TestCrossUserIsolationSelfSettings(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	cases := []struct{ path, method, body string }{
		{"/me/preferences", "PATCH", `{"secret":"PRIVATE-%s","user_id":"%s"}`},
		{"/settings/checklist-template", "PUT", `{"content":"PRIVATE-%s","user_id":"%s"}`},
		{"/settings/risk-rules", "PUT", `{"max_daily_loss":%s,"user_id":"%s"}`},
		{"/settings/annual-goal?year=2026", "PUT", `{"year":2026,"currency":"JPY","amount":%s,"user_id":"%s"}`},
		{"/settings/alerts", "PUT", `{"timezone":"Asia/Tokyo","loss_streak_n":%s,"user_id":"%s"}`},
	}
	for i, tc := range cases {
		av, bv := "a", "b"
		if i >= 2 {
			av, bv = "2", "9"
		}
		isolationObject(t, f.s, tc.method, tc.path, fmt.Sprintf(tc.body, bv, a.id), b.session, 200)
		before := do(f.s, "GET", "/api/v1"+tc.path, "", b.session)
		require.Equal(t, 200, before.Code)
		isolationObject(t, f.s, tc.method, tc.path, fmt.Sprintf(tc.body, av, b.id), a.session, 200)
		sep := "?"
		if strings.Contains(tc.path, "?") {
			sep = "&"
		}
		require.JSONEq(t, before.Body.String(), do(f.s, "GET", "/api/v1"+tc.path+sep+"user_id="+a.id, "", b.session).Body.String())
		require.NotEqual(t, before.Body.String(), do(f.s, "GET", "/api/v1"+tc.path, "", a.session).Body.String())
	}
	before := do(f.s, "GET", "/api/v1/trades/"+a.trade, "", a.session)
	after := isolationObject(t, f.s, "PATCH", "/trades/"+a.trade, `{"notes":"updated-owner-note"}`, a.session, 200)
	var old map[string]any
	require.NoError(t, json.Unmarshal(before.Body.Bytes(), &old))
	for _, key := range []string{"setup_ids", "initial_risk", "target_price", "stop_price", "net_pnl", "direction", "fills", "tags"} {
		require.Equal(t, old[key], after[key], key)
	}
	require.Equal(t, "updated-owner-note", after["notes"])
}

func TestCrossUserIsolationAdminBackupAndAnonymousRoutes(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "a", "ISOA", 2)
	member := f.seed(t, "b", "ISOB", 9876)
	for _, tc := range []struct{ method, path, body string }{
		{"GET", "/admin/users", ""}, {"POST", "/admin/users", `{"email":"x@isolation.invalid","password":"synthetic-pass"}`},
		{"PATCH", "/admin/users/" + owner.id, `{"is_admin":true}`}, {"POST", "/admin/users/" + owner.id + "/password", `{"new_password":"synthetic-pass"}`}, {"DELETE", "/admin/users/" + owner.id, ""},
		{"GET", "/admin/backup", ""}, {"POST", "/admin/backup", ""},
	} {
		before := f.databaseState(t)
		for _, tok := range []string{member.session, member.pat} {
			rec := do(f.s, tc.method, "/api/v1"+tc.path, tc.body, tok)
			require.Equal(t, 403, rec.Code)
			require.NotContains(t, rec.Body.String(), filepath.Dir(f.files))
		}
		require.Equal(t, before, f.databaseState(t))
	}
	st := isolationObject(t, f.s, "POST", "/admin/backup", `{"dir":"../../escape","keep":0}`, owner.session, 200)
	require.Equal(t, float64(1), st["file_count"])
	require.Equal(t, float64(2), st["keep"])
	dir := st["dir"].(string)
	name := st["latest"].(map[string]any)["name"].(string)
	contents, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	for _, path := range []string{"/api/v1/admin/backup/" + name, "/api/v1/admin/backup/" + name + "/download", "/backups/" + name, "/api/v1/attachments/" + name + "/file"} {
		rec := do(f.s, "GET", path, "", member.session)
		want := 404
		if strings.HasPrefix(path, "/api/v1/admin/") {
			want = 403
		}
		require.Equal(t, want, rec.Code)
		require.NotContains(t, rec.Body.String(), dir)
	}
	unchanged, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	require.Equal(t, sha256.Sum256(contents), sha256.Sum256(unchanged))
	// Every registered private route must reject anonymous callers, including newly added routes.
	public := map[string]bool{"/api/v1/setup/status": true, "/api/v1/setup": true, "/api/v1/auth/login": true, "/api/v1/auth/register": true, "/api/v1/auth/refresh": true, "/api/v1/public/share/:token": true}
	for _, route := range f.s.Echo.Router().Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") || public[route.Path] || route.Method == "echo_route_not_found" {
			continue
		}
		path := route.Path
		for _, part := range strings.Split(path, "/") {
			if strings.HasPrefix(part, ":") {
				path = strings.ReplaceAll(path, part, "known-id")
			}
		}
		require.Equal(t, 401, do(f.s, route.Method, path, `{}`, "").Code, "%s %s", route.Method, route.Path)
	}
}

func TestCrossUserIsolationNotificationDelivery(t *testing.T) {
	f := newIsolationFixture(t)
	q := store.NewForDriver(f.conn, "sqlite")
	jwt := auth.NewJWT("notification-fixture")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Logger: logger, Alerts: alerts.NewService(q, logger, true)})
	a := registerAndLogin(t, s, "a@notification.invalid")
	b := registerAndLogin(t, s, "b@notification.invalid")
	var received atomic.Int64
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received.Add(1); w.WriteHeader(200) }))
	t.Cleanup(endpoint.Close)
	ch := isolationObject(t, s, "POST", "/settings/alert-channels", isolationJSON(t, map[string]any{"kind": "webhook", "target": endpoint.URL, "label": "B-private-channel"}), b, 200)["id"].(string)
	before := f.databaseState(t)
	require.Equal(t, 404, do(s, "POST", "/api/v1/settings/alert-channels/"+ch+"/test", "", a).Code)
	require.Zero(t, received.Load())
	require.Equal(t, before, f.databaseState(t))
	require.Equal(t, 200, do(s, "POST", "/api/v1/settings/alert-channels/"+ch+"/test", "", b).Code)
	require.EqualValues(t, 1, received.Load())
	uid := isolationObject(t, s, "GET", "/me", "", b, 200)["id"].(string)
	_, err := q.InsertAlertEvent(context.Background(), store.InsertAlertEventParams{ID: "b-event", UserID: uid, Rule: "risk", DedupeKey: "synthetic", Title: "B-private-alert", Body: "B-private-alert"})
	require.NoError(t, err)
	rec := do(s, "GET", "/api/v1/alerts/events?user_id="+uid+"&limit=1", "", a)
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `[]`, rec.Body.String())
	require.Contains(t, do(s, "GET", "/api/v1/alerts/events", "", b).Body.String(), "B-private-alert")
	isolationObject(t, s, "POST", "/me/push-tokens", `{"token":"ExpoPushToken[synthetic-B]","label":"B-private-push"}`, b, 200)
	before = f.databaseState(t)
	require.Equal(t, 204, do(s, "DELETE", "/api/v1/me/push-tokens", `{"token":"ExpoPushToken[synthetic-B]"}`, a).Code)
	require.Equal(t, before, f.databaseState(t))
}

func TestCrossUserIsolationOwnerMutations(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	readback := []string{"/accounts/" + b.accounts[0], "/trades/" + b.trade, "/news/" + b.news, "/notes/" + b.note, "/media/" + b.media + "/file"}
	baseline := map[string]string{}
	for _, path := range readback {
		rec := do(f.s, "GET", "/api/v1"+path, "", b.session)
		require.Equal(t, 200, rec.Code)
		baseline[path] = rec.Body.String()
	}
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/trades/regroup", isolationJSON(t, map[string]any{"account_id": a.accounts[0]}), 204},
		{"PUT", "/accounts/" + a.accounts[0], `{"name":"owner-update"}`, 200},
		{"PATCH", "/notes/" + a.note, `{"body":"owner-update","occurred_at":"2026-01-05"}`, 200},
		{"PATCH", "/setups/" + a.setup, `{"name":"owner-update"}`, 200},
		{"PATCH", "/tags/" + a.tag, `{"name":"owner-update"}`, 204},
		{"PUT", "/cash-transactions/" + a.cash, `{"type":"deposit","amount":100,"currency":"JPY","occurred_at":"2026-01-05T00:00:00Z"}`, 200},
		{"PATCH", "/news/" + a.news, `{"title":"owner-update","source":"synthetic","published_at":"2026-01-05T00:00:00Z"}`, 200},
		{"PATCH", "/news/" + a.news + "/assets/" + a.asset, `{"asset_type":"stock","symbol":"ISOA","market":"JP"}`, 200},
		{"PATCH", "/news/" + a.news + "/predictions/" + a.prediction, `{"direction":"bearish","horizons":[1]}`, 200},
		{"POST", "/news/" + a.news + "/predictions/" + a.prediction + "/validate", `{}`, 200},
		{"PATCH", "/settings/alert-channels/" + a.channel, `{"enabled":false}`, 200},
		{"DELETE", "/media/" + a.media, "", 204}, {"DELETE", "/attachments/" + a.attachment, "", 204},
		{"DELETE", "/cash-transactions/" + a.cash, "", 204}, {"DELETE", "/notes/" + a.note, "", 204},
		{"DELETE", "/news/" + a.news + "/predictions/" + a.prediction, "", 204}, {"DELETE", "/news/" + a.news + "/assets/" + a.asset, "", 204}, {"DELETE", "/news/" + a.news, "", 204},
		{"DELETE", "/setups/" + a.setup, "", 204}, {"DELETE", "/tags/" + a.tag, "", 204},
		{"DELETE", "/accounts/" + a.accounts[0] + "/flex-sync", "", 204}, {"DELETE", "/accounts/" + a.accounts[0] + "/prop-settings", "", 204},
		{"DELETE", "/settings/alert-channels/" + a.channel, "", 204},
		{"DELETE", "/executions/" + a.execution, "", 200}, {"DELETE", "/trades/" + a.trade, "", 204},
		{"DELETE", "/imports/" + a.batch, "", 204}, {"DELETE", "/accounts/" + a.accounts[0] + "/trades", "", 204}, {"DELETE", "/accounts/" + a.accounts[0], "", 204},
	} {
		rec := do(f.s, tc.method, "/api/v1"+tc.path, tc.body, a.session)
		require.Equal(t, tc.status, rec.Code, "%s %s: %s", tc.method, tc.path, rec.Body.String())
	}
	for _, path := range readback {
		rec := do(f.s, "GET", "/api/v1"+path, "", b.session)
		require.Equal(t, 200, rec.Code)
		require.Equal(t, baseline[path], rec.Body.String(), "owner operation changed B")
	}
}

func TestCrossUserIsolationCredentials(t *testing.T) {
	f := newIsolationFixture(t)
	a := registerAndLogin(t, f.s, "a@credential.invalid")
	b := registerAndLogin(t, f.s, "b@credential.invalid")
	bid := isolationObject(t, f.s, "GET", "/me", "", b, 200)["id"].(string)
	// Client-supplied identifiers cannot redirect a self-service operation to B.
	start := isolationObject(t, f.s, "POST", "/me/totp/start?user_id="+bid, isolationJSON(t, map[string]any{"user_id": bid}), a, 200)
	secret := start["secret"].(string)
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	rec := do(f.s, "POST", "/api/v1/me/totp/confirm", isolationJSON(t, map[string]any{"secret": secret, "code": code, "user_id": bid}), a)
	require.Equal(t, 204, rec.Code)
	require.Equal(t, false, isolationObject(t, f.s, "GET", "/me", "", b, 200)["totp_enabled"])
	require.Equal(t, true, isolationObject(t, f.s, "GET", "/me", "", a, 200)["totp_enabled"])
	rec = do(f.s, "POST", "/api/v1/me/totp/disable", isolationJSON(t, map[string]any{"code": code, "password": testPassword, "user_id": bid}), a)
	require.Equal(t, 204, rec.Code)
	rec = do(f.s, "PUT", "/api/v1/me/password", isolationJSON(t, map[string]any{"current_password": testPassword, "new_password": "new-synthetic-password", "user_id": bid}), a)
	require.Equal(t, 200, rec.Code)
	require.Equal(t, 200, do(f.s, "POST", "/api/v1/auth/login", `{"email":"b@credential.invalid","password":"`+testPassword+`"}`, "").Code)
	require.Equal(t, 401, do(f.s, "POST", "/api/v1/auth/login", `{"email":"a@credential.invalid","password":"`+testPassword+`"}`, "").Code)
	require.Equal(t, 200, do(f.s, "POST", "/api/v1/auth/login", `{"email":"a@credential.invalid","password":"new-synthetic-password"}`, "").Code)
}

func TestCrossUserIsolationRouteInventory(t *testing.T) {
	f := newIsolationFixture(t)
	data, err := os.ReadFile("../../../docs/validation/issue244/endpoints.md")
	require.NoError(t, err)
	inventoried := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 4 {
			continue
		}
		method := strings.TrimSpace(fields[1])
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			continue
		}
		inventoried[method+" "+strings.Trim(strings.TrimSpace(fields[2]), "`")] = true
	}
	registered := map[string]bool{}
	for _, route := range f.s.Echo.Router().Routes() {
		switch route.Method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			continue
		}
		registered[route.Method+" "+route.Path] = true
	}
	require.Equal(t, inventoried, registered, "a route was added/removed: audit its ownership and update the #244 inventory")
}

func TestCrossUserIsolationConfiguredServices(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	q := store.NewForDriver(f.conn, "sqlite")
	jwt := auth.NewJWT("disposable-isolation-test-secret")
	var received atomic.Int64
	var remote *httptest.Server
	remote = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		if r.URL.Path == "/send" {
			fmt.Fprintf(w, `<FlexStatementResponse><Status>Success</Status><ReferenceCode>fixture</ReferenceCode><Url>%s/get</Url></FlexStatementResponse>`, remote.URL)
			return
		}
		if r.URL.Path == "/get" {
			fmt.Fprint(w, "ClientAccountID,Symbol,Buy/Sell,Quantity,TradePrice,DateTime,IBCommission,AssetClass,Multiplier,Put/Call\nSYNTHETIC,ISOF,BUY,1,100,20260105;093000,0,STK,1,\n")
			return
		}
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		payload := `{"notes":[{"tone":"tip","headline":"private fixture review","detail":"synthetic owner context"}]}`
		if request["stream"] == true {
			enc, _ := json.Marshal(payload)
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s}}]}\n\ndata: [DONE]\n\n", enc)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": payload}}}})
	}))
	t.Cleanup(remote.Close)
	s := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), FlexClient: &flexsync.Client{BaseURL: remote.URL + "/send", HTTP: remote.Client(), PollAttempts: 1}})
	isolationObject(t, s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": remote.URL, "api_key": "synthetic-unused-key", "model": "fixture"}), a.session, 200)
	for _, u := range []isolationUser{a, b} {
		other := b
		if u.id == b.id {
			other = a
		}
		for _, path := range []string{"/trades/" + other.trade + "/coach", "/trades/" + other.trade + "/coach/stream", "/accounts/" + other.accounts[0] + "/flex-sync/run"} {
			count := received.Load()
			before := f.databaseState(t)
			require.Equal(t, 404, do(s, "POST", "/api/v1"+path, `{}`, u.session).Code)
			require.Equal(t, count, received.Load())
			require.Equal(t, before, f.databaseState(t))
		}
		rec := do(s, "POST", "/api/v1/trades/"+u.trade+"/coach", `{}`, u.session)
		require.Equal(t, 200, rec.Code)
		require.Contains(t, rec.Body.String(), `"source":"llm"`)
		rec = do(s, "POST", "/api/v1/trades/"+u.trade+"/coach/stream", `{}`, u.session)
		require.Equal(t, 200, rec.Code)
		require.Contains(t, rec.Body.String(), "event: done")
		rec = do(s, "GET", "/api/v1/trades/"+u.trade+"/coach/reviews", "", u.session)
		require.Equal(t, 200, rec.Code)
		require.Contains(t, rec.Body.String(), "private fixture review")
		require.Equal(t, 200, do(s, "POST", "/api/v1/accounts/"+u.accounts[0]+"/flex-sync/run", `{}`, u.session).Code)
	}
	// The legacy batch commit must still use the authenticated batch's parent,
	// even when multipart fields try to redirect it to the other user's account.
	before := do(f.s, "GET", "/api/v1/executions?account_id="+b.accounts[1], "", b.session)
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/"+a.batch+"/commit", a.session, "fixture.html", mt5StatementHTML, map[string]string{"account_id": b.accounts[1]}))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, before.Body.String(), do(f.s, "GET", "/api/v1/executions?account_id="+b.accounts[1], "", b.session).Body.String())
}

func TestCrossUserIsolationExcursionAndGoalDelete(t *testing.T) {
	f := newIsolationFixture(t)
	a := f.seed(t, "a", "ISOA", 2)
	b := f.seed(t, "b", "ISOB", 9876)
	for _, u := range []isolationUser{a, b} {
		isolationObject(t, f.s, "PUT", "/settings/annual-goal", `{"year":2026,"amount":123,"currency":"JPY"}`, u.session, 200)
	}
	baseline := do(f.s, "GET", "/api/v1/settings/annual-goal?year=2026", "", b.session)
	require.Equal(t, 200, do(f.s, "DELETE", "/api/v1/settings/annual-goal?year=2026&user_id="+b.id, "", a.session).Code)
	require.JSONEq(t, baseline.Body.String(), do(f.s, "GET", "/api/v1/settings/annual-goal?year=2026", "", b.session).Body.String())
	provider := marketdata.NewYahooProvider()
	// A fixed holding period over 120 days always selects daily bars, independent
	// of the current date and Yahoo's moving intraday retention window.
	var timestamps []int64
	for day := 3; day <= 10; day++ {
		timestamps = append(timestamps, time.Date(2026, 1, day, 0, 0, 0, 0, time.UTC).Unix())
	}
	payload := isolationJSON(t, map[string]any{"chart": map[string]any{"result": []any{map[string]any{"timestamp": timestamps, "meta": map[string]any{"currency": "JPY", "exchangeTimezoneName": "Asia/Tokyo"}, "indicators": map[string]any{"quote": []any{map[string]any{"open": []int{100, 100, 100, 100, 100, 100, 100, 100}, "close": []int{105, 105, 105, 105, 105, 105, 105, 105}, "high": []int{110, 110, 110, 110, 110, 110, 110, 110}, "low": []int{90, 90, 90, 90, 90, 90, 90, 90}, "volume": []int{100, 100, 100, 100, 100, 100, 100, 100}}}}}}}})
	provider.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}
	q := store.NewForDriver(f.conn, "sqlite")
	jwt := auth.NewJWT("disposable-isolation-test-secret")
	s := api.New(api.Deps{JWT: jwt, Store: q, Trades: trades.NewService(q), Market: marketdata.NewService(q, provider), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	var trade string
	for i, side := range []string{"buy", "sell"} {
		at := "2025-08-01T10:00:00Z"
		if i == 1 {
			at = "2026-01-05T11:00:00Z"
		}
		obj := isolationObject(t, f.s, "POST", "/executions", isolationJSON(t, map[string]any{"account_id": a.accounts[0], "symbol": "EXCUR", "side": side, "quantity": 1, "price": 100 + i, "executed_at": at}), a.session, 201)
		trade = obj["trade_id"].(string)
	}
	path := "/api/v1/trades/" + trade + "/excursion"
	for _, tok := range []string{b.session, b.pat, ""} {
		before := f.databaseState(t)
		want := 404
		if tok == "" {
			want = 401
		}
		require.Equal(t, want, do(s, "POST", path, `{}`, tok).Code)
		require.Equal(t, before, f.databaseState(t))
	}
	rec := do(s, "POST", path, `{}`, a.session)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"interval":"D"`)
	detail := isolationObject(t, f.s, "GET", "/trades/"+trade, "", a.session, 200)
	require.NotNil(t, detail["mae"])
	require.NotNil(t, detail["mfe"])
}
