package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// A minimal MT5 Trade History Report: one balance deal and one EURUSD
// round-trip. Detection is by content sniff, so the filename is irrelevant.
const mt5StatementHTML = `<html><head><title>Trade History Report</title></head><body>
<table>
<tr><td colspan="14"><b>Deals</b></td></tr>
<tr><th>Time</th><th>Deal</th><th>Symbol</th><th>Type</th><th>Direction</th><th>Volume</th><th>Price</th><th>Order</th><th>Commission</th><th>Fee</th><th>Swap</th><th>Profit</th><th>Balance</th><th>Comment</th></tr>
<tr><td>2024.01.10 09:00:00</td><td>400100</td><td></td><td>balance</td><td></td><td></td><td></td><td></td><td>0.00</td><td>0.00</td><td>0.00</td><td>10000.00</td><td>10000.00</td><td>Deposit</td></tr>
<tr><td>2024.01.15 10:30:00</td><td>400101</td><td>EURUSD</td><td>buy</td><td>in</td><td>0.50</td><td>1.09312</td><td>300101</td><td>-1.75</td><td>0.00</td><td>0.00</td><td>0.00</td><td>9998.25</td><td></td></tr>
<tr><td>2024.01.15 14:45:30</td><td>400102</td><td>EURUSD</td><td>sell</td><td>out</td><td>0.50</td><td>1.09501</td><td>300102</td><td>-1.75</td><td>0.00</td><td>-1.20</td><td>94.50</td><td>10089.80</td><td></td></tr>
<tr><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td>-3.50</td><td>0.00</td><td>-1.20</td><td>94.50</td><td>10089.80</td><td></td></tr>
</table>
</body></html>`

func TestMTStatementImportEndToEnd(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "mt5@x.com")
	acc := accountID(t, s, tok)

	// Preview: recognized as a statement — no column mapping, MetaTrader
	// server-time (EET) suggested instead of UTC.
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "report.html", mt5StatementHTML,
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview struct {
		Source            string            `json:"source"`
		Format            string            `json:"format"`
		DetectedBroker    string            `json:"detected_broker"`
		SuggestedSourceTZ string            `json:"suggested_source_tz"`
		SuggestedMapping  map[string]string `json:"suggested_mapping"`
		RowCount          int               `json:"row_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Equal(t, "statement", preview.Source)
	require.Equal(t, "executions", preview.Format)
	require.Equal(t, "MetaTrader 5 (Trade History Report)", preview.DetectedBroker)
	require.Equal(t, "Europe/Athens", preview.SuggestedSourceTZ)
	require.Empty(t, preview.SuggestedMapping)
	require.Equal(t, 3, preview.RowCount)

	// Commit needs no column_mapping for statements.
	rec = httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "report.html", mt5StatementHTML,
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var res struct {
		Inserted int `json:"inserted"`
		Skipped  int `json:"skipped"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, 2, res.Inserted) // balance deal is not a fill

	// The pair regroups into one closed EURUSD trade with EET times → UTC.
	rec = do(s, http.MethodGet, "/api/v1/trades?account_id="+acc+"&status=closed", "", tok)
	require.Equal(t, http.StatusOK, rec.Code)
	var trades []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &trades))
	require.Len(t, trades, 1)
	require.Equal(t, "EURUSD", trades[0]["symbol"])
	require.Equal(t, "2024-01-15T08:30:00Z", trades[0]["opened_at"])

	// Re-import dedups every fill.
	rec = httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "report.html", mt5StatementHTML,
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, 0, res.Inserted)
	require.Equal(t, 2, res.Skipped)
}

func TestMTStatementSourceTZOverride(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "mt5tz@x.com")
	acc := accountID(t, s, tok)

	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "report.html", mt5StatementHTML,
		map[string]string{"account_id": acc, "source_tz": "UTC"}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = do(s, http.MethodGet, "/api/v1/trades?account_id="+acc+"&status=closed", "", tok)
	var trades []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &trades))
	require.Len(t, trades, 1)
	require.Equal(t, "2024-01-15T10:30:00Z", trades[0]["opened_at"])
}

func TestUnrecognizedStatementUploadFails(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "mt5bad@x.com")
	acc := accountID(t, s, tok)

	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "page.html",
		"<html><body><table><tr><td>not a statement</td></tr></table></body></html>",
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "MetaTrader")
}

func TestSBIImportEndToEnd(t *testing.T) {
	fixture, err := os.ReadFile("../importer/testdata/sbi-trade-executions.csv")
	require.NoError(t, err)
	cp932, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), fixture)
	require.NoError(t, err)

	s := testServer(t)
	tok := registerAndLogin(t, s, "sbi@x.com")
	acc := accountID(t, s, tok)

	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "sbi.csv", string(cp932),
		map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview struct {
		DetectedBroker    string            `json:"detected_broker"`
		SuggestedSourceTZ string            `json:"suggested_source_tz"`
		SuggestedMapping  map[string]string `json:"suggested_mapping"`
		RowCount          int               `json:"row_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Equal(t, "SBI Securities (Execution History)", preview.DetectedBroker)
	require.Equal(t, "Asia/Tokyo", preview.SuggestedSourceTZ)
	require.Equal(t, "銘柄コード", preview.SuggestedMapping["symbol"])
	require.Equal(t, "銘柄", preview.SuggestedMapping["stock_name"])
	require.Equal(t, 11, preview.RowCount)

	fields := map[string]string{
		"account_id":     acc,
		"column_mapping": `{"symbol":"銘柄コード","stock_name":"銘柄","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","fees":"手数料/諸経費等"}`,
	}
	commit := func() struct {
		Inserted int `json:"inserted"`
		Skipped  int `json:"skipped"`
		Errors   []struct {
			Row int `json:"row"`
		} `json:"errors"`
	} {
		rec := httptest.NewRecorder()
		s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "sbi.csv", string(cp932), fields))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var result struct {
			Inserted int `json:"inserted"`
			Skipped  int `json:"skipped"`
			Errors   []struct {
				Row int `json:"row"`
			} `json:"errors"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		return result
	}

	first := commit()
	require.Equal(t, 11, first.Inserted)
	require.Equal(t, 0, first.Skipped)
	require.Len(t, first.Errors, 1)
	second := commit()
	require.Equal(t, 0, second.Inserted)
	require.Equal(t, 11, second.Skipped)

	rec = do(s, http.MethodGet, "/api/v1/trades?account_id="+acc, "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var trades []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &trades))
	converted := []map[string]any{}
	for _, trade := range trades {
		if trade["symbol"] == "9102" {
			converted = append(converted, trade)
		}
	}
	require.Len(t, converted, 2)
	require.ElementsMatch(t, []any{"closed", "open"}, []any{converted[0]["status"], converted[1]["status"]})
	for _, trade := range trades {
		if trade["symbol"] == "9101" {
			require.Equal(t, "合成甲", trade["stock_name"])
			return
		}
	}
	require.Fail(t, "imported SBI trade not found")
}
