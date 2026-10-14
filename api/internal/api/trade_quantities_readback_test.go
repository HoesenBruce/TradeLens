package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuantityListAndDetailUseSameInventory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sides []string
		qty   []float64
		want  [][3]float64
	}{
		{"partial", []string{"buy", "sell"}, []float64{500, 400}, [][3]float64{{500, 400, 100}}},
		{"pyramid", []string{"buy", "sell", "buy"}, []float64{500, 400, 200}, [][3]float64{{700, 400, 300}}},
		{"closed", []string{"buy", "sell"}, []float64{500, 500}, [][3]float64{{500, 500, 0}}},
		{"short", []string{"sell", "buy"}, []float64{70, 60}, [][3]float64{{70, 60, 10}}},
		{"reversal", []string{"buy", "sell"}, []float64{500, 700}, [][3]float64{{500, 500, 0}, {200, 0, 200}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testServer(t)
			token := registerAndLogin(t, s, tc.name+"@quantity.invalid")
			account := accountID(t, s, token)
			for i, side := range tc.sides {
				body := fmt.Sprintf(`{"account_id":%q,"symbol":"TEST","instrument_type":"stock","side":%q,"quantity":%g,"price":100,"fees":1,"executed_at":"2026-09-01T%02d:00:00Z"}`, account, side, tc.qty[i], i+1)
				rec := do(s, http.MethodPost, "/api/v1/executions", body, token)
				require.Equal(t, 201, rec.Code, rec.Body.String())
			}
			rec := do(s, http.MethodGet, "/api/v1/trades?account_id="+account, "", token)
			require.Equal(t, 200, rec.Code)
			var rows []map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rows))
			require.Len(t, rows, len(tc.want))
			seen := map[[3]float64]bool{}
			for _, row := range rows {
				q := row["quantities"].(map[string]any)
				seen[[3]float64{q["entry_qty"].(float64), q["exit_qty"].(float64), q["open_qty"].(float64)}] = true
				require.Equal(t, row["qty_opened"], q["entry_qty"])
				require.Equal(t, row["qty_remaining"], q["open_qty"])
				for range 2 {
					detail := do(s, http.MethodGet, "/api/v1/trades/"+row["id"].(string), "", token)
					require.Equal(t, 200, detail.Code)
					var saved map[string]any
					require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &saved))
					require.Equal(t, q, saved["quantities"])
					require.Equal(t, row["net_pnl"], saved["net_pnl"])
					require.Equal(t, row["fees_total"], saved["fees_total"])
				}
			}
			for _, want := range tc.want {
				require.True(t, seen[want], "missing quantity tuple %v", want)
			}
		})
	}
}
