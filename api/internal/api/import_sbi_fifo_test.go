package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/importer"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSBIFIFOAPIConservationReplayAndAtomicRefusal(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "synthetic-347@test.invalid")
	account := accountID(t, s, token)
	upload := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		s.Echo.ServeHTTP(r, multipartFileReq(t, path, token, "synthetic-347.csv", body, map[string]string{"account_id": account, "column_mapping": `{"symbol":"銘柄コード","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","fees":"手数料/諸経費等","instrument_type":"=stock"}`}))
		return r
	}
	header := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等,信用区分,金利,貸株料,その他諸経費,受渡金額/決済損益\n"
	history := header + "2032-03-02T09:00:00+09:00,9101,信用新規買,100,100,0,制度信用,0,0,0,--\n2032-03-02T10:00:00+09:00,9101,信用新規買,100,100,0,制度信用,0,0,0,--\n2032-03-02T11:00:00+09:00,9101,信用返済売,150,110,0,制度信用,0,0,0,1500\n"
	result := upload("/api/v1/imports/commit", history)
	require.Equal(t, 200, result.Code, result.Body.String())
	get := func(path string) string {
		r := do(s, http.MethodGet, path+"?account_id="+account, "", token)
		require.Equal(t, 200, r.Code, r.Body.String())
		return r.Body.String()
	}
	before := get("/api/v1/executions")
	tradesBefore := get("/api/v1/trades")
	report := "実現損益詳細\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2032/03/02,9101,信用返済売,150,110,100,1500\n"
	// A valid pending enrichment plus an unresolved row must write nothing.
	mixedBefore := report + "2032/03/02,9999,信用返済売,1,110,100,10\n"
	require.Equal(t, 409, upload("/api/v1/imports/commit", mixedBefore).Code)
	require.Equal(t, before, get("/api/v1/executions"))
	var preview struct {
		Rows []importer.SBIMarginPnLRow `json:"enrichment_rows"`
	}
	for range 2 {
		r := upload("/api/v1/imports", report)
		require.Equal(t, 200, r.Code, r.Body.String())
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &preview))
		require.Equal(t, "fifo_inferred", preview.Rows[0].OpeningMatch)
		require.Len(t, preview.Rows[0].Allocations, 2)
		require.Equal(t, before, get("/api/v1/executions"))
	}
	var committed struct {
		Enriched int                        `json:"enriched"`
		Rows     []importer.SBIMarginPnLRow `json:"enrichment_rows"`
	}
	r := upload("/api/v1/imports/commit", report)
	require.Equal(t, 200, r.Code, r.Body.String())
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &committed))
	require.Equal(t, 1, committed.Enriched)
	require.Equal(t, preview.Rows, committed.Rows)
	after := get("/api/v1/executions")
	require.Contains(t, after, "sbi_opening_match_provenance")
	require.Contains(t, after, "fifo_inferred")
	var b, a []map[string]any
	require.NoError(t, json.Unmarshal([]byte(before), &b))
	require.NoError(t, json.Unmarshal([]byte(after), &a))
	require.Len(t, a, len(b))
	for i := range a {
		for _, key := range []string{"id", "quantity", "price", "fees", "commission", "executed_at", "dedup_hash"} {
			require.Equal(t, b[i][key], a[i][key], key)
		}
	}
	var bt, at []map[string]any
	require.NoError(t, json.Unmarshal([]byte(tradesBefore), &bt))
	require.NoError(t, json.Unmarshal([]byte(get("/api/v1/trades")), &at))
	require.Len(t, at, 1)
	for _, key := range []string{"id", "qty_opened", "qty_remaining", "avg_entry_price", "avg_exit_price", "net_pnl", "fees_total"} {
		require.Equal(t, bt[0][key], at[0][key], key)
	}
	require.Equal(t, 50.0, at[0]["qty_remaining"])
	require.Nil(t, at[0]["time_in_trade_secs"])
	require.Contains(t, at[0]["accounting_warning"], "fifo_inferred")
	r = upload("/api/v1/imports/commit", report)
	require.Equal(t, 200, r.Code, r.Body.String())
	require.Contains(t, r.Body.String(), `"enriched":0`)
	require.Equal(t, after, get("/api/v1/executions"))
	r = upload("/api/v1/imports/commit", history)
	require.Equal(t, 200, r.Code, r.Body.String())
	require.Contains(t, r.Body.String(), `"inserted":0`)
	require.Equal(t, after, get("/api/v1/executions"))
	// A missing close in the same report makes the entire batch fail atomically.
	invalid := report + "2032/03/02,9999,信用返済売,1,110,100,10\n"
	require.Equal(t, 409, upload("/api/v1/imports/commit", invalid).Code)
	require.Equal(t, after, get("/api/v1/executions"))
	conflict := strings.Replace(report, ",100,1500", ",100,1499", 1)
	require.Equal(t, 409, upload("/api/v1/imports/commit", conflict).Code)
	require.Equal(t, after, get("/api/v1/executions"))
}
