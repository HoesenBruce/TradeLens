package predictionvalidation

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/store"
)

// PerformanceFilter dates select news publication dates in UTC, inclusive.
type PerformanceFilter struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	AssetType string `json:"asset_type"`
	Category  string `json:"category"`
	Horizon   int64  `json:"horizon"`
	From      string `json:"from"`
	To        string `json:"to"`
}

func (f PerformanceFilter) Validate() error {
	if f.Source != "" && f.Source != "ai" && f.Source != "user" {
		return fmt.Errorf("source must be ai or user")
	}
	if f.AssetType != "" && f.AssetType != "stock" && f.AssetType != "etf" && f.AssetType != "index" {
		return fmt.Errorf("invalid asset_type")
	}
	if f.Horizon != 0 && f.Horizon != 1 && f.Horizon != 3 && f.Horizon != 5 && f.Horizon != 10 && f.Horizon != 20 {
		return fmt.Errorf("invalid trading-day horizon")
	}
	for _, v := range []string{f.From, f.To} {
		if v != "" {
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return fmt.Errorf("dates must use YYYY-MM-DD")
			}
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return fmt.Errorf("from must not be after to")
	}
	return nil
}

// MatchesNews uses the report's inclusive UTC publication-date and exact category semantics.
func (f PerformanceFilter) MatchesNews(n store.News) bool {
	date := n.PublishedAt.UTC().Format("2006-01-02")
	return (f.From == "" || date >= f.From) && (f.To == "" || date <= f.To) && (f.Category == "" || n.Category == f.Category)
}

func (f PerformanceFilter) MatchesAsset(a store.NewsAsset) bool {
	symbol := strings.ToUpper(strings.TrimSpace(f.Symbol))
	return (symbol == "" || a.Symbol == symbol) && (f.AssetType == "" || a.AssetType == f.AssetType)
}

func (f PerformanceFilter) MatchesPrediction(p store.Prediction) bool {
	return f.Source == "" || p.Source == f.Source
}

func (f PerformanceFilter) MatchesHorizon(h int64) bool {
	return f.Horizon == 0 || h == f.Horizon
}

type PerformanceCounts struct {
	Total       int `json:"total"`
	Pending     int `json:"pending"`
	Validated   int `json:"validated"`
	Unavailable int `json:"unavailable"`
	Incomplete  int `json:"incomplete"`
}
type PerformanceGroup struct {
	Key    string `json:"key"`
	Source string `json:"source"`
	PerformanceCounts
	Correct     int      `json:"correct"`
	SampleCount int      `json:"sample_count"`
	HitRate     *float64 `json:"hit_rate"` // percentage; null when no finalized samples
}
type Performance struct {
	Unit       string             `json:"unit"`
	Filters    PerformanceFilter  `json:"filters"`
	Counts     PerformanceCounts  `json:"counts"`
	BySource   []PerformanceGroup `json:"by_source"`
	ByHorizon  []PerformanceGroup `json:"by_horizon"`
	ByAsset    []PerformanceGroup `json:"by_asset"`
	ByCategory []PerformanceGroup `json:"by_category"`
}

func (c *PerformanceCounts) add(status string) {
	c.Total++
	switch status {
	case "validated":
		c.Validated++
	case "pending":
		c.Pending++
	case "incomplete":
		c.Incomplete++
	default:
		c.Unavailable++
	}
}
func addPerformanceGroup(groups map[[2]string]*PerformanceGroup, key, source, status string, correct bool) {
	k := [2]string{key, source}
	g := groups[k]
	if g == nil {
		g = &PerformanceGroup{Key: key, Source: source}
		groups[k] = g
	}
	g.add(status)
	if status == "validated" {
		g.SampleCount++
		if correct {
			g.Correct++
		}
	}
}
func performanceGroups(groups map[[2]string]*PerformanceGroup) []PerformanceGroup {
	out := make([]PerformanceGroup, 0, len(groups))
	for _, g := range groups {
		if g.SampleCount > 0 {
			rate := 100 * float64(g.Correct) / float64(g.SampleCount)
			g.HitRate = &rate
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key == out[j].Key {
			return out[i].Source < out[j].Source
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// Performance reads persisted current outcomes only; it never calls the price provider.
func (e *Engine) Performance(ctx context.Context, owner string, f PerformanceFilter) (Performance, error) {
	f.Symbol = strings.ToUpper(strings.TrimSpace(f.Symbol))
	out := Performance{Unit: "prediction_horizon", Filters: f}
	if err := f.Validate(); err != nil {
		return out, err
	}
	// ponytail: per-prediction history reads suit personal journals; add a batched store query if volume makes this slow.
	err := store.InTx(ctx, e.Store, func(q store.Querier) error {
		reader := Engine{Store: q}
		news, err := q.ListNews(ctx, owner)
		if err != nil {
			return err
		}
		bySource, byHorizon, byAsset, byCategory := map[[2]string]*PerformanceGroup{}, map[[2]string]*PerformanceGroup{}, map[[2]string]*PerformanceGroup{}, map[[2]string]*PerformanceGroup{}
		for _, n := range news {
			if !f.MatchesNews(n) {
				continue
			}
			assets, err := q.ListNewsAssets(ctx, store.ListNewsAssetsParams{NewsID: n.ID, UserID: owner})
			if err != nil {
				return err
			}
			assetMap := map[string]store.NewsAsset{}
			for _, a := range assets {
				assetMap[a.ID] = a
			}
			predictions, err := q.ListPredictions(ctx, store.ListPredictionsParams{NewsID: n.ID, UserID: owner})
			if err != nil {
				return err
			}
			for _, p := range predictions {
				a := assetMap[p.NewsAssetID]
				if !f.MatchesPrediction(p) || !f.MatchesAsset(a) {
					continue
				}
				history, err := reader.History(ctx, owner, n.ID, p.ID)
				if err != nil {
					return err
				}
				current := map[int]Result{}
				for _, v := range history {
					if v.Current {
						current[v.Result.Horizon] = v.Result
					}
				}
				horizons, err := q.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: p.ID, UserID: owner})
				if err != nil {
					return err
				}
				for _, h := range horizons {
					if !f.MatchesHorizon(h.TradingDays) {
						continue
					}
					status, correct := "pending", false
					if r, ok := current[int(h.TradingDays)]; ok {
						status = r.Status
						// Malformed or obsolete rule results cannot become a finalized denominator.
						if r.Rules != RulesVersion || (status == "validated" && r.DirectionCorrect == nil) {
							status = "unavailable"
						}
						if r.DirectionCorrect != nil {
							correct = *r.DirectionCorrect
						}
					}
					out.Counts.add(status)
					addPerformanceGroup(bySource, p.Source, p.Source, status, correct)
					addPerformanceGroup(byHorizon, strconv.FormatInt(h.TradingDays, 10), p.Source, status, correct)
					addPerformanceGroup(byAsset, strings.Join([]string{a.AssetType, a.Market, a.Exchange, a.Symbol}, " / "), p.Source, status, correct)
					addPerformanceGroup(byCategory, n.Category, p.Source, status, correct)
				}
			}
		}
		out.BySource = performanceGroups(bySource)
		out.ByHorizon = performanceGroups(byHorizon)
		sort.SliceStable(out.ByHorizon, func(i, j int) bool {
			a, _ := strconv.Atoi(out.ByHorizon[i].Key)
			b, _ := strconv.Atoi(out.ByHorizon[j].Key)
			return a < b
		})
		out.ByAsset = performanceGroups(byAsset)
		out.ByCategory = performanceGroups(byCategory)
		return nil
	})
	return out, err
}
