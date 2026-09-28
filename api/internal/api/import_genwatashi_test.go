package api_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenwatashiImportAccounting(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		cash, short, delivered, fee                float64
		extra, receipt, pnl, basis                 string
		want, remainingCash, remainingShort, costs float64
		source                                     string
	}{
		{"simple", 100, 100, 100, 0, "", "--", "--", "--", 10000, 0, 0, 0, "calculated_sbi_cash_basis"},
		{"costs", 100, 100, 100, 600, "", "99400", "--", "--", 9400, 0, 0, 600, "calculated_sbi_cash_basis"},
		{"partial", 200, 200, 100, 0, "", "--", "--", "--", 10000, 100, 100, 0, "calculated_sbi_cash_basis"},
		{"cash excess", 200, 100, 100, 0, "", "--", "--", "--", 10000, 100, 0, 0, "calculated_sbi_cash_basis"},
		{"short excess", 100, 200, 100, 0, "", "--", "--", "--", 10000, 0, 100, 0, "calculated_sbi_cash_basis"},
		{"cash average", 100, 100, 100, 0, "2026/08/02,1515,現物買,100,1100,0,--,--,--\n", "--", "--", "--", 0, 100, 0, 0, "calculated_sbi_cash_basis"},
		{"reported", 100, 100, 100, 600, "", "99400", "9100", "903", 9100, 0, 0, 600, "broker_reported"},
		{"receipt overrides costs", 100, 100, 100, 600, "", "99600", "--", "--", 9600, 0, 0, 400, "calculated_sbi_cash_basis"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testServer(t)
			tok := registerAndLogin(t, s, "genwatashi@x.com")
			acc := accountID(t, s, tok)
			csv := "約定履歴照会\n約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等,受渡金額/決済損益,実現損益,平均取得価額\n" + fmt.Sprintf("2026/08/01,1515,現物買,%g,900,0,--,--,--\n2026/08/02,1515,信用新規売,%g,1000,0,--,--,--\n", tc.cash, tc.short) + tc.extra + fmt.Sprintf("2026/08/04,1515,現渡,%g,1000,%g,%s,%s,%s\n", tc.delivered, tc.fee, tc.receipt, tc.pnl, tc.basis)
			for attempt := 0; attempt < 2; attempt++ {
				rec := httptest.NewRecorder()
				s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "sbi.csv", csv, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード"}`}))
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			}
			rec := do(s, http.MethodGet, "/api/v1/trades?account_id="+acc, "", tok)
			var trs []struct {
				ID        string   `json:"id"`
				Direction string   `json:"direction"`
				Qty       float64  `json:"qty_remaining"`
				Pnl       *float64 `json:"net_pnl"`
				Fees      float64  `json:"fees_total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &trs))
			require.Len(t, trs, 2)
			if tc.remainingCash == 0 && tc.remainingShort == 0 {
				daily := do(s, http.MethodGet, "/api/v1/analytics/daily?account_id="+acc+"&date_basis=close", "", tok)
				var days map[string]float64
				require.NoError(t, json.Unmarshal(daily.Body.Bytes(), &days))
				sum := 0.0
				for _, pnl := range days {
					sum += pnl
				}
				require.Equal(t, tc.want, sum)
			}

			for _, tr := range trs {
				detail := do(s, http.MethodGet, "/api/v1/trades/"+tr.ID, "", tok)
				require.Contains(t, detail.Body.String(), `"settlement_type":"genwatashi"`)
				require.Contains(t, detail.Body.String(), `"event_type":"position_settlement"`)
				require.Contains(t, detail.Body.String(), `"realized_pnl_source":"`+tc.source+`"`)
				require.NotNil(t, tr.Pnl)
				if tr.Direction == "long" {
					require.Equal(t, tc.remainingCash, tr.Qty)
					require.Equal(t, tc.want, *tr.Pnl)
					require.Equal(t, tc.costs, tr.Fees)
				} else {
					require.Equal(t, tc.remainingShort, tr.Qty)
					require.Zero(t, *tr.Pnl)
					require.Zero(t, tr.Fees)
				}
			}
		})
	}
}

func TestGenwatashiInsufficientHistoryRollsBack(t *testing.T) {
	for i, rows := range []string{
		"2026/08/04,1515,現渡,100,1000,0\n",
		"2026/08/01,1515,現物買,50,900,0\n2026/08/02,1515,信用新規売,100,1000,0\n2026/08/04,1515,現渡,100,1000,0\n",
		"2026/08/01,1515,現物買,100,900,0\n2026/08/02,1515,信用新規売,100,1000,0\n2026/08/03,1515,信用新規売,100,1200,0\n2026/08/04,1515,現渡,100,1000,0\n",
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := testServer(t)
			tok := registerAndLogin(t, s, "missing-genwatashi@x.com")
			acc := accountID(t, s, tok)
			rec := httptest.NewRecorder()
			s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "sbi.csv", "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n"+rows, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"銘柄コード"}`}))
			require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
			fills := do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok)
			require.JSONEq(t, `[]`, fills.Body.String())
		})
	}
}
