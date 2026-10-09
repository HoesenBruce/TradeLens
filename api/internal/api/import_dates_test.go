package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportDatePreviewMatchesCommit(t *testing.T) {
	for _, tc := range []struct{ name, stamp, second, order, zone, want string }{
		{"day-first", "05/01/2026 09:30:00", "13/01/2026 10:00:00", "", "UTC", "2026-01-05T09:30:00Z"},
		{"month-first", "05/01/2026 09:30:00", "01/13/2026 10:00:00", "", "UTC", "2026-05-01T09:30:00Z"},
		{"ambiguous-day", "05/01/2026 09:30:00", "", "day_first", "Asia/Tokyo", "2026-01-05T00:30:00Z"},
		{"ambiguous-month", "05/01/2026 09:30:00", "", "month_first", "UTC", "2026-05-01T09:30:00Z"},
		{"12h", "13/01/2026 3:04 PM", "", "", "UTC", "2026-01-13T15:04:00Z"},
		{"offset", "2026-01-05T09:30:00+08:00", "", "", "America/New_York", "2026-01-05T01:30:00Z"},
		{"dst", "2026-07-10 09:30:00", "", "", "America/New_York", "2026-07-10T13:30:00Z"},
		{"winter", "2026-01-05 09:30:00", "", "", "America/New_York", "2026-01-05T14:30:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testServer(t)
			tok := registerAndLogin(t, s, tc.name+"@x.com")
			acc := accountID(t, s, tok)
			csv := "Symbol,Side,Qty,Price,Time\nAAPL,Buy,100,10," + tc.stamp + "\n"
			if tc.second != "" {
				csv += "AAPL,Buy,100,10," + tc.second + "\n"
			}
			fields := map[string]string{"account_id": acc, "column_mapping": `{"symbol":"Symbol","side":"Side","quantity":"Qty","price":"Price","executed_at":"Time"}`, "source_tz": tc.zone, "date_order": tc.order}
			rec := httptest.NewRecorder()
			s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "dates.csv", csv, fields))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var preview struct {
				Executions []struct {
					At string `json:"executed_at"`
				} `json:"parsed_executions"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
			require.NotEmpty(t, preview.Executions)
			require.Equal(t, tc.want, preview.Executions[0].At)
			// Cancelling a preview leaves no batch and no executions.
			require.JSONEq(t, `[]`, do(s, http.MethodGet, "/api/v1/imports", "", tok).Body.String())
			require.JSONEq(t, `[]`, do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok).Body.String())
			rec = httptest.NewRecorder()
			s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "dates.csv", csv, fields))
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			rec = do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok)
			var fills []struct {
				At string `json:"executed_at"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fills))
			found := false
			for _, fill := range fills {
				if fill.At == tc.want {
					found = true
				}
			}
			require.True(t, found)
		})
	}
}

func TestAmbiguousDateChoiceRequiredBeforeWrites(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "ambiguous@x.com")
	acc := accountID(t, s, tok)
	csv := "Symbol,Side,Qty,Price,Time\nAAPL,Buy,1,10,05/01/2026 09:30:00\n"
	fields := map[string]string{"account_id": acc, "column_mapping": `{"symbol":"Symbol","side":"Side","quantity":"Qty","price":"Price","executed_at":"Time"}`}
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "dates.csv", csv, fields))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview struct {
		Order struct {
			Order, Example string
			Day            string `json:"example_day_first"`
			Month          string `json:"example_month_first"`
		} `json:"date_order"`
		Executions []any `json:"parsed_executions"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.Equal(t, "ambiguous", preview.Order.Order)
	require.Equal(t, "2026-01-05", preview.Order.Day)
	require.Equal(t, "2026-05-01", preview.Order.Month)
	require.Empty(t, preview.Executions)
	for _, order := range []string{"", "sideways"} {
		fields["date_order"] = order
		rec = httptest.NewRecorder()
		s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "dates.csv", csv, fields))
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	}
	require.JSONEq(t, `[]`, do(s, http.MethodGet, "/api/v1/imports", "", tok).Body.String())
}

func TestSBIYearFirstPreviewHasSyntheticTimesWithoutDatePrompt(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "sbi-date@x.com")
	acc := accountID(t, s, tok)
	csv := "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/09/01,7203,現物買,100,1000,0\n"
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports", tok, "sbi.csv", csv, map[string]string{"account_id": acc}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var preview map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	require.NotContains(t, preview, "date_order")
	var fills []struct {
		At        string `json:"executed_at"`
		Precision string `json:"source_time_precision"`
	}
	require.NoError(t, json.Unmarshal(preview["parsed_executions"], &fills))
	require.Len(t, fills, 1)
	require.Equal(t, "date", fills[0].Precision)
	require.Contains(t, fills[0].At, "2026-09-01")
}
