package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHoldingListDetailReadback(t *testing.T) {
	for _, precision := range []string{"date", ""} {
		name := precision
		if name == "" {
			name = "precise"
		}
		t.Run(name, func(t *testing.T) {
			s := testServer(t)
			token := registerAndLogin(t, s, "holding@synthetic.invalid")
			account := accountID(t, s, token)
			for i, side := range []string{"buy", "sell"} {
				body := fmt.Sprintf(`{"account_id":%q,"symbol":"6920","instrument_type":"stock","side":%q,"quantity":100,"price":1000,"executed_at":"2026-10-%02dT09:00:00+09:00","details":{"source_time_precision":%q}}`, account, side, 13+i, precision)
				rec := do(s, http.MethodPost, "/api/v1/executions", body, token)
				require.Equal(t, 201, rec.Code, rec.Body.String())
			}
			rec := do(s, http.MethodGet, "/api/v1/trades?account_id="+account, "", token)
			require.Equal(t, 200, rec.Code)
			var rows []map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rows))
			require.Len(t, rows, 1)
			h := rows[0]["holding"].(map[string]any)
			if precision == "date" {
				require.Equal(t, "calendar", h["kind"])
				require.Equal(t, float64(1), h["calendar_days"])
				require.Nil(t, h["seconds"])
			} else {
				require.Equal(t, "elapsed", h["kind"])
				require.Equal(t, float64(86400), h["seconds"])
			}
			detail := do(s, http.MethodGet, "/api/v1/trades/"+rows[0]["id"].(string), "", token)
			require.Equal(t, 200, detail.Code)
			var saved map[string]any
			require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &saved))
			require.Equal(t, h, saved["holding"])
			require.Equal(t, rows[0]["net_pnl"], saved["net_pnl"])
			require.Equal(t, rows[0]["fees_total"], saved["fees_total"])
		})
	}
}
