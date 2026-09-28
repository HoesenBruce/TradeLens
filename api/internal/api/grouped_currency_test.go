package api_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGroupedAndTradeCurrency(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(fmt.Sprint(available), func(t *testing.T) {
			provider := marketdata.NewYahooProvider()
			provider.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
				if !available {
					return nil, fmt.Errorf("unavailable")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[{"timestamp":[1780000000],"indicators":{"quote":[{"close":[150]}]}}]}}`)), Header: http.Header{}}, nil
			})}
			s := testServerWithProvider(t, true, nil, provider)
			token := registerAndLogin(t, s, "grouped-currency@x.com")
			jpy := accountWithCurrency(t, s, token, "JPY", "JPY")
			usd := accountWithCurrency(t, s, token, "USD", "USD")
			seedClosedTrade(t, s, token, jpy, "7203", 100)
			seedClosedTrade(t, s, token, usd, "AAPL", 10)
			r := do(s, http.MethodGet, "/api/v1/analytics/r-summary", "", token)
			require.Equal(t, 200, r.Code, r.Body.String())

			for _, path := range []string{"/analytics/breakdown?by=tag&", "/trades?", "/analytics/behavior?", "/analytics/montecarlo?paths=10&horizon=1&seed=42&"} {
				for _, target := range []string{"JPY", "USD"} {
					rec := do(s, http.MethodGet, "/api/v1"+path+"target_currency="+target, "", token)
					if !available {
						require.Equal(t, 502, rec.Code, rec.Body.String())
						require.Contains(t, rec.Body.String(), "fx_unavailable")
						require.NotContains(t, rec.Body.String(), `"trades"`)
						continue
					}
					require.Equal(t, 200, rec.Code, rec.Body.String())
					var out map[string]any
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
					require.Equal(t, target, out["currency"])
					require.Equal(t, "latest", out["fx_policy"])
					want := 3200.0
					if target == "USD" {
						want /= 150
					}
					if rows, ok := out["trades"].([]any); ok {
						require.Len(t, rows, 2)
						sum := 0.0
						for _, r := range rows {
							row := r.(map[string]any)
							require.Equal(t, target, row["pnl_currency"])
							require.Contains(t, []any{"JPY", "USD"}, row["source_pnl_currency"])
							sum += row["net_pnl"].(float64)
							// Native detail remains untouched despite list display normalization.
							detail := do(s, http.MethodGet, "/api/v1/trades/"+row["id"].(string), "", token)
							require.Equal(t, 200, detail.Code)
							require.Contains(t, detail.Body.String(), fmt.Sprintf(`"pnl_currency":%q`, row["source_pnl_currency"]))
						}
						require.InDelta(t, want, sum, .001)
					}
					if groups, ok := out["groups"].([]any); ok {
						require.Len(t, groups, 1)
						g := groups[0].(map[string]any)
						summary := g["summary"].(map[string]any)
						require.InDelta(t, want, summary["net_pnl"], .01)
						require.EqualValues(t, 2, summary["total_trades"])
					}
				}
				rec := do(s, http.MethodGet, "/api/v1"+path+"account_id="+jpy+"&target_currency=JPY", "", token)
				require.Equal(t, 200, rec.Code, rec.Body.String())
			}
			if available {
				for i := 0; i < 4; i++ {
					seedClosedTrade(t, s, token, jpy, fmt.Sprintf("JP%d", i), 100)
					seedClosedTrade(t, s, token, usd, fmt.Sprintf("US%d", i), 10)
				}
				var outputs []map[string]any
				for _, target := range []string{"JPY", "USD"} {
					rec := do(s, http.MethodGet, "/api/v1/analytics/montecarlo?paths=200&horizon=10&seed=42&target_currency="+target, "", token)
					require.Equal(t, 200, rec.Code, rec.Body.String())
					var out map[string]any
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
					require.Equal(t, false, out["insufficient_data"])
					outputs = append(outputs, out)
				}
				j := outputs[0]["terminal"].(map[string]any)
				u := outputs[1]["terminal"].(map[string]any)
				require.InDelta(t, j["mean"].(float64)/150, u["mean"], .001)
			}

		})
	}
}
