package predictionvalidation

import (
	"context"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

type Benchmark struct {
	Symbol   string `json:"symbol"`
	Market   string `json:"market"`
	Currency string `json:"currency"`
}

// WithBenchmark leaves the asset result intact even if benchmark evidence fails.
func (e *Engine) WithBenchmark(ctx context.Context, r Result, selection *Benchmark, now time.Time) Result {
	r.BenchmarkStatus = "not_requested"
	r.BenchmarkReason = ""
	r.Benchmark, r.BenchmarkEvidence, r.BenchmarkReturn, r.ExcessReturn, r.ExcessDirectionCorrect = nil, nil, nil, nil, nil
	if selection == nil {
		return r
	}
	selected := *selection
	selected.Symbol = strings.ToUpper(strings.TrimSpace(selected.Symbol))
	selected.Market = strings.ToUpper(strings.TrimSpace(selected.Market))
	selected.Currency = strings.ToUpper(strings.TrimSpace(selected.Currency))
	r.Benchmark = &selected
	fail := func(reason string) Result { r.BenchmarkStatus = "unavailable"; r.BenchmarkReason = reason; return r }
	if selected.Symbol == "" {
		return fail("unsupported_benchmark")
	}
	cal, err := marketdata.TradingCalendar(selected.Market)
	if err != nil {
		return fail("calendar_unavailable")
	}
	if selected.Currency != cal.Currency || selected.Currency != r.Currency {
		return fail("benchmark_currency_mismatch")
	}
	start := sort.Search(len(cal.Sessions), func(i int) bool { return cal.Sessions[i].Open.After(r.PredictionAsOf) })
	if start == 0 || start+r.Horizon > len(cal.Sessions) || len(r.Sessions) == 0 {
		return fail("calendar_unavailable")
	}
	if !alignedSessions(r.Sessions, cal.Sessions[start:start+r.Horizon]) {
		return fail("benchmark_calendar_mismatch")
	}
	in := r.Input
	// A benchmark is explicitly selected price evidence, not another persisted prediction.
	in.Asset = store.NewsAsset{NewsID: in.Asset.NewsID, AssetType: "stock", Symbol: selected.Symbol, Market: selected.Market, Source: "user"}
	benchmark := e.Evaluate(ctx, in, r.Horizon, now)
	benchmark.DirectionCorrect = nil // the prediction is about the original asset, not the benchmark
	r.BenchmarkEvidence = &benchmark
	r.BenchmarkStatus = benchmark.Status
	r.BenchmarkReason = benchmark.Reason
	if benchmark.Status != "validated" {
		return r
	}
	r.BenchmarkReturn = benchmark.AssetReturn
	if r.Status != "validated" {
		return r
	}
	// Calculate the difference and classify in decimal rationals, before float/display rounding.
	excess := new(big.Rat).Sub(new(big.Rat).Quo(decimal(*r.OutcomePrice), decimal(*r.ReferencePrice)), new(big.Rat).Quo(decimal(*benchmark.OutcomePrice), decimal(*benchmark.ReferencePrice)))
	value, _ := excess.Float64()
	if !finite(value) {
		r.BenchmarkStatus = "incomplete"
		r.BenchmarkReason = "invalid_excess_return"
		return r
	}
	direction := "neutral"
	if excess.Cmp(big.NewRat(1, 1000)) > 0 {
		direction = "bullish"
	} else if excess.Cmp(big.NewRat(-1, 1000)) < 0 {
		direction = "bearish"
	}
	correct := direction == r.Input.Prediction.Direction
	r.ExcessReturn = &value
	r.ExcessDirectionCorrect = &correct
	return r
}

func alignedSessions(asset, benchmark []marketdata.Session) bool {
	if len(asset) != len(benchmark) {
		return false
	}
	for i, s := range asset {
		b := benchmark[i]
		if s.Date != b.Date || !s.Open.Equal(b.Open) || !s.Close.Equal(b.Close) {
			return false
		}
	}
	return true
}
