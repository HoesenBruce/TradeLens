package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/marketdata"
)

type summaryFXTransport func(*http.Request) (*http.Response, error)

func (f summaryFXTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSummaryMixedCurrencyNormalization(t *testing.T) {
	for _, quote := range []string{"150", "0", "null", "1e309", "network-error"} {
		t.Run(quote, func(t *testing.T) {
			calls := 0
			provider := marketdata.NewYahooProvider()
			provider.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if quote == "network-error" {
					return nil, fmt.Errorf("FX unavailable")
				}
				body := fmt.Sprintf(`{"chart":{"result":[{"timestamp":[1780000000],"indicators":{"quote":[{"close":[%s]}]}}]}}`, quote)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
			})}
			s := testServerWithProvider(t, true, nil, provider)
			token := registerAndLogin(t, s, "mixed-summary@x.com")
			jpy := accountWithCurrency(t, s, token, "JPY", "JPY")
			usd := accountWithCurrency(t, s, token, "USD", "USD")
			backtestAccountID(t, s, token)
			// JPY: +200, -100; USD: +20. Different counts and magnitudes
			// distinguish normalized samples from account-summary composition.
			seedClosedTrade(t, s, token, jpy, "7203", 100)
			seedClosedTrade(t, s, token, usd, "AAPL", 10)
			for _, leg := range []struct {
				side  string
				price int
			}{{"buy", 12}, {"sell", 10}} {
				body := fmt.Sprintf(`{"account_id":%q,"symbol":"6758","instrument_type":"stock","side":%q,"quantity":50,"price":%d,"fees":5,"executed_at":"2026-01-06T1%d:00:00Z"}`, jpy, leg.side, leg.price, map[string]int{"buy": 0, "sell": 1}[leg.side])
				rec := do(s, http.MethodPost, "/api/v1/executions", body, token)
				require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			}
			if quote != "150" {
				rec := do(s, http.MethodGet, "/api/v1/analytics/summary?target_currency=JPY", "", token)
				require.Equal(t, http.StatusBadGateway, rec.Code, rec.Body.String())
				require.Contains(t, rec.Body.String(), "fx_unavailable")
				require.NotContains(t, rec.Body.String(), "net_pnl")
				return
			}
			for _, target := range []string{"JPY", "USD", "JPY"} {
				out := summaryFor(t, s, token, "?target_currency="+target)
				require.Equal(t, target, out["currency"])
				require.Equal(t, target, out["target_currency"])
				require.Equal(t, "latest", out["fx_policy"])
				jpyRate, usdRate := 1.0, 150.0
				if target == "USD" {
					jpyRate, usdRate = 1.0/150, 1
				}
				want := analytics.Summarize([]analytics.ClosedTrade{
					{NetPnl: 200 * jpyRate, GrossPnl: 200 * jpyRate},
					{NetPnl: -110 * jpyRate, GrossPnl: -100 * jpyRate, FeesTotal: 10 * jpyRate},
					{NetPnl: 20 * usdRate, GrossPnl: 20 * usdRate},
				})
				payload, err := json.Marshal(want)
				require.NoError(t, err)
				var metrics map[string]any
				require.NoError(t, json.Unmarshal(payload, &metrics))
				for key, value := range metrics {
					require.Equal(t, value, out[key], key)
				}
				require.Len(t, out["fx_rates"], 1)
			}
			pair := summaryFor(t, s, token, "?account_id="+jpy+","+usd+"&target_currency=JPY")
			require.EqualValues(t, 3090, pair["net_pnl"])
			require.Equal(t, 1, calls, "one FX lookup per currency, with inverse cached")
			out := summaryFor(t, s, token, "?account_id="+jpy+"&target_currency=JPY")
			require.EqualValues(t, 90, out["net_pnl"])
			require.Empty(t, out["fx_rates"])
			for _, target := range []string{"JP", "JPY1"} {
				rec := do(s, http.MethodGet, "/api/v1/analytics/summary?target_currency="+target, "", token)
				require.Equal(t, http.StatusBadRequest, rec.Code)
			}
			rec := do(s, http.MethodGet, "/api/v1/analytics/summary?account_id=missing&target_currency=JPY", "", token)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "unknown_currency")
		})
	}
}
