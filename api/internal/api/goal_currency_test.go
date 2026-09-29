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

func TestAnnualGoalCurrency(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(fmt.Sprint(available), func(t *testing.T) {
			p := marketdata.NewYahooProvider()
			p.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
				if !available {
					return nil, fmt.Errorf("unavailable")
				}
				rate := 150.
				if strings.Contains(r.URL.Path, "JPYUSD") {
					rate = 1. / 150
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"chart":{"result":[{"timestamp":[1780000000],"indicators":{"quote":[{"close":[%g]}]}}]}}`, rate))), Header: http.Header{}}, nil
			})}
			s := testServerWithProvider(t, true, nil, p)
			token := registerAndLogin(t, s, "goal-currency@x.com")
			rec := do(s, http.MethodPut, "/api/v1/settings/annual-goal", `{"year":2026,"amount":15000,"currency":"JPY"}`, token)
			require.Equal(t, 200, rec.Code, rec.Body.String())
			for _, target := range []string{"JPY", "USD"} {
				rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2026&target_currency="+target, "", token)
				if !available && target == "USD" {
					require.Equal(t, 502, rec.Code)
					require.Contains(t, rec.Body.String(), "fx_unavailable")
					continue
				}
				require.Equal(t, 200, rec.Code, rec.Body.String())
				var out map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
				require.Equal(t, target, out["currency"])
				require.Equal(t, "JPY", out["source_currency"])
				require.Equal(t, "latest", out["fx_policy"])
				want := 15000.
				if target == "USD" {
					want = 100
				}
				require.InDelta(t, want, out["amount"], .001)
			}
			// Reads never mutate the original goal or silently assign legacy currency.
			rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2026", "", token)
			require.Contains(t, rec.Body.String(), `"amount":15000`)
			require.Contains(t, rec.Body.String(), `"currency":"JPY"`)
			rec = do(s, http.MethodPut, "/api/v1/settings/annual-goal", `{"year":2025,"amount":500}`, token)
			require.Equal(t, 200, rec.Code)
			rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2025&target_currency=JPY", "", token)
			require.Equal(t, 400, rec.Code)
			require.Contains(t, rec.Body.String(), "currency_required")
			rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2025", "", token)
			require.Contains(t, rec.Body.String(), `"amount":500`)
			rec = do(s, http.MethodPut, "/api/v1/settings/annual-goal", `{"year":2026,"amount":100,"currency":"YEN!"}`, token)
			require.Equal(t, 400, rec.Code)
			rec = do(s, http.MethodDelete, "/api/v1/settings/annual-goal?year=2026", "", token)
			require.Equal(t, 200, rec.Code)
			rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2026&target_currency=JPY", "", token)
			require.Equal(t, 200, rec.Code)
			require.Contains(t, rec.Body.String(), `"amount":null`)
			require.Contains(t, rec.Body.String(), `"currency":"JPY"`)
			rec = do(s, http.MethodGet, "/api/v1/settings/annual-goal?year=2026&target_currency=INVALID", "", token)
			require.Equal(t, 400, rec.Code)
		})
	}
}
