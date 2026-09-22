package exporter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tradermemos/api/internal/predictionvalidation"
	"github.com/tradermemos/api/internal/store"
)

// MatchesNews selects whole theses, preserving their AI/User context in the export.
// Prediction constraints must match the same prediction and its affected asset.
func MatchesNews(in NewsInput, f predictionvalidation.PerformanceFilter) bool {
	if !f.MatchesNews(in.News) {
		return false
	}
	assets := make(map[string]store.NewsAsset, len(in.Assets))
	assetMatch := strings.TrimSpace(f.Symbol) == "" && f.AssetType == ""
	for _, a := range in.Assets {
		assets[a.ID] = a
		if f.MatchesAsset(a) {
			assetMatch = true
		}
	}
	if !assetMatch {
		return false
	}
	if f.Source == "" && f.Horizon == 0 {
		return true
	}
	for _, p := range in.Predictions {
		a, ok := assets[p.Prediction.NewsAssetID]
		if !ok || !f.MatchesAsset(a) || !f.MatchesPrediction(p.Prediction) {
			continue
		}
		if f.Horizon == 0 {
			return true
		}
		for _, h := range p.Horizons {
			if f.MatchesHorizon(h) {
				return true
			}
		}
	}
	return false
}

// NewsBatchMarkdown sorts like ListNews (publication descending, then ID descending).
// Embedded single-note frontmatter is fenced so it cannot become a setext heading.
func NewsBatchMarkdown(inputs []NewsInput) string {
	rows := append([]NewsInput(nil), inputs...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].News.PublishedAt.Equal(rows[j].News.PublishedAt) {
			return rows[i].News.ID > rows[j].News.ID
		}
		return rows[i].News.PublishedAt.After(rows[j].News.PublishedAt)
	})
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntype: news-thesis-batch\ncount: %d\n---\n\n# News Thesis export\n", len(rows))
	if len(rows) == 0 {
		b.WriteString("\nNo News Theses match the selection or filters.\n")
		return b.String()
	}
	b.WriteString("\n## Index\n\n")
	for i, in := range rows {
		fmt.Fprintf(&b, "%d. [%s](#thesis-%d)\n", i+1, markdownText(in.News.Title), i+1)
	}
	for i, in := range rows {
		fmt.Fprintf(&b, "\n---\n\n<a id=\"thesis-%d\"></a>\n\n", i+1)
		note := NewsMarkdown(in)
		end := strings.Index(note[4:], "\n---\n") + 4
		b.WriteString("```yaml\n")
		b.WriteString(note[4:end])
		b.WriteString("\n```\n")
		b.WriteString(note[end+5:])
	}
	return b.String()
}
