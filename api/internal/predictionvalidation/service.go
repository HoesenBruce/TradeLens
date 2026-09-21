package predictionvalidation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"
	"uuid"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

type Evaluation struct {
	ID          string `json:"id"`
	PreviousID  string `json:"previous_id"`
	Fingerprint string `json:"fingerprint"`
	LastAttempt string `json:"last_attempt_at"`
	Current     bool   `json:"current"`
	Result      Result `json:"result"`
}

func (e *Engine) Load(ctx context.Context, owner, newsID, predictionID string) (Input, error) {
	p, err := e.Store.GetPrediction(ctx, store.GetPredictionParams{ID: predictionID, UserID: owner})
	if err != nil {
		return Input{}, err
	}
	a, err := e.Store.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: p.NewsAssetID, UserID: owner})
	if err != nil {
		return Input{}, err
	}
	if a.NewsID != newsID {
		return Input{}, sql.ErrNoRows
	}
	n, err := e.Store.GetNews(ctx, store.GetNewsParams{ID: newsID, UserID: owner})
	if err != nil {
		return Input{}, err
	}
	revision, err := e.Store.GetPredictionRevision(ctx, store.GetPredictionRevisionParams{PredictionID: predictionID, UserID: owner})
	if err != nil {
		return Input{}, err
	}
	return Input{Prediction: p, Asset: a, PublishedAt: n.PublishedAt, OwnerID: owner, RevisionNumber: revision}, nil
}

func (e *Engine) Validate(ctx context.Context, owner, newsID, predictionID string, now time.Time, benchmark ...*Benchmark) ([]Evaluation, error) {
	in, err := e.Load(ctx, owner, newsID, predictionID)
	if err != nil {
		return nil, err
	}
	horizons, err := e.Store.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: predictionID, UserID: owner})
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(horizons))
	for _, h := range horizons {
		result := e.Evaluate(ctx, in, int(h.TradingDays), now)
		if len(benchmark) > 0 {
			result = e.WithBenchmark(ctx, result, benchmark[0], now)
		}
		results = append(results, result)
	}
	out := make([]Evaluation, 0, len(results))
	err = store.InTx(ctx, e.Store, func(q store.Querier) error {
		previous, err := q.ListPredictionEvaluations(ctx, store.ListPredictionEvaluationsParams{PredictionID: predictionID, UserID: owner})
		if err != nil {
			return err
		}
		for _, r := range results {
			raw, err := json.Marshal(r)
			if err != nil {
				return err
			}
			fingerprint, err := Fingerprint(r)
			if err != nil {
				return err
			}
			previousID := ""
			for _, p := range previous {
				if p.PredictionRevision == r.Revision && p.Horizon == int64(r.Horizon) {
					previousID = p.ID
					break
				}
			}
			saved, err := q.SavePredictionEvaluation(ctx, store.SavePredictionEvaluationParams{ID: uuid.New().String(), PredictionID: predictionID, PredictionRevision: r.Revision, Horizon: int64(r.Horizon), Fingerprint: fingerprint, PreviousID: previousID, ResultJson: string(raw), AttemptedAt: now.UTC().Format("2006-01-02T15:04:05.000000000Z"), UserID: owner})
			if err != nil {
				return err
			}
			item, err := evaluation(saved)
			if err != nil {
				return err
			}
			item.Current = true
			out = append(out, item)
		}
		return nil
	})
	return out, err
}

// Fingerprint excludes acquisition/attempt clocks, but includes all price/rule/source evidence
// and resulting state. A refresh of unchanged evidence does not create another contribution.
func Fingerprint(r Result) (string, error) {
	if r.BenchmarkEvidence != nil {
		snapshot := *r.BenchmarkEvidence
		clearEvidenceClocks(&snapshot)
		r.BenchmarkEvidence = &snapshot
	}
	clearEvidenceClocks(&r)
	raw, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func evaluation(row store.PredictionEvaluation) (Evaluation, error) {
	out := Evaluation{ID: row.ID, PreviousID: row.PreviousID, Fingerprint: row.Fingerprint, LastAttempt: row.AttemptedAt}
	err := json.Unmarshal([]byte(row.ResultJson), &out.Result)
	if out.Result.BenchmarkStatus == "" {
		out.Result.BenchmarkStatus = "not_requested"
	}
	return out, err
}
func (e *Engine) History(ctx context.Context, owner, newsID, predictionID string) ([]Evaluation, error) {
	in, err := e.Load(ctx, owner, newsID, predictionID)
	if err != nil {
		return nil, err
	}
	rows, err := e.Store.ListPredictionEvaluations(ctx, store.ListPredictionEvaluationsParams{PredictionID: predictionID, UserID: owner})
	if err != nil {
		return nil, err
	}
	horizons, err := e.Store.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: predictionID, UserID: owner})
	if err != nil {
		return nil, err
	}
	selected := map[int]bool{}
	for _, h := range horizons {
		selected[int(h.TradingDays)] = true
	}
	seen := map[int]bool{}
	revision := Revision(in)
	out := make([]Evaluation, 0, len(rows))
	for _, row := range rows {
		item, err := evaluation(row)
		if err != nil {
			return nil, err
		}
		h := item.Result.Horizon
		item.Current = row.PredictionRevision == revision && selected[h] && !seen[h]
		if item.Current {
			seen[h] = true
		}
		out = append(out, item)
	}
	return out, nil
}

func clearEvidenceClocks(r *Result) {
	r.CalculatedAt = time.Time{}
	if r.Evidence != nil {
		copyResponse := *r.Evidence
		copyResponse.FetchedAt = nil
		copyResponse.Cached = false
		copyResponse.Bars = append([]marketdata.Bar(nil), copyResponse.Bars...)
		for i := range copyResponse.Bars {
			copyResponse.Bars[i].FetchedAt = nil
		}
		r.Evidence = &copyResponse
	}
}
