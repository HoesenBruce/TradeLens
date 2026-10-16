package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSBIRealizedMoneyPartialFinalFiltersAndDefaultIsolation(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "synthetic-realized@test.invalid")
	account := func(name string) string {
		r := do(s, http.MethodPost, "/api/v1/accounts", fmt.Sprintf(`{"name":%q,"broker":"SBI","account_type":"margin","base_currency":"JPY"}`, name), tok)
		require.Equal(t, 201, r.Code, r.Body.String())
		var a struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &a))
		return a.ID
	}
	sbi, other := account("Synthetic source SBI"), account("SBI label with generic executions")
	upload := func(body string) {
		r := httptest.NewRecorder()
		s.Echo.ServeHTTP(r, multipartFileReq(t, "/api/v1/imports/commit", tok, "synthetic.csv", body, map[string]string{"account_id": sbi, "column_mapping": `{"symbol":"銘柄コード","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","fees":"手数料/諸経費等","instrument_type":"=stock"}`}))
		require.Equal(t, 200, r.Code, r.Body.String())
	}
	history := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/10/10,1515,現物買,100,100,0\n2026/10/11,1515,現物売,50,96,0\n"
	upload(history)
	upload("実現損益詳細\n約定日,銘柄コード,取引,数量,単価,平均取得価額,実現損益\n2026/10/11,1515,売却,50,96,100,-200\n")
	manual := func(side string, qty, price float64, day string) {
		r := do(s, http.MethodPost, "/api/v1/executions", fmt.Sprintf(`{"account_id":%q,"symbol":"GENERIC","instrument_type":"stock","side":%q,"quantity":%v,"price":%v,"executed_at":%q,"multiplier":1}`, other, side, qty, price, day+"T01:00:00Z"), tok)
		require.Equal(t, 201, r.Code, r.Body.String())
	}
	manual("buy", 100, 100, "2026-10-10")
	manual("sell", 50, 98, "2026-10-11")
	for _, c := range []struct {
		kind   string
		amount float64
	}{{"deposit", 1000}, {"dividend", 10}, {"fee", -3}} {
		r := do(s, http.MethodPost, "/api/v1/cash-transactions", fmt.Sprintf(`{"account_id":%q,"type":%q,"amount":%v,"currency":"JPY","occurred_at":"2026-10-09T00:00:00Z"}`, sbi, c.kind, c.amount), tok)
		require.Equal(t, 201, r.Code, r.Body.String())
	}
	summary := func(query string, net, review float64, count int) {
		r := do(s, http.MethodGet, "/api/v1/analytics/summary?"+query, "", tok)
		require.Equal(t, 200, r.Code, r.Body.String())
		var v struct {
			Realized float64 `json:"realized_pnl"`
			Net      float64 `json:"net_pnl"`
			Count    int     `json:"total_trades"`
		}
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &v))
		require.Equal(t, net, v.Realized, query)
		require.Equal(t, review, v.Net, query)
		require.Equal(t, count, v.Count, query)
	}
	beforeFills := do(s, http.MethodGet, "/api/v1/executions?account_id="+sbi, "", tok).Body.String()
	beforeTrades := do(s, http.MethodGet, "/api/v1/trades?account_id="+sbi, "", tok).Body.String()
	summary("account_id="+sbi, -200, 0, 0)
	summary("account_id="+other, 0, 0, 0)
	summary("account_id="+sbi+","+other, -200, 0, 0)
	r := do(s, http.MethodGet, "/api/v1/analytics/equity-curve?account_id="+sbi, "", tok)
	require.Equal(t, 200, r.Code, r.Body.String())
	var curve struct {
		Points []struct {
			Equity float64 `json:"equity"`
		} `json:"points"`
	}
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &curve))
	require.Equal(t, 807.0, curve.Points[len(curve.Points)-1].Equity)
	require.Equal(t, beforeFills, do(s, http.MethodGet, "/api/v1/executions?account_id="+sbi, "", tok).Body.String())
	require.Equal(t, beforeTrades, do(s, http.MethodGet, "/api/v1/trades?account_id="+sbi, "", tok).Body.String())
	upload(history + "2026/10/12,1515,現物売,50,101,0\n")
	summary("account_id="+sbi, -150, -150, 1)
	period := func(from, to string) string {
		return "account_id=" + sbi + "&from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to)
	}
	summary(period("2026-10-11T00:00:00Z", "2026-10-11T23:59:59Z"), -200, 0, 0)
	summary(period("2026-10-12T00:00:00Z", "2026-10-12T23:59:59Z"), 50, -150, 1)
	summary(period("2026-10-11T04:00:00Z", "2026-10-12T03:59:59Z")+"&tz=America%2FNew_York", -200, -150, 1)
	summary(period("2026-10-13T00:00:00Z", "2026-10-14T23:59:59Z"), 0, 0, 0)
	summary(period("2026-10-11T00:00:00Z", "2026-10-12T23:59:59Z")+"&date_basis=open", -150, 0, 0)
	manual("sell", 50, 104, "2026-10-12")
	summary("account_id="+other, 100, 100, 1)
	summary("account_id="+sbi+","+other, -50, -50, 2)
	summary("account_id="+sbi+"&symbol=UNKNOWN", 0, 0, 0)
	r = do(s, http.MethodGet, "/api/v1/analytics/equity-curve?account_id="+sbi, "", tok)
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &curve))
	require.Equal(t, 857.0, curve.Points[len(curve.Points)-1].Equity)
}

func TestSBIRealizedMarginClosesAndConversionZero(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(fmt.Sprint(short), func(t *testing.T) {
			s := testServer(t)
			tok := registerAndLogin(t, s, "synthetic-margin@test.invalid")
			acc := accountID(t, s, tok)
			opening, closing := "信用新規買", "信用返済売"
			if short {
				opening, closing = "信用新規売", "信用返済買"
			}
			csv := fmt.Sprintf("約定日,銘柄コード,取引,約定数量,約定単価,受渡金額/決済損益\n2026/10/01,1515,%s,100,100,--\n2026/10/02,1515,%s,25,110,-200\n", opening, closing)
			if !short {
				csv += "2026/10/03,1515,現引,25,100,--\n"
			}
			r := httptest.NewRecorder()
			s.Echo.ServeHTTP(r, multipartFileReq(t, "/api/v1/imports/commit", tok, "synthetic.csv", csv, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード","side":"取引","quantity":"約定数量","price":"約定単価","executed_at":"約定日","instrument_type":"=stock"}`}))
			require.Equal(t, 200, r.Code, r.Body.String())
			r = do(s, http.MethodGet, "/api/v1/analytics/summary?account_id="+acc, "", tok)
			require.Equal(t, 200, r.Code, r.Body.String())
			var result struct {
				Money   float64 `json:"realized_pnl"`
				Count   int     `json:"realized_event_count"`
				Reviews int     `json:"total_trades"`
			}
			require.NoError(t, json.Unmarshal(r.Body.Bytes(), &result))
			require.Equal(t, -200.0, result.Money)
			require.Equal(t, 1, result.Count)
			require.Zero(t, result.Reviews)
		})
	}
}
