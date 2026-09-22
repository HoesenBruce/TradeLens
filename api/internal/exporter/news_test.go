package exporter

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"github.com/tradermemos/api/internal/store"
	"go.yaml.in/yaml/v3"
)

func newsFixture(validated bool) NewsInput {
	at := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	in := NewsInput{
		News:   store.News{ID: "news-1", Title: "政策: \"AI\" #1 [link]\n---", Source: "Desk & <wire>", Url: "https://example.com/?a=1&b=2", PublishedAt: at, CreatedAt: at, UpdatedAt: at, Category: "Industry", Tags: `["AI: chips","#watch","[x]"]`, Summary: "Demand improves", OriginalText: "```\n<script> & [[link]]\n---\n````", Notes: "Review: wait\n- no trade"},
		Assets: []store.NewsAsset{{ID: "asset-1", Symbol: "285A", DisplayName: "日本企業", AssetType: "stock", Market: "JP", Exchange: "TSE", Source: "user", Relation: "Supplier"}},
		Predictions: []NewsPrediction{
			{Prediction: store.Prediction{ID: "p-user", NewsAssetID: "asset-1", Source: "user", Direction: "neutral", Reasoning: "Wait & see"}, Horizons: []int64{5, 1}},
			{Prediction: store.Prediction{ID: "p-ai", NewsAssetID: "asset-1", Source: "ai", Direction: "bullish", Confidence: sql.NullInt64{Int64: 80, Valid: true}, Reasoning: "Demand > supply", Catalysts: "Launch", Risks: "Delay", Invalidation: "Cancellation"}, Horizons: []int64{1}},
		},
	}
	if validated {
		ret, excess, yes := 0.1, 0.05, true
		in.Predictions[1].Evaluations = []predictionvalidation.Evaluation{{ID: "eval-1", Current: true, Result: predictionvalidation.Result{Horizon: 1, Status: "validated", Rules: predictionvalidation.RulesVersion, CalculatedAt: at, AssetReturn: &ret, DirectionCorrect: &yes, Benchmark: &predictionvalidation.Benchmark{Symbol: "1306", Market: "JP", Currency: "JPY"}, BenchmarkStatus: "validated", BenchmarkReturn: &excess, ExcessReturn: &excess, ExcessDirectionCorrect: &yes, Explanation: "Price return; not causation"}}}
	}
	return in
}

func TestNewsMarkdownGolden(t *testing.T) {
	for _, state := range []string{"pending", "validated"} {
		t.Run(state, func(t *testing.T) {
			in := newsFixture(state == "validated")
			got := NewsMarkdown(in)
			path := filepath.Join("testdata", "news-"+state+".md")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				require.NoError(t, os.MkdirAll("testdata", 0755))
				require.NoError(t, os.WriteFile(path, []byte(got), 0644))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(want), got)
			var front struct {
				Title string
				Tags  []string
			}
			require.NoError(t, yaml.Unmarshal([]byte(strings.SplitN(got, "---\n", 3)[1]), &front))
			require.Equal(t, in.News.Title, front.Title)
			require.Equal(t, []string{"AI: chips", "#watch", "[x]"}, front.Tags)
			require.Contains(t, got, "`````text\n```\n<script>")
			require.NotContains(t, got, "# 政策: \"AI\"")
			in.Predictions[0], in.Predictions[1] = in.Predictions[1], in.Predictions[0]
			require.Equal(t, got, NewsMarkdown(in))
			require.Equal(t, []int64{5, 1}, in.Predictions[1].Horizons, "renderer must not mutate input")
		})
	}
	require.Equal(t, "news-thesis-etc-passwd.md", NewsFilename(store.News{ID: "../../etc/passwd\r\n"}))
	require.Equal(t, "news-thesis-news.md", NewsFilename(store.News{ID: "日本"}))
}
