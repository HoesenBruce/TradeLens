package marketdata

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func httpRequest() Request {
	return Request{Symbol: "AAPL", InstrumentType: "stock", Interval: "D", From: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)}
}
func TestHTTPProviderContract(t *testing.T) {
	for _, symbol := range []string{"285A", "5803", "AAPL"} {
		for _, interval := range []string{"1", "5", "D"} {
			t.Run(symbol+interval, func(t *testing.T) {
				req := httpRequest()
				req.Symbol = symbol
				req.Interval = interval
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/v1/bars", r.URL.Path)
					require.Equal(t, chartSymbol(req), r.URL.Query().Get("symbol"))
					require.Equal(t, interval, r.URL.Query().Get("interval"))
					require.Equal(t, "stock", r.URL.Query().Get("instrument_type"))
					require.Equal(t, "2026-09-18T00:00:00Z", r.URL.Query().Get("from"))
					require.Equal(t, "2026-09-19T00:00:00Z", r.URL.Query().Get("to"))
					require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
					fmt.Fprintf(w, `{"symbol":%q,"interval":%q,"source":"archive","timezone":"Asia/Tokyo","adjustment_status":"unadjusted","bars":[{"timestamp":"2026-09-18T00:00:00Z","open":100,"high":110,"low":90,"close":105,"volume":0,"split_ratio":2}]}`, chartSymbol(req), interval)
				}))
				defer server.Close()
				p := NewProvider("http", "", server.URL, "secret").(*HTTPProvider)
				p.Client = server.Client()
				result, err := NewService(nil, p).GetBars(context.Background(), req)
				require.NoError(t, err)
				require.Equal(t, "archive", result.Source)
				require.Equal(t, "unadjusted", result.AdjustmentStatus)
				require.Equal(t, "2026-09-18", result.Bars[0].MarketDate)
				require.Equal(t, 2.0, result.Bars[0].SplitRatio)
			})
		}
	}
}
func TestHTTPProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		ok     bool
	}{
		{"empty", 200, `{"symbol":"AAPL","interval":"D","bars":[]}`, true},
		{"auth", 401, "secret", false}, {"unsupported", 422, "secret", false}, {"server", 500, "secret", false},
		{"malformed", 200, "{secret", false}, {"missing", 200, `{}`, false},
		{"partial", 200, `{"symbol":"AAPL","interval":"D","bars":[{"timestamp":"2026-09-18T00:00:00Z","close":100}]}`, false},
		{"invalid", 200, `{"symbol":"AAPL","interval":"D","bars":[{"timestamp":"2026-09-18T00:00:00Z","open":100,"high":90,"low":80,"close":100,"volume":1}]}`, false},
		{"naive", 200, `{"symbol":"AAPL","interval":"D","bars":[{"timestamp":"2026-09-18T00:00:00"}]}`, false},
		{"oversize", 200, strings.Repeat("x", maxHTTPBarsBytes+1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Empty(t, r.Header.Get("Authorization"))
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			result, err := NewHTTPProvider(server.URL, "").FetchResponse(context.Background(), httpRequest())
			if tc.ok {
				require.NoError(t, err)
				require.Empty(t, result.Bars)
				require.Equal(t, "unknown", result.AdjustmentStatus)
			} else {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "secret")
			}
		})
	}
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
		defer server.Close()
		p := NewHTTPProvider(server.URL, "secret")
		p.Client.Timeout = 10 * time.Millisecond
		_, err := p.FetchBars(context.Background(), httpRequest())
		require.ErrorContains(t, err, "timed out")
	})
	t.Run("network", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()
		_, err := NewHTTPProvider(server.URL, "secret").FetchBars(context.Background(), httpRequest())
		require.Error(t, err)
		require.NotContains(t, err.Error(), server.URL)
	})
	t.Run("redirect", func(t *testing.T) {
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("must not follow redirect") }))
		defer target.Close()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
		defer server.Close()
		_, err := NewHTTPProvider(server.URL, "secret").FetchBars(context.Background(), httpRequest())
		require.ErrorContains(t, err, "302")
	})
}

func TestHTTPOrderingAndRecordValidation(t *testing.T) {
	valid := `{"timestamp":"2026-09-18T01:00:00Z","open":100,"high":110,"low":90,"close":105,"volume":1}`
	early := strings.ReplaceAll(valid, "01:00:00Z", "00:00:00Z")
	for _, tc := range []struct {
		name, bars string
		ok         bool
	}{
		{"sort", valid + "," + early, true}, {"duplicate", valid + "," + valid, false},
		{"exclusive end", strings.ReplaceAll(valid, "18T01", "19T00"), false},
		{"before start", strings.ReplaceAll(valid, "18T01", "17T23"), false},
		{"nonfinite", strings.ReplaceAll(valid, `"close":105`, `"close":1e999`), false},
		{"date conflict", strings.ReplaceAll(valid, `"open":100`, `"market_date":"2026-09-17","open":100`), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"symbol":"AAPL","interval":"D","timezone":"Asia/Tokyo","bars":[%s]}`, tc.bars)
			}))
			defer server.Close()
			result, err := NewHTTPProvider(server.URL, "").FetchResponse(context.Background(), httpRequest())
			if tc.ok {
				require.NoError(t, err)
				require.Less(t, result.Bars[0].Time, result.Bars[1].Time)
			} else {
				require.Error(t, err)
			}
		})
	}
}
