package exporter

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"github.com/tradermemos/api/internal/store"
)

func TestNewsBatchMarkdown(t *testing.T) {
	pending, validated := newsFixture(false), newsFixture(true)
	pending.News.ID = "z-pending"
	pending.News.Title = "Pending thesis"
	validated.News.ID = "a-validated"
	validated.News.Title = "Validated thesis"
	got := NewsBatchMarkdown([]NewsInput{validated, pending})
	require.Equal(t, got, NewsBatchMarkdown([]NewsInput{pending, validated}))
	require.Less(t, strings.Index(got, "# Pending thesis"), strings.Index(got, "# Validated thesis"))
	require.Contains(t, got, "count: 2\n")
	require.Contains(t, got, "1 trading days: pending")
	require.Contains(t, got, "1 trading days: validated")
	for _, in := range []NewsInput{pending, validated} {
		single := NewsMarkdown(in)
		body := strings.SplitN(single, "\n---\n", 2)[1]
		require.Contains(t, got, body, "batch must reuse the entire single-record body")
	}
	validated.News.PublishedAt = validated.News.PublishedAt.Add(time.Hour)
	reordered := NewsBatchMarkdown([]NewsInput{pending, validated})
	require.Less(t, strings.Index(reordered, "# Validated thesis"), strings.Index(reordered, "# Pending thesis"))
	require.Contains(t, NewsBatchMarkdown(nil), "count: 0\n")
	require.Contains(t, NewsBatchMarkdown(nil), "No News Theses match")
}

func TestNewsBatchFilters(t *testing.T) {
	in := newsFixture(true)
	in.Assets = append(in.Assets, store.NewsAsset{ID: "other", Symbol: "AAPL", Market: "US", AssetType: "stock"})
	in.Predictions[0].Prediction.NewsAssetID = "other"
	for _, tc := range []struct {
		f    predictionvalidation.PerformanceFilter
		want bool
	}{
		{predictionvalidation.PerformanceFilter{}, true},
		{predictionvalidation.PerformanceFilter{Symbol: " 285a ", Source: "ai", Horizon: 1, Category: "Industry", AssetType: "stock", From: "2026-09-20", To: "2026-09-20"}, true},
		{predictionvalidation.PerformanceFilter{Symbol: "285"}, false},
		{predictionvalidation.PerformanceFilter{Symbol: "285A", Source: "user"}, false},
		{predictionvalidation.PerformanceFilter{Symbol: "285A", Horizon: 5}, false},
		{predictionvalidation.PerformanceFilter{Symbol: "AAPL", Source: "user", Horizon: 5}, true},
		{predictionvalidation.PerformanceFilter{Category: "Other"}, false},
		{predictionvalidation.PerformanceFilter{AssetType: "etf"}, false},
		{predictionvalidation.PerformanceFilter{From: "2026-09-21"}, false},
		{predictionvalidation.PerformanceFilter{To: "2026-09-19"}, false},
	} {
		require.Equal(t, tc.want, MatchesNews(in, tc.f), "%+v", tc.f)
	}
	in.Predictions = nil
	require.True(t, MatchesNews(in, predictionvalidation.PerformanceFilter{Symbol: "285A"}))
	require.False(t, MatchesNews(in, predictionvalidation.PerformanceFilter{Source: "ai"}))
	in.Assets = nil
	require.True(t, MatchesNews(in, predictionvalidation.PerformanceFilter{}))
}
