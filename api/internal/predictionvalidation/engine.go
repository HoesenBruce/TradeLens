// Package predictionvalidation implements news-validation-v1 over shared market data.
package predictionvalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

const RulesVersion = "news-validation-v1"

type Input struct {
	RevisionNumber int64            `json:"revision_number"`
	Prediction     store.Prediction `json:"prediction"`
	Asset          store.NewsAsset  `json:"asset"`
	PublishedAt    time.Time        `json:"published_at"`
	OwnerID        string           `json:"owner_id"`
}
type Result struct {
	Benchmark                  *Benchmark                            `json:"benchmark"`
	BenchmarkStatus            string                                `json:"benchmark_status"`
	BenchmarkReason            string                                `json:"benchmark_reason"`
	BenchmarkEvidence          *Result                               `json:"benchmark_evidence"`
	BenchmarkReturn            *float64                              `json:"benchmark_return"`
	ExcessReturn               *float64                              `json:"excess_return"`
	ExcessDirectionCorrect     *bool                                 `json:"excess_direction_correct"`
	Input                      Input                                 `json:"input"`
	Revision                   string                                `json:"prediction_revision"`
	Rules                      string                                `json:"rules_version"`
	Epsilon                    string                                `json:"neutral_epsilon"`
	HorizonKind                string                                `json:"horizon_kind"`
	Horizon                    int                                   `json:"horizon_value"`
	ReferencePolicy            string                                `json:"reference_policy"`
	OutcomePolicy              string                                `json:"outcome_policy"`
	ReferenceResolution        string                                `json:"reference_resolution"`
	OutcomeResolution          string                                `json:"outcome_resolution"`
	ReferencePriceField        string                                `json:"reference_price_field"`
	OutcomePriceField          string                                `json:"outcome_price_field"`
	PredictionAsOf             time.Time                             `json:"prediction_as_of"`
	CalculatedAt               time.Time                             `json:"calculated_at"`
	CalendarVersion            string                                `json:"calendar_version"`
	Market                     string                                `json:"market"`
	Currency                   string                                `json:"currency"`
	Timezone                   string                                `json:"timezone"`
	Sessions                   []marketdata.Session                  `json:"sessions"`
	CheckSession               *marketdata.Session                   `json:"corporate_action_check_session"`
	EligibleAt                 *time.Time                            `json:"eligible_at"`
	Request                    *marketdata.Request                   `json:"request"`
	Evidence                   *marketdata.Response                  `json:"evidence"`
	InvalidEvidence            []string                              `json:"invalid_evidence"`
	MissingDates               []string                              `json:"missing_dates"`
	CorporateActions           []marketdata.CorporateActionCandidate `json:"corporate_actions"`
	CorporateActionLimitations string                                `json:"corporate_action_limitations"`
	ReferencePrice             *float64                              `json:"reference_price"`
	OutcomePrice               *float64                              `json:"outcome_price"`
	AssetReturn                *float64                              `json:"asset_return"`
	ObservedDirection          *string                               `json:"observed_direction"`
	DirectionCorrect           *bool                                 `json:"direction_correct"`
	Status                     string                                `json:"status"`
	Reason                     string                                `json:"reason_code"`
	Explanation                string                                `json:"explanation"`
}

type Engine struct {
	Market *marketdata.Service
	Store  store.Querier
}

