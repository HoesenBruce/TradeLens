package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const sbiCashCSV = "\ufeff\n円貨入出金明細\n\n" +
	"入出金日,取引,区分,摘要,出金額,入金額\n" +
	"2026/01/05,入金,金融機関からの入金,即時入金 匿名銀行,0,100000\n" +
	"2026/01/05,入金,金融機関からの入金,即時入金 匿名銀行,0,100000\n" +
	"2026/01/06,出金,その他,譲渡益税源泉徴収金,300,0\n"

func TestSBICashImportDedupAndRollback(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "sbi-cash@x.com")
	acc := accountID(t, s, tok)

	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "cash.csv", sbiCashCSV,
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview struct {
		Format         string `json:"format"`
		DetectedBroker string `json:"detected_broker"`
		RowCount       int    `json:"row_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Equal(t, "cash_transactions", preview.Format)
	require.Equal(t, "SBI Securities (Cash Transactions)", preview.DetectedBroker)
	require.Equal(t, 3, preview.RowCount)

	commit := func() (string, int, int) {
		t.Helper()
		rec = httptest.NewRecorder()
		s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "cash.csv", sbiCashCSV,
			map[string]string{"account_id": acc}))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result struct {
			CashInserted int `json:"cash_inserted"`
			Skipped      int `json:"skipped"`
			Errors       []struct {
				Row int `json:"row"`
			} `json:"errors"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		require.NotNil(t, result.Errors)
		require.Empty(t, result.Errors)

		rec = do(s, http.MethodGet, "/api/v1/imports", "", tok)
		var batches []struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &batches))
		return batches[0].ID, result.CashInserted, result.Skipped
	}

	batchID, inserted, skipped := commit()
	require.Equal(t, 3, inserted)
	require.Zero(t, skipped)
	_, inserted, skipped = commit()
	require.Zero(t, inserted)
	require.Equal(t, 3, skipped)

	rec = do(s, http.MethodDelete, "/api/v1/imports/"+batchID, "", tok)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = do(s, http.MethodGet, "/api/v1/cash-transactions?account_id="+acc, "", tok)
	var cash []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cash))
	require.Len(t, cash, 1) // account opening deposit remains
}
