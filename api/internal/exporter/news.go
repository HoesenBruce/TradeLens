package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tradermemos/api/internal/predictionvalidation"
	"github.com/tradermemos/api/internal/store"
)

// NewsInput contains only saved data. Loading an export never fetches market prices.
type NewsInput struct {
	News        store.News
	Assets      []store.NewsAsset
	Predictions []NewsPrediction
}

type NewsPrediction struct {
	Prediction  store.Prediction
	Horizons    []int64
	Evaluations []predictionvalidation.Evaluation
}

// LoadNews reads through the caller's snapshot transaction and owner-scoped queries.
func LoadNews(ctx context.Context, q store.Querier, owner, id string) (NewsInput, error) {
	var out NewsInput
	var err error
	out.News, err = q.GetNews(ctx, store.GetNewsParams{ID: id, UserID: owner})
	if err != nil {
		return out, err
	}
	out.Assets, err = q.ListNewsAssets(ctx, store.ListNewsAssetsParams{NewsID: id, UserID: owner})
	if err != nil {
		return out, err
	}
	predictions, err := q.ListPredictions(ctx, store.ListPredictionsParams{NewsID: id, UserID: owner})
	if err != nil {
		return out, err
	}
	engine := predictionvalidation.Engine{Store: q}
	for _, p := range predictions {
		item := NewsPrediction{Prediction: p}
		horizons, err := q.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: p.ID, UserID: owner})
		if err != nil {
			return out, err
		}
		for _, h := range horizons {
			item.Horizons = append(item.Horizons, h.TradingDays)
		}
		item.Evaluations, err = engine.History(ctx, owner, id, p.ID)
		if err != nil {
			return out, err
		}
		out.Predictions = append(out.Predictions, item)
	}
	return out, nil
}

// NewsFilename is independent of export time and never contains user-controlled path syntax.
func NewsFilename(n store.News) string {
	token := filenameToken(n.ID)
	if token == "" {
		token = "news"
	}
	return "news-thesis-" + token + ".md"
}

// markdownText escapes punctuation as entities, including Obsidian wikilinks and HTML.
func markdownText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' {
			b.WriteByte(' ')
		} else if strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r) {
			fmt.Fprintf(&b, "&#%d;", r)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func yamlString(s string) string {
	// JSON double-quoted strings are also valid YAML scalars.
	b, _ := json.Marshal(s)
	return string(b)
}

func newsText(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "\n### %s\n\n", label)
	if value == "" {
		b.WriteString("(blank)\n")
		return
	}
	// A fence longer than any run in the input keeps arbitrary Markdown literal.
	fence := "```"
	for strings.Contains(value, fence) {
		fence += "`"
	}
	fmt.Fprintf(b, "%stext\n%s\n%s\n", fence, value, fence)
}

func optionalNumber(v *float64) string {
	if v == nil {
		return "(blank)"
	}
	return strconv.FormatFloat(*v, 'g', -1, 64)
}
func optionalBool(v *bool) string {
	if v == nil {
		return "(blank)"
	}
	return strconv.FormatBool(*v)
}

