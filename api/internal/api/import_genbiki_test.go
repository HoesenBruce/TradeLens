package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenbikiImportReportsZeroAndTransferredBasis(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "genbiki@x.com")
	acc := accountID(t, s, tok)
	csv := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/08/01,1515,信用新規買,100,1000,0\n2026/08/04,1515,現引,100,1090,500\n"
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "sbi.csv", csv, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード"}`}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = do(s, http.MethodGet, "/api/v1/trades?account_id="+acc, "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var trs []struct {
		ID      string   `json:"id"`
		Status  string   `json:"status"`
		NetPnl  *float64 `json:"net_pnl"`
		Average float64  `json:"avg_entry_price"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &trs))
	require.Len(t, trs, 2)
	for _, tr := range trs {
		if tr.Status == "closed" {
			require.NotNil(t, tr.NetPnl)
			require.Zero(t, *tr.NetPnl)
		} else {
			require.Equal(t, 1005.0, tr.Average)
		}
		detail := do(s, http.MethodGet, "/api/v1/trades/"+tr.ID, "", tok)
		require.Equal(t, http.StatusOK, detail.Code)
		require.Contains(t, detail.Body.String(), "position_conversion")
		if tr.Status == "closed" {
			require.Contains(t, detail.Body.String(), `"gross_pnl":0`)
			require.Contains(t, detail.Body.String(), `"fees_total":0`)
		}
	}
	rec = do(s, http.MethodGet, "/api/v1/analytics/daily?account_id="+acc+"&date_basis=close", "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var days map[string]float64
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &days))
	require.NotEmpty(t, days)
	for _, d := range days {
		require.Zero(t, d)
	}
}

func TestGenbikiWithoutSourceHistoryDoesNotUseReferencePrice(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "missing-genbiki@x.com")
	acc := accountID(t, s, tok)
	csv := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/08/04,1515,現引,100,1090,500\n"
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "sbi.csv", csv, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード"}`}))
	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	rec = do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var fills []any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fills))
	require.Empty(t, fills)
}
