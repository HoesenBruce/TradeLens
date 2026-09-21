package predictionvalidation

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

type evidenceProvider struct {
	name   string
	mutate func(*marketdata.Response)
	fail   bool
	calls  int
}

func (p *evidenceProvider) Name() string { return p.name }
func (p *evidenceProvider) FetchBars(ctx context.Context, req marketdata.Request) ([]marketdata.Bar, error) {
	r, e := p.FetchResponse(ctx, req)
	return r.Bars, e
}
func (p *evidenceProvider) FetchResponse(_ context.Context, req marketdata.Request) (marketdata.Response, error) {
	p.calls++
	if p.fail {
		return marketdata.Response{}, errors.New("offline")
	}
	market := "JP"
	if req.Symbol == "AAPL" {
		market = "US"
	}
	cal, _ := marketdata.TradingCalendar(market)
	loc, _ := time.LoadLocation(cal.Timezone)
	fetched := req.To.Add(time.Hour)
	r := marketdata.Response{Symbol: req.Symbol, Instrument: req.Symbol, Interval: req.Interval, Provider: p.name, Source: p.name, Timezone: cal.Timezone, AdjustmentStatus: "unadjusted", FetchedAt: &fetched}
	for _, s := range cal.Sessions {
		date, _ := time.ParseInLocation("2006-01-02", s.Date, loc)
		if date.Before(req.From) || !date.Before(req.To) {
			continue
		}
		r.Bars = append(r.Bars, marketdata.Bar{Time: date.Unix(), MarketDate: s.Date, Open: 100, High: 110, Low: 90, Close: 102, Volume: 0})
	}
	if p.mutate != nil {
		p.mutate(&r)
	}
	return r, nil
}
func instant(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
func input() Input {
	return Input{OwnerID: "owner", Prediction: store.Prediction{ID: "p", NewsAssetID: "a", Source: "user", Direction: "bullish", Reasoning: "unchanged", CreatedAt: instant("2026-09-13T00:00:00Z"), UpdatedAt: instant("2026-09-13T00:00:00Z")}, Asset: store.NewsAsset{ID: "a", NewsID: "n", AssetType: "stock", Symbol: "285A", Market: "JP"}}
}
func fixture(p *evidenceProvider) *Engine { return &Engine{Market: marketdata.NewService(nil, p)} }
func TestHorizonsAndProviderIndependence(t *testing.T) {
	for _, name := range []string{"yahoo", "finnhub", "http", "future"} {
		for _, h := range []int{1, 3, 5, 10, 20} {
			t.Run(name+string(rune('A'+h)), func(t *testing.T) {
				p := &evidenceProvider{name: name}
				in := input()
				before := in
				r := fixture(p).Evaluate(context.Background(), in, h, instant("2026-12-01T00:00:00Z"))
				require.Equal(t, "validated", r.Status, r.Reason)
				require.Len(t, r.Sessions, h)
				require.Equal(t, "2026-09-14", r.Sessions[0].Date)
				require.InDelta(t, .02, *r.AssetReturn, 1e-12)
				require.True(t, *r.DirectionCorrect)
				require.Equal(t, before, in)
				require.Equal(t, "D", r.ReferenceResolution)
				require.Equal(t, "trading_days", r.HorizonKind)
			})
		}
	}
}
func TestCalendarWindowsAndEligibility(t *testing.T) {
	for _, tc := range []struct{ at, date string }{
		{"2026-09-14T00:00:00+09:00", "2026-09-14"}, {"2026-09-14T09:00:00+09:00", "2026-09-15"},
		{"2026-09-14T12:00:00+09:00", "2026-09-15"}, {"2026-09-18T15:30:00+09:00", "2026-09-24"},
		{"2026-09-19T12:00:00+09:00", "2026-09-24"}, {"2026-09-22T12:00:00+09:00", "2026-09-24"},
	} {
		in := input()
		in.Prediction.CreatedAt = instant(tc.at)
		r := fixture(&evidenceProvider{name: "fake"}).Evaluate(context.Background(), in, 1, instant("2026-12-01T00:00:00Z"))
		require.Equal(t, tc.date, r.Sessions[0].Date)
	}
	p := &evidenceProvider{name: "fake"}
	e := fixture(p)
	r := e.Evaluate(context.Background(), input(), 1, instant("2026-09-14T06:31:00Z"))
	require.Equal(t, "pending", r.Status)
	require.Zero(t, p.calls)
	r = e.Evaluate(context.Background(), input(), 1, instant("2026-09-14T07:30:00Z"))
	require.Equal(t, "validated", r.Status)
	r = e.Evaluate(context.Background(), input(), 20, instant("2026-09-14T07:30:00Z"))
	require.Equal(t, "pending", r.Status)
	in := input()
	in.Asset.Symbol = "AAPL"
	in.Asset.Market = "US"
	in.Prediction.CreatedAt = instant("2026-11-26T12:00:00Z")
	r = e.Evaluate(context.Background(), in, 1, instant("2026-11-27T19:00:00Z"))
	require.Equal(t, instant("2026-11-27T18:00:00Z"), r.Sessions[0].Close)
	require.Equal(t, instant("2026-11-27T19:00:00Z"), *r.EligibleAt)
	cal, err := marketdata.TradingCalendar("US")
	require.NoError(t, err)
	opens := map[string]int{}
	for _, s := range cal.Sessions {
		opens[s.Date] = s.Open.Hour()
	}
	require.Equal(t, 14, opens["2026-03-06"])
	require.Equal(t, 13, opens["2026-03-09"])
}
func TestDataQualityStates(t *testing.T) {
	for _, tc := range []struct {
		name, status, reason string
		mutate               func(*marketdata.Response)
	}{
		{"no data", "unavailable", "no_bars", func(r *marketdata.Response) { r.Bars = nil }},
		{"missing suspended day", "incomplete", "missing_bars", func(r *marketdata.Response) { r.Bars = append(r.Bars[:2], r.Bars[3:]...) }},
		{"new listing check missing", "incomplete", "corporate_action_evidence_missing", func(r *marketdata.Response) { r.Bars = r.Bars[1:] }},
		{"duplicate", "incomplete", "conflicting_bars", func(r *marketdata.Response) { r.Bars = append(r.Bars, r.Bars[1]) }},
		{"nan", "incomplete", "invalid_prices", func(r *marketdata.Response) { r.Bars[1].Close = math.NaN() }},
		{"zero", "incomplete", "invalid_prices", func(r *marketdata.Response) { r.Bars[1].Open = 0 }},
		{"unknown adjustment", "incomplete", "adjustment_uncertain", func(r *marketdata.Response) { r.AdjustmentStatus = "unknown" }},
		{"old snapshot", "incomplete", "insufficient_fetch_time_evidence", func(r *marketdata.Response) { old := instant("2026-09-14T01:00:00Z"); r.FetchedAt = &old }},
		{"no provenance", "incomplete", "insufficient_fetch_time_evidence", func(r *marketdata.Response) { r.FetchedAt = nil }},
		{"split on reference", "incomplete", "corporate_action_boundary", func(r *marketdata.Response) { r.Bars[1].SplitRatio = 2 }},
		{"price discontinuity", "incomplete", "corporate_action_boundary", func(r *marketdata.Response) {
			for i := 1; i < len(r.Bars); i++ {
				r.Bars[i].Open /= 2
				r.Bars[i].High /= 2
				r.Bars[i].Low /= 2
				r.Bars[i].Close /= 2
			}
		}},
		{"wrong date", "incomplete", "invalid_market_date", func(r *marketdata.Response) { r.Bars[1].MarketDate = "2026-09-13" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := fixture(&evidenceProvider{name: "fake", mutate: tc.mutate}).Evaluate(context.Background(), input(), 5, instant("2026-12-01T00:00:00Z"))
			require.Equal(t, tc.status, r.Status)
			require.Equal(t, tc.reason, r.Reason)
			require.Nil(t, r.AssetReturn)
			_, err := Fingerprint(r)
			require.NoError(t, err)
		})
	}
	r := fixture(&evidenceProvider{fail: true}).Evaluate(context.Background(), input(), 1, instant("2026-12-01T00:00:00Z"))
	require.Equal(t, "provider_error", r.Reason)
	for _, market := range []string{"", "GB"} {
		in := input()
		in.Asset.Market = market
		r = fixture(&evidenceProvider{}).Evaluate(context.Background(), in, 1, time.Now())
		require.Equal(t, "unsupported_instrument", r.Reason)
	}
	in := input()
	in.Prediction.CreatedAt = instant("2031-01-01T00:00:00Z")
	r = fixture(&evidenceProvider{}).Evaluate(context.Background(), in, 1, time.Now())
	require.Equal(t, "calendar_unavailable", r.Reason)
}
func TestDecimalDirectionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		price     float64
		direction string
	}{{101, "bullish"}, {99, "bearish"}, {100, "neutral"}, {100.1, "neutral"}, {99.9, "neutral"}, {100.1001, "bullish"}, {99.8999, "bearish"}, {math.Nextafter(100.1, math.Inf(1)), "bullish"}, {math.Nextafter(99.9, math.Inf(-1)), "bearish"}} {
		require.Equal(t, tc.direction, priceDirection(100, tc.price))
		for _, direction := range []string{"bullish", "bearish", "neutral"} {
			in := input()
			in.Prediction.Direction = direction
			p := &evidenceProvider{name: "fake", mutate: func(r *marketdata.Response) { r.Bars[len(r.Bars)-1].Close = tc.price }}
			r := fixture(p).Evaluate(context.Background(), in, 1, instant("2026-12-01T00:00:00Z"))
			require.Equal(t, "validated", r.Status)
			require.Equal(t, direction == tc.direction, *r.DirectionCorrect)
		}
	}
}

func TestRejectedCandidateAndDividendEvidence(t *testing.T) {
	p := &evidenceProvider{name: "fake", mutate: func(r *marketdata.Response) {
		r.Bars[1].SplitRatio = 2
		r.CorporateActions = []marketdata.CorporateActionCandidate{
			{EffectiveDate: r.Bars[1].MarketDate, CandidateType: "stock_split", SuspectedRatio: 2, Status: "rejected", Source: "reviewed-source"},
			{EffectiveDate: r.Bars[1].MarketDate, CandidateType: "cash_dividend", Status: "confirmed", Source: "reviewed-source", Evidence: []string{"cash dividend excluded from price return"}},
		}
	}}
	r := fixture(p).Evaluate(context.Background(), input(), 1, instant("2026-12-01T00:00:00Z"))
	require.Equal(t, "validated", r.Status, r.Reason)
	require.Len(t, r.CorporateActions, 2)
}
