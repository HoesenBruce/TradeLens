package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/importer"
)

func TestSBIMarginPnLDisambiguationPreviewCommitAndReplay(t *testing.T) {
	// Both files are fictional; report order deliberately differs from history order.
	history, err := os.ReadFile("../importer/testdata/sbi-margin-pnl-disambiguation-history.csv")
	require.NoError(t, err)
	report, err := os.ReadFile("../importer/testdata/sbi-margin-pnl-disambiguation-report.csv")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(report)), "\n")
	slices.Reverse(lines[2:])
	reversed := strings.Join(lines, "\n") + "\n"

	for _, tc := range []struct{ name, body string }{{"report_order", string(report)}, {"reversed_report", reversed}} {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			s := testServer(t)
			token := registerAndLogin(t, s, "synthetic-350@test.invalid")
			account := accountID(t, s, token)
			upload := func(path, csv string) *httptest.ResponseRecorder {
				r := httptest.NewRecorder()
				s.Echo.ServeHTTP(r, multipartFileReq(t, path, token, "synthetic-350.csv", csv, map[string]string{"account_id": account, "column_mapping": `{"symbol":"銘柄コード","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","fees":"手数料/諸経費等","instrument_type":"=stock"}`}))
				return r
			}
			get := func(path string) string {
				r := do(s, http.MethodGet, path+"?account_id="+account, "", token)
				require.Equal(t, http.StatusOK, r.Code, r.Body.String())
				return r.Body.String()
			}
			type execution struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
			}
			type result struct {
				Inserted int                        `json:"inserted"`
				Enriched int                        `json:"enriched"`
				Rows     []importer.SBIMarginPnLRow `json:"enrichment_rows"`
			}
			seed := upload("/api/v1/imports/commit", string(history))
			require.Equal(t, http.StatusOK, seed.Code, seed.Body.String())
			var seeded result
			require.NoError(t, json.Unmarshal(seed.Body.Bytes(), &seeded))
			require.Equal(t, 8, seeded.Inserted)
			before, tradesBefore := get("/api/v1/executions"), get("/api/v1/trades")
			var executions []execution
			require.NoError(t, json.Unmarshal([]byte(before), &executions))
			var preview result
			for range 2 {
				r := upload("/api/v1/imports", body)
				require.Equal(t, http.StatusOK, r.Code, r.Body.String())
				require.NoError(t, json.Unmarshal(r.Body.Bytes(), &preview))
				require.Len(t, preview.Rows, 4)
				ids := map[string]bool{}
				for _, row := range preview.Rows {
					require.Equal(t, "enrichable", row.Status)
					require.Equal(t, "exact", row.MatchProvenance)
					require.Equal(t, "exact", row.OpeningMatch)
					require.False(t, ids[row.ExecutionID])
					ids[row.ExecutionID] = true
					index := slices.IndexFunc(executions, func(e execution) bool { return e.ID == row.ExecutionID })
					require.NotEqual(t, -1, index)
					require.Equal(t, row.Symbol, executions[index].Symbol)
				}
				require.Equal(t, before, get("/api/v1/executions"))
			}
			// One inconsistent row prevents the other valid enrichments from being written.
			conflicting := strings.Replace(body, ",1200,1960", ",1200,1959", 1)
			require.Equal(t, http.StatusConflict, upload("/api/v1/imports/commit", conflicting).Code)
			require.Equal(t, before, get("/api/v1/executions"))
			r := upload("/api/v1/imports/commit", body)
			require.Equal(t, http.StatusOK, r.Code, r.Body.String())
			var committed result
			require.NoError(t, json.Unmarshal(r.Body.Bytes(), &committed))
			require.Equal(t, 4, committed.Enriched)
			require.Equal(t, preview.Rows, committed.Rows)
			after := get("/api/v1/executions")
			var enriched []execution
			require.NoError(t, json.Unmarshal([]byte(after), &enriched))
			require.Len(t, enriched, len(executions))
			for i, e := range enriched {
				require.Equal(t, executions[i].ID, e.ID)
			}
			require.JSONEq(t, tradesBefore, get("/api/v1/trades"))
			for _, repeat := range []string{string(report), reversed} {
				r = upload("/api/v1/imports/commit", repeat)
				require.Equal(t, http.StatusOK, r.Code, r.Body.String())
				require.NoError(t, json.Unmarshal(r.Body.Bytes(), &committed))
				require.Zero(t, committed.Enriched)
				for _, row := range committed.Rows {
					require.Equal(t, "already_enriched", row.Status)
					index := slices.IndexFunc(preview.Rows, func(prior importer.SBIMarginPnLRow) bool {
						return prior.Symbol == row.Symbol && *prior.PnL == *row.PnL
					})
					require.NotEqual(t, -1, index)
					require.Equal(t, preview.Rows[index].ExecutionID, row.ExecutionID)
					require.Equal(t, preview.Rows[index].Basis, row.Basis)
				}
				require.Equal(t, after, get("/api/v1/executions"))
			}
		})
	}
}
