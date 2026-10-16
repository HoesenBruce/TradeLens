package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/importer"
)

func TestSBICashPnLPreviewCommitAndConflict(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "synthetic-pnl@x.com")
	acc := accountID(t, s, tok)
	upload := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		s.Echo.ServeHTTP(r, multipartFileReq(t, path, tok, "synthetic.csv", body, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","fees":"手数料/諸経費等","instrument_type":"=stock"}`}))
		return r
	}
	history := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/09/01,1515,現物買,200,900,0\n2026/09/03,1515,現物売,100,1000,10\n2026/09/04,1515,現物売,100,1100,10\n"
	require.Equal(t, 200, upload("/api/v1/imports/commit", history).Code)
	report := "実現損益詳細\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2026/09/03,1515,売却,100,1000,901,9890\n"
	before := do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok).Body.String()
	tradesBefore := do(s, http.MethodGet, "/api/v1/trades?account_id="+acc, "", tok).Body.String()
	preview := upload("/api/v1/imports", report)
	require.Equal(t, 200, preview.Code, preview.Body.String())
	require.Contains(t, preview.Body.String(), "enrichable")
	require.Equal(t, before, do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok).Body.String())
	var result struct {
		Enriched int                        `json:"enriched"`
		Rows     []importer.SBIMarginPnLRow `json:"enrichment_rows"`
	}
	commit := upload("/api/v1/imports/commit", report)
	require.Equal(t, 200, commit.Code, commit.Body.String())
	require.NoError(t, json.Unmarshal(commit.Body.Bytes(), &result))
	require.Equal(t, 1, result.Enriched)
	after := do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok).Body.String()
	require.Contains(t, after, "pretax")
	require.Contains(t, after, "sbi_realized_pnl_detail")
	// Enrichment cannot replace executions or review identities.
	var b, a []map[string]any
	require.NoError(t, json.Unmarshal([]byte(before), &b))
	require.NoError(t, json.Unmarshal([]byte(after), &a))
	require.Len(t, a, len(b))
	for i := range a {
		require.Equal(t, b[i]["id"], a[i]["id"])
		require.Equal(t, b[i]["price"], a[i]["price"])
	}
	require.Equal(t, tradesBefore, do(s, http.MethodGet, "/api/v1/trades?account_id="+acc, "", tok).Body.String())
	repeat := upload("/api/v1/imports/commit", report)
	require.Equal(t, 200, repeat.Code)
	require.Contains(t, repeat.Body.String(), "already_enriched")
	// A conflict plus a valid new close must roll back as a unit.
	mixed := strings.Replace(report, "901,9890", "902,9790", 1) + "2026/09/04,1515,売却,100,1100,901,19890\n"
	require.Contains(t, upload("/api/v1/imports", mixed).Body.String(), "conflict")
	require.Equal(t, 409, upload("/api/v1/imports/commit", mixed).Code)
	require.Equal(t, after, do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok).Body.String())
	unmatched := strings.Replace(report, "1515", "1516", 1)
	require.Contains(t, upload("/api/v1/imports", unmatched).Body.String(), "no_matching_execution")
}
