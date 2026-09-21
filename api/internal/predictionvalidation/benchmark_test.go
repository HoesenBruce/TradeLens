package predictionvalidation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
)

func TestBenchmarkReturnsAndFailuresPreserveAsset(t *testing.T) {
	now := instant("2026-12-01T00:00:00Z")
	selection := &Benchmark{Symbol: "1306", Market: "JP", Currency: "JPY"}
	for _, h := range []int{1, 3, 5, 10, 20} {
		p := &evidenceProvider{name: "shared", mutate: func(r *marketdata.Response) {
			if r.Symbol == "1306" {
				for i := range r.Bars {
					r.Bars[i].Close = 103
				}
			}
		}}
		e := fixture(p)
		asset := e.Evaluate(context.Background(), input(), h, now)
		r := e.WithBenchmark(context.Background(), asset, selection, now)
		require.Equal(t, "validated", r.Status)
		require.Equal(t, "validated", r.BenchmarkStatus)
		require.InDelta(t, .03, *r.BenchmarkReturn, 1e-12)
		require.Equal(t, -.01, *r.ExcessReturn)
		require.True(t, *r.DirectionCorrect)
		require.False(t, *r.ExcessDirectionCorrect)
		require.Equal(t, asset.AssetReturn, r.AssetReturn)
		require.Equal(t, r.Sessions, r.BenchmarkEvidence.Sessions)
		without := e.WithBenchmark(context.Background(), r, nil, now)
		require.Equal(t, "not_requested", without.BenchmarkStatus)
		require.Nil(t, without.Benchmark)
		require.Nil(t, without.ExcessReturn)
		require.Equal(t, asset.AssetReturn, without.AssetReturn)
	}
	for _, tc := range []struct {
		name, status, reason string
		mutate               func(*marketdata.Response)
	}{
		{"absent", "unavailable", "no_bars", func(r *marketdata.Response) { r.Bars = nil }},
		{"partial", "incomplete", "missing_bars", func(r *marketdata.Response) { r.Bars = r.Bars[:2] }},
		{"split", "incomplete", "corporate_action_boundary", func(r *marketdata.Response) { r.Bars[1].SplitRatio = 2 }},
		{"stale", "incomplete", "insufficient_fetch_time_evidence", func(r *marketdata.Response) { r.FetchedAt = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &evidenceProvider{name: "shared", mutate: func(r *marketdata.Response) {
				if r.Symbol == "1306" {
					tc.mutate(r)
				}
			}}
			e := fixture(p)
			asset := e.Evaluate(context.Background(), input(), 3, now)
			r := e.WithBenchmark(context.Background(), asset, selection, now)
			require.Equal(t, "validated", r.Status)
			require.Equal(t, tc.status, r.BenchmarkStatus)
			require.Equal(t, tc.reason, r.BenchmarkReason)
			require.NotNil(t, r.AssetReturn)
			require.Nil(t, r.ExcessReturn)
		})
	}
	failed := fixture(&evidenceProvider{name: "shared", failSymbol: "1306"})
	raw := failed.Evaluate(context.Background(), input(), 1, now)
	failedBenchmark := failed.WithBenchmark(context.Background(), raw, selection, now)
	require.Equal(t, "validated", failedBenchmark.Status)
	require.Equal(t, "provider_error", failedBenchmark.BenchmarkReason)
	require.NotNil(t, failedBenchmark.AssetReturn)
	require.Nil(t, failedBenchmark.ExcessReturn)
	e := fixture(&evidenceProvider{name: "shared"})
	asset := e.Evaluate(context.Background(), input(), 1, now)
	mismatch := e.WithBenchmark(context.Background(), asset, &Benchmark{Symbol: "SPY", Market: "US", Currency: "USD"}, now)
	require.Equal(t, "benchmark_currency_mismatch", mismatch.BenchmarkReason)
	require.Equal(t, "validated", mismatch.Status)
	// A declared currency must match the benchmark market too.
	mismatch = e.WithBenchmark(context.Background(), asset, &Benchmark{Symbol: "SPY", Market: "US", Currency: "JPY"}, now)
	require.Equal(t, "benchmark_currency_mismatch", mismatch.BenchmarkReason)
	asset.Sessions[0].Close = asset.Sessions[0].Close.Add(-time.Hour)
	mismatch = e.WithBenchmark(context.Background(), asset, selection, now)
	require.Equal(t, "benchmark_calendar_mismatch", mismatch.BenchmarkReason)
}
func TestAlignmentComparesEverySession(t *testing.T) {
	sessions := []marketdata.Session{{Date: "2026-09-14", Open: instant("2026-09-14T00:00:00Z"), Close: instant("2026-09-14T06:30:00Z")}, {Date: "2026-09-15", Open: instant("2026-09-15T00:00:00Z"), Close: instant("2026-09-15T06:30:00Z")}, {Date: "2026-09-16", Open: instant("2026-09-16T00:00:00Z"), Close: instant("2026-09-16T06:30:00Z")}}
	require.True(t, alignedSessions(sessions, sessions))
	require.False(t, alignedSessions(sessions, append(sessions[:1:1], sessions[2:]...)), "non-overlapping holiday")
	for _, change := range []func(*marketdata.Session){func(s *marketdata.Session) { s.Date = "2026-09-17" }, func(s *marketdata.Session) { s.Open = s.Open.Add(time.Hour) }, func(s *marketdata.Session) { s.Close = s.Close.Add(-time.Hour) }} {
		copy := append([]marketdata.Session(nil), sessions...)
		change(&copy[1])
		require.False(t, alignedSessions(sessions, copy))
	}
}
func TestExcessBoundaryAndIncompleteAsset(t *testing.T) {
	now := instant("2026-12-01T00:00:00Z")
	for _, tc := range []struct {
		close   float64
		neutral bool
	}{{100.1, true}, {99.9, true}, {100.1001, false}, {99.8999, false}} {
		in := input()
		in.Prediction.Direction = "neutral"
		e := fixture(&evidenceProvider{name: "shared", mutate: func(r *marketdata.Response) {
			price := tc.close
			if r.Symbol == "1306" {
				price = 100
			}
			for i := range r.Bars {
				r.Bars[i].Close = price
			}
		}})
		asset := e.Evaluate(context.Background(), in, 1, now)
		r := e.WithBenchmark(context.Background(), asset, &Benchmark{Symbol: "1306", Market: "JP", Currency: "JPY"}, now)
		require.Equal(t, tc.neutral, *r.ExcessDirectionCorrect)
	}
	e := fixture(&evidenceProvider{name: "shared", mutate: func(r *marketdata.Response) {
		if r.Symbol == "285A" {
			r.Bars = nil
		}
	}})
	asset := e.Evaluate(context.Background(), input(), 1, now)
	r := e.WithBenchmark(context.Background(), asset, &Benchmark{Symbol: "1306", Market: "JP", Currency: "JPY"}, now)
	require.Equal(t, "unavailable", r.Status)
	require.Equal(t, "validated", r.BenchmarkStatus)
	require.NotNil(t, r.BenchmarkReturn)
	require.Nil(t, r.ExcessReturn)
}
