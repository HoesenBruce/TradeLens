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

func TestDailyEquityCurrency(t *testing.T) {
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
			token := registerAndLogin(t, s, "portfolio-currency@x.com")
			jpy := accountWithCurrency(t, s, token, "JPY", "JPY")
			usd := accountWithCurrency(t, s, token, "USD", "USD")
			empty := accountWithCurrency(t, s, token, "empty", "JPY")
			backtestAccountID(t, s, token)
			seedClosedTrade(t, s, token, jpy, "7203", 100)
			seedClosedTrade(t, s, token, usd, "AAPL", 10)
			for _, input := range []struct {
				account, currency string
				amount            int
			}{{jpy, "JPY", 1000}, {usd, "USD", 100}} {
				rec := do(s, http.MethodPost, "/api/v1/cash-transactions", fmt.Sprintf(`{"account_id":%q,"type":"deposit","amount":%d,"currency":%q,"occurred_at":"2026-01-01T00:00:00Z"}`, input.account, input.amount, input.currency), token)
				require.Equal(t, 201, rec.Code, rec.Body.String())
			}
			for _, endpoint := range []string{"daily", "equity-curve"} {
				for _, target := range []string{"JPY", "USD"} {
					for _, scope := range []string{"", jpy + "," + usd, jpy, empty} {
						rec := do(s, http.MethodGet, "/api/v1/analytics/"+endpoint+"?target_currency="+target+"&account_id="+scope, "", token)
						needsFX := (scope != empty && (scope != jpy || target == "USD")) || (endpoint == "equity-curve" && scope == empty && target == "USD")
						if !available && needsFX {
							require.Equal(t, 502, rec.Code, rec.Body.String())
							require.Contains(t, rec.Body.String(), "fx_unavailable")
							require.NotContains(t, rec.Body.String(), `"pnl"`)
							require.NotContains(t, rec.Body.String(), `"points"`)
							continue
						}
						require.Equal(t, 200, rec.Code, rec.Body.String())
						var out struct {
							Currency, TargetCurrency, FXPolicy string
							Pnl                                map[string]float64
							Points                             []struct{ Equity float64 }
							MaxDrawdown                        float64 `json:"max_drawdown"`
							Cash                               []struct {
								Amount   float64
								Currency string
							} `json:"cash_transactions"`
						}
						var meta map[string]any
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &meta))
						require.Equal(t, target, meta["currency"])
						require.Equal(t, target, meta["target_currency"])
						require.Equal(t, "latest", meta["fx_policy"])
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
						pnl, deposit := 3200.0, 1536000.0
						if scope == jpy+","+usd {
							deposit = 1526000
						}
						if scope == jpy {
							pnl, deposit = 200, 11000
						}
						if scope == empty {
							pnl, deposit = 0, 10000
						}
						if target == "USD" {
							pnl /= 150
							deposit /= 150
						}
						if endpoint == "daily" {
							sum := 0.0
							for _, v := range out.Pnl {
								sum += v
							}
							require.InDelta(t, pnl, sum, .01)
						} else {
							require.InDelta(t, pnl+deposit, out.Points[len(out.Points)-1].Equity, .02)
							sum := 0.0
							for _, ct := range out.Cash {
								require.Equal(t, target, ct.Currency)
								sum += ct.Amount
							}
							require.InDelta(t, deposit, sum, .01)
						}
					}
				}
				rec := do(s, http.MethodGet, "/api/v1/analytics/"+endpoint+"?target_currency=JP", "", token)
				require.Equal(t, 400, rec.Code)
				rec = do(s, http.MethodGet, "/api/v1/analytics/"+endpoint+"?target_currency=JPY&account_id=missing", "", token)
				require.Equal(t, 400, rec.Code)
			}
		})
	}
}

// Cash currency is independent of account metadata and of closed-trade currency.
func TestEquityCashCurrencyAndDrawdown(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(fmt.Sprint(available), func(t *testing.T) {
			provider := marketdata.NewYahooProvider()
			provider.Client = &http.Client{Transport: summaryFXTransport(func(r *http.Request) (*http.Response, error) {
				if !available {
					return nil, fmt.Errorf("cash FX unavailable")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"chart":{"result":[{"timestamp":[1780000000],"indicators":{"quote":[{"close":[150]}]}}]}}`)), Header: http.Header{}}, nil
			})}
			s := testServerWithProvider(t, true, nil, provider)
			token := registerAndLogin(t, s, "cash-currency@x.com")
			jpy := accountWithCurrency(t, s, token, "JPY", "JPY")
			rec := do(s, http.MethodPost, "/api/v1/cash-transactions", fmt.Sprintf(`{"account_id":%q,"type":"withdrawal","amount":-10,"currency":"USD","occurred_at":"2030-01-01T00:00:00Z"}`, jpy), token)
			require.Equal(t, 201, rec.Code, rec.Body.String())
			rec = do(s, http.MethodGet, "/api/v1/analytics/equity-curve?target_currency=JPY", "", token)
			if !available {
				require.Equal(t, 502, rec.Code, rec.Body.String())
				require.NotContains(t, rec.Body.String(), `"points"`)
				return
			}
			require.Equal(t, 200, rec.Code, rec.Body.String())
			var out struct {
				MaxDrawdown float64 `json:"max_drawdown"`
				Points      []struct{ Equity float64 }
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
			require.Equal(t, 1500.0, out.MaxDrawdown)
			require.Equal(t, 8500.0, out.Points[len(out.Points)-1].Equity)
		})
	}
}
