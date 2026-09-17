package marketdata

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

var japaneseEquitySymbol = regexp.MustCompile(`^\d{4}$`)

func IsJapaneseEquity(req Request) bool {
	return req.InstrumentType == "stock" && japaneseEquitySymbol.MatchString(strings.TrimSpace(req.Symbol))
}

func MarketTimezone(req Request) string {
	if IsJapaneseEquity(req) {
		return "Asia/Tokyo"
	}
	return "America/New_York"
}

func normalizeBars(req Request, bars []Bar) []Bar {
	loc, err := time.LoadLocation(MarketTimezone(req))
	if err != nil {
		return bars
	}
	for i := range bars {
		bars[i].MarketDate = time.Unix(bars[i].Time, 0).In(loc).Format("2006-01-02")
	}
	return bars
}

// ChartableSymbol reports whether we should attempt a market data fetch.
// Test fixtures (E2E*) and obviously invalid tickers are skipped.
func ChartableSymbol(symbol string) bool {
	s := strings.TrimSpace(symbol)
	if s == "" {
		return false
	}
	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "E2E") {
		return false
	}
	if japaneseEquitySymbol.MatchString(s) {
		return true
	}
	// Require at least one letter; other pure numeric placeholders are not tickers.
	hasLetter := false
	for _, r := range upper {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	return hasLetter
}

// EmptyResponse builds a no-data payload for symbols we skip or when upstream has none.
func EmptyResponse(req Request, provider string) Response {
	return Response{
		Symbol: req.Symbol, Instrument: req.Symbol, Interval: req.Interval,
		From: FormatTimeRFC3339(req.From), To: FormatTimeRFC3339(req.To),
		Provider: provider, Source: provider, Timezone: MarketTimezone(req),
		AdjustmentStatus: "unadjusted", Cached: false, Bars: []Bar{},
	}
}