func Revision(in Input) string {
	// Asset identity/market edits must not reuse outcomes for a different instrument.
	raw, _ := json.Marshal(struct {
		Prediction store.Prediction
		Revision   int64
		Asset      store.NewsAsset
	}{in.Prediction, in.RevisionNumber, in.Asset})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (e *Engine) Evaluate(ctx context.Context, in Input, horizon int, now time.Time) Result {
	r := Result{BenchmarkStatus: "not_requested", Input: in, Revision: Revision(in), Rules: RulesVersion, Epsilon: "0.001", HorizonKind: "trading_days", Horizon: horizon, ReferencePolicy: "next_regular_session_open", OutcomePolicy: "horizon_regular_session_close", ReferenceResolution: "D", OutcomeResolution: "D", ReferencePriceField: "open", OutcomePriceField: "close", PredictionAsOf: in.Prediction.UpdatedAt, CalculatedAt: now.UTC(), Market: strings.ToUpper(in.Asset.Market), CorporateActionLimitations: "reported split ratios plus OHLC gap heuristic; no authoritative dividend, listing, suspension or event coverage"}
	fail := func(status, reason string) Result {
		r.Status = status
		r.Reason = reason
		r.Explanation = strings.ReplaceAll(reason, "_", " ")
		return r
	}
	if in.Prediction.CreatedAt.After(r.PredictionAsOf) {
		r.PredictionAsOf = in.Prediction.CreatedAt
	}
	if r.PredictionAsOf.IsZero() {
		return fail("unavailable", "invalid_prediction_time")
	}
	if in.Asset.AssetType != "stock" || (r.Market != "JP" && r.Market != "US") || strings.TrimSpace(in.Asset.Symbol) == "" {
		return fail("unavailable", "unsupported_instrument")
	}
	if horizon != 1 && horizon != 3 && horizon != 5 && horizon != 10 && horizon != 20 {
		return fail("unavailable", "unsupported_horizon")
	}
	if in.Prediction.Direction != "bullish" && in.Prediction.Direction != "bearish" && in.Prediction.Direction != "neutral" {
		return fail("unavailable", "unsupported_direction")
	}
	cal, err := marketdata.TradingCalendar(r.Market)
	if err != nil {
		return fail("unavailable", "calendar_unavailable")
	}
	r.CalendarVersion = cal.Version
	r.Timezone = cal.Timezone
	r.Currency = cal.Currency
	start := sort.Search(len(cal.Sessions), func(i int) bool { return cal.Sessions[i].Open.After(r.PredictionAsOf) })
	if start == 0 || start+horizon > len(cal.Sessions) {
		return fail("unavailable", "calendar_unavailable")
	}
	r.Sessions = append([]marketdata.Session(nil), cal.Sessions[start:start+horizon]...)
	check := cal.Sessions[start-1]
	r.CheckSession = &check
	eligible := r.Sessions[horizon-1].Close.Add(time.Hour)
	r.EligibleAt = &eligible
	if now.Before(eligible) {
		return fail("pending", "awaiting_close")
	}
	loc, _ := time.LoadLocation(cal.Timezone)
	first := check.Open.In(loc)
	last := r.Sessions[horizon-1].Close.In(loc)
	req := marketdata.Request{Symbol: in.Asset.Symbol, InstrumentType: in.Asset.AssetType, Interval: "D", From: time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc).UTC(), To: time.Date(last.Year(), last.Month(), last.Day()+1, 0, 0, 0, 0, loc).UTC()}
	r.Request = &req
	response, err := e.Market.RefreshBars(ctx, req)
	if err != nil {
		if errors.Is(err, marketdata.ErrUnsupportedResolution) {
			return fail("unavailable", "unsupported_resolution")
		}
		return fail("unavailable", "provider_error")
	}
	r.Evidence = &response
	if len(response.Bars) == 0 {
		return fail("unavailable", "no_bars")
	}
	// Non-finite source values cannot be serialized as JSON. Preserve their textual
	// evidence separately and never replace them with synthetic prices.
	safe := make([]marketdata.Bar, 0, len(response.Bars))
	for _, b := range response.Bars {
		if !finite(b.Open, b.High, b.Low, b.Close, b.Volume, b.SplitRatio) {
			raw := []string{strconv.FormatInt(b.Time, 10), b.MarketDate, strconv.FormatFloat(b.Open, 'g', -1, 64), strconv.FormatFloat(b.High, 'g', -1, 64), strconv.FormatFloat(b.Low, 'g', -1, 64), strconv.FormatFloat(b.Close, 'g', -1, 64), strconv.FormatFloat(b.Volume, 'g', -1, 64)}
			r.InvalidEvidence = append(r.InvalidEvidence, strings.Join(raw, "|"))
			continue
		}
		safe = append(safe, b)
	}
	response.Bars = safe
	if len(r.InvalidEvidence) > 0 {
		return fail("incomplete", "invalid_prices")
	}
	if response.Interval != "D" || response.Timezone != cal.Timezone || response.Provider == "" || response.Source == "" {
		return fail("incomplete", "insufficient_metadata")
	}
	if response.AdjustmentStatus != "unadjusted" {
		return fail("incomplete", "adjustment_uncertain")
	}
	expected := map[string]bool{check.Date: true}
	for _, s := range r.Sessions {
		expected[s.Date] = true
	}
	bars := map[string]marketdata.Bar{}
	for _, b := range response.Bars {
		if b.Time <= 0 || b.MarketDate != time.Unix(b.Time, 0).In(loc).Format("2006-01-02") || !expected[b.MarketDate] {
			return fail("incomplete", "invalid_market_date")
		}
		if _, exists := bars[b.MarketDate]; exists {
			return fail("incomplete", "conflicting_bars")
		}
		if b.Open <= 0 || b.Close <= 0 || b.Low <= 0 || b.High < b.Open || b.High < b.Close || b.Low > b.Open || b.Low > b.Close || b.Volume < 0 {
			return fail("incomplete", "invalid_prices")
		}
		bars[b.MarketDate] = b
	}
	for _, s := range r.Sessions {
		if _, ok := bars[s.Date]; !ok {
			r.MissingDates = append(r.MissingDates, s.Date)
		}
	}
	if len(r.MissingDates) > 0 {
		return fail("incomplete", "missing_bars")
	}
	if _, ok := bars[check.Date]; !ok {
		return fail("incomplete", "corporate_action_evidence_missing")
	}
	// The entire source snapshot must have been acquired after this horizon became eligible.
	// A bars-only live provider exposes request-start time; an archive may expose per-bar times.
	for _, s := range append([]marketdata.Session{check}, r.Sessions...) {
		b := bars[s.Date]
		fetched := b.FetchedAt
		if fetched == nil {
			fetched = response.FetchedAt
		}
		if fetched == nil || fetched.Before(eligible) {
			return fail("incomplete", "insufficient_fetch_time_evidence")
		}
	}
	ordered := append([]marketdata.Bar(nil), response.Bars...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Time < ordered[j].Time })
	for i := 1; i < len(ordered); i++ {
		if !finite(ordered[i-1].Close/ordered[i].Open, ordered[i].Open/ordered[i-1].Close) {
			return fail("incomplete", "invalid_prices")
		}
	}
	r.CorporateActions = append([]marketdata.CorporateActionCandidate(nil), response.CorporateActions...)
	for _, detected := range marketdata.FindCorporateActionCandidates(response) {
		rejected := false
		for _, known := range response.CorporateActions {
			if known.Status == "rejected" && known.EffectiveDate == detected.EffectiveDate && known.CandidateType == detected.CandidateType && known.SuspectedRatio == detected.SuspectedRatio {
				rejected = true
				break
			}
		}
		if !rejected {
			r.CorporateActions = append(r.CorporateActions, detected)
		}
	}
	for _, c := range r.CorporateActions {
		if _, err := time.Parse("2006-01-02", c.EffectiveDate); err != nil || c.CandidateType == "" || (c.Status != "rejected" && c.Status != "confirmed" && c.Status != "unconfirmed") {
			return fail("incomplete", "insufficient_metadata")
		}
		if c.EffectiveDate >= r.Sessions[0].Date && c.EffectiveDate <= last.Format("2006-01-02") && c.Status != "rejected" && c.CandidateType != "cash_dividend" {
			return fail("incomplete", "corporate_action_boundary")
		}
	}
	p0 := bars[r.Sessions[0].Date].Open
	ph := bars[r.Sessions[horizon-1].Date].Close
	ret := ph/p0 - 1
	if !finite(ret) {
		return fail("incomplete", "invalid_prices")
	}
	observed := priceDirection(p0, ph)
	correct := observed == in.Prediction.Direction
	r.ReferencePrice = &p0
	r.OutcomePrice = &ph
	r.AssetReturn = &ret
	r.ObservedDirection = &observed
	r.DirectionCorrect = &correct
	r.Status = "validated"
	r.Explanation = "unadjusted price return; does not establish news causation"
	return r
}

func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func decimal(v float64) *big.Rat {
	r, _ := new(big.Rat).SetString(strconv.FormatFloat(v, 'f', -1, 64))
	return r
}
func priceDirection(open, close float64) string {
	p, c := decimal(open), decimal(close)
	if c.Cmp(new(big.Rat).Mul(p, big.NewRat(1001, 1000))) > 0 {
		return "bullish"
	}
	if c.Cmp(new(big.Rat).Mul(p, big.NewRat(999, 1000))) < 0 {
		return "bearish"
	}
	return "neutral"
}