// NewsMarkdown renders a deterministic Obsidian note; no export-time clock is included.
func NewsMarkdown(in NewsInput) string {
	var b strings.Builder
	n := in.News
	fmt.Fprintf(&b, "---\ntype: news-thesis\nid: %s\ntitle: %s\nsource: %s\nurl: %s\npublished_at: %s\ncreated_at: %s\nupdated_at: %s\ncategory: %s\n", yamlString(n.ID), yamlString(n.Title), yamlString(n.Source), yamlString(n.Url), yamlString(formatTime(n.PublishedAt)), yamlString(formatTime(n.CreatedAt)), yamlString(formatTime(n.UpdatedAt)), yamlString(n.Category))
	var tags []string
	_ = json.Unmarshal([]byte(n.Tags), &tags)
	if len(tags) == 0 {
		b.WriteString("tags: []\n")
	} else {
		b.WriteString("tags:\n")
		for _, tag := range tags {
			fmt.Fprintf(&b, "  - %s\n", yamlString(tag))
		}
	}
	fmt.Fprintf(&b, "---\n\n# %s\n\n- Source: %s\n- URL: %s\n- Published (UTC): %s\n", markdownText(n.Title), markdownText(n.Source), markdownText(n.Url), formatTime(n.PublishedAt))
	newsText(&b, "Summary", n.Summary)
	newsText(&b, "Original text", n.OriginalText)
	newsText(&b, "Review notes", n.Notes)
	b.WriteString("\n## Affected assets\n")
	assets := append([]store.NewsAsset(nil), in.Assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	if len(assets) == 0 {
		b.WriteString("\nNone.\n")
	}
	for _, a := range assets {
		fmt.Fprintf(&b, "\n- %s (%s); type: %s; market: %s; exchange: %s; source: %s; relation: %s; ID: %s\n", markdownText(a.Symbol), markdownText(a.DisplayName), markdownText(a.AssetType), markdownText(a.Market), markdownText(a.Exchange), markdownText(a.Source), markdownText(a.Relation), markdownText(a.ID))
	}
	b.WriteString("\n## Predictions\n")
	predictions := append([]NewsPrediction(nil), in.Predictions...)
	sort.Slice(predictions, func(i, j int) bool { return predictions[i].Prediction.ID < predictions[j].Prediction.ID })
	if len(predictions) == 0 {
		b.WriteString("\nPending — no predictions saved.\n")
	}
	for _, item := range predictions {
		p := item.Prediction
		source := "User"
		if p.Source == "ai" {
			source = "AI"
		}
		confidence := "(blank)"
		if p.Confidence.Valid {
			confidence = strconv.FormatInt(p.Confidence.Int64, 10)
		}
		fmt.Fprintf(&b, "\n### %s prediction: %s\n\n- Asset ID: %s\n- Direction: %s\n- Confidence: %s\n", source, markdownText(p.ID), markdownText(p.NewsAssetID), markdownText(p.Direction), confidence)
		newsText(&b, "Reasoning", p.Reasoning)
		newsText(&b, "Catalysts", p.Catalysts)
		newsText(&b, "Risks", p.Risks)
		newsText(&b, "Invalidation", p.Invalidation)
		b.WriteString("\n### Current validation by trading-day horizon\n")
		horizons := append([]int64(nil), item.Horizons...)
		sort.Slice(horizons, func(i, j int) bool { return horizons[i] < horizons[j] })
		for _, h := range horizons {
			status := "pending"
			for _, e := range item.Evaluations {
				if e.Current && int64(e.Result.Horizon) == h {
					status = e.Result.Status
					break
				}
			}
			fmt.Fprintf(&b, "\n- %d trading days: %s\n", h, markdownText(status))
		}
		evaluations := append([]predictionvalidation.Evaluation(nil), item.Evaluations...)
		sort.Slice(evaluations, func(i, j int) bool { return evaluations[i].ID < evaluations[j].ID })
		for _, e := range evaluations {
			r := e.Result
			fmt.Fprintf(&b, "\n#### Saved evaluation: %s\n\n- Current: %t\n- Horizon: %d trading days\n- Status: %s\n- Rules: %s\n- Calculated (UTC): %s\n- Reference price: %s\n- Outcome price: %s\n- Asset return (ratio): %s\n- Direction correct: %s\n- Benchmark status: %s\n- Benchmark return (ratio): %s\n- Excess return (ratio): %s\n- Excess direction correct: %s\n", markdownText(e.ID), e.Current, r.Horizon, markdownText(r.Status), markdownText(r.Rules), formatTime(r.CalculatedAt), optionalNumber(r.ReferencePrice), optionalNumber(r.OutcomePrice), optionalNumber(r.AssetReturn), optionalBool(r.DirectionCorrect), markdownText(r.BenchmarkStatus), optionalNumber(r.BenchmarkReturn), optionalNumber(r.ExcessReturn), optionalBool(r.ExcessDirectionCorrect))
			if r.Benchmark != nil {
				fmt.Fprintf(&b, "- Benchmark: %s / %s / %s\n", markdownText(r.Benchmark.Symbol), markdownText(r.Benchmark.Market), markdownText(r.Benchmark.Currency))
			}
			newsText(&b, "Validation explanation", r.Explanation)
			newsText(&b, "Benchmark explanation", r.BenchmarkReason)
		}
	}
	return b.String()
}
