package marketdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOccUnderlying(t *testing.T) {
	require.Equal(t, "TSLA", occUnderlying("TSLA250117C00425000"))
	require.Equal(t, "AAPL", occUnderlying("AAPL240315P00170000"))
	require.Equal(t, "", occUnderlying("TSLA"))
}

func TestJapaneseEquityUsesTokyoYahooSymbol(t *testing.T) {
	for symbol, want := range map[string]string{"7203": "7203.T", "584A": "584A.T"} {
		req := Request{Symbol: symbol, InstrumentType: "stock"}
		require.Equal(t, want, chartSymbol(req))
	}
}

func TestNormalizeBarsAddsTokyoMarketDate(t *testing.T) {
	req := Request{Symbol: "7203", InstrumentType: "stock"}
	bars := normalizeBars(req, []Bar{{Time: time.Date(2026, 9, 15, 15, 0, 0, 0, time.UTC).Unix()}})
	require.Equal(t, "2026-09-16", bars[0].MarketDate)
}

func TestApplyYahooSplitsRestoresUnadjustedBarsAndAddsProviderEvent(t *testing.T) {
	req := Request{Symbol: "5803", InstrumentType: "stock"}
	effective := time.Date(2026, 4, 1, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	bars := []Bar{
		{Time: effective.AddDate(0, 0, -1).Unix(), Open: 100, High: 102, Low: 99, Close: 101, Volume: 600},
		{Time: effective.Unix(), Open: 110, High: 112, Low: 109, Close: 111, Volume: 900},
	}

	applyYahooSplits(req, bars, map[string]yahooSplit{
		"event": {Date: effective.Unix(), Numerator: 6, Denominator: 1},
	})

	require.Equal(t, Bar{
		Time: effective.AddDate(0, 0, -1).Unix(), Open: 600, High: 612, Low: 594, Close: 606, Volume: 100,
	}, bars[0])
	require.Equal(t, 6.0, bars[1].SplitRatio)
	require.Equal(t, 110.0, bars[1].Open)
	require.Equal(t, "unadjusted", adjustmentStatus("yahoo"))
}

func TestDefaultInterval(t *testing.T) {
	from := time.Date(2026, 3, 10, 9, 30, 0, 0, time.UTC)
	require.Equal(t, "1", DefaultInterval(from, from.Add(90*time.Minute)))
	require.Equal(t, "5", DefaultInterval(from, from.Add(4*time.Hour)))
	require.Equal(t, "D", DefaultInterval(from, from.Add(30*24*time.Hour)))
}

func TestCacheKeyStable(t *testing.T) {
	from := time.Date(2026, 3, 10, 9, 30, 0, 0, time.UTC)
	to := from.Add(time.Hour)
	req := Request{Symbol: "aapl", InstrumentType: "stock", Interval: "5", From: from, To: to}
	k1 := CacheKey(req)
	req.Symbol = "AAPL"
	k2 := CacheKey(req)
	require.Equal(t, k1, k2)
}

func TestParseInterval(t *testing.T) {
	v, err := ParseInterval("")
	require.NoError(t, err)
	require.Equal(t, "5", v)
	_, err = ParseInterval("2h")
	require.Error(t, err)
}
