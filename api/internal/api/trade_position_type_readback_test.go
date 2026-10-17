package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPositionTypeListDetailReadback(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "positiontype@synthetic.invalid")
	account := accountID(t, s, token)
	for i, tc := range []struct{ lot, want, side string }{
		{"sbi:cash", "cash", "buy"},
		{"sbi:margin-long", "margin", "buy"},
		{"sbi:margin-short", "margin", "sell"},
		{"", "unknown", "buy"},
		{"ibkr:cash", "unknown", "buy"},
	} {
		// Cash and margin share a symbol and account, but remain separate groups.
		for j := range 2 {
			side := tc.side
			if j == 1 {
				if side == "buy" {
					side = "sell"
				} else {
					side = "buy"
				}
			}
			body := fmt.Sprintf(`{"account_id":%q,"symbol":"SYNTH","instrument_type":"stock","side":%q,"quantity":10,"price":100,"executed_at":"2026-10-%02dT09:00:00Z","details":{"lot":%q}}`, account, side, 1+i*2+j, tc.lot)
			rec := do(s, http.MethodPost, "/api/v1/executions", body, token)
			require.Equal(t, 201, rec.Code, rec.Body.String())
			rec = do(s, http.MethodGet, "/api/v1/trades?account_id="+account, "", token)
			require.Equal(t, 200, rec.Code, rec.Body.String())
			var rows []map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rows))
			require.Len(t, rows, i+1)
			var row map[string]any
			for _, r := range rows {
				if r["status"] == "open" || r["opened_at"] == fmt.Sprintf("2026-10-%02dT09:00:00Z", 1+i*2) {
					row = r
					break
				}
			}
			require.NotNil(t, row)
			require.Equal(t, tc.want, row["position_type"])
			detail := do(s, http.MethodGet, "/api/v1/trades/"+row["id"].(string), "", token)
			require.Equal(t, 200, detail.Code, detail.Body.String())
			var saved map[string]any
			require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &saved))
			for _, key := range []string{"position_type", "direction", "qty_remaining", "avg_entry_price", "net_pnl", "fees_total"} {
				require.Equal(t, row[key], saved[key], key)
			}
			if j == 0 {
				require.Equal(t, "open", row["status"])
			} else {
				require.Equal(t, "closed", row["status"])
				require.Equal(t, float64(0), row["net_pnl"])
			}
		}
	}
}
