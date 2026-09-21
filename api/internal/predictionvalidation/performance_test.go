package predictionvalidation

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
	"path/filepath"
	"testing"
	"time"
	"uuid"
)

func TestPerformanceCurrentDenominatorsAndFilters(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "performance.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn, db.DriverSQLite))
	q := store.NewForDriver(conn, db.DriverSQLite)
	e := Engine{Store: q}
	owner := uuid.New().String()
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: owner, Email: "performance@test.example", PasswordHash: "x"})
	require.NoError(t, err)
	makePrediction := func(source, category, symbol, date string, horizons []int64) (string, string) {
		n, err := q.CreateNews(ctx, store.CreateNewsParams{ID: uuid.New().String(), UserID: owner, Title: "news", Source: "manual", PublishedAt: instant(date), Category: category, Tags: "[]"})
		require.NoError(t, err)
		a, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{ID: uuid.New().String(), NewsID: n.ID, UserID: owner, Source: source, AssetType: "stock", Symbol: symbol, Market: "JP"})
		require.NoError(t, err)
		p, err := q.CreatePrediction(ctx, store.CreatePredictionParams{ID: uuid.New().String(), NewsAssetID: a.ID, UserID: owner, Source: source, Direction: "bullish"})
		require.NoError(t, err)
		for _, h := range horizons {
			require.NoError(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: p.ID, UserID: owner, TradingDays: h}))
		}
		return n.ID, p.ID
	}
	n, p := makePrediction("user", "Industry", "285A", "2026-09-01T23:30:00Z", []int64{1, 3, 5, 10, 20})
	an, ap := makePrediction("ai", "Macro", "1306", "2026-09-02T00:30:00Z", []int64{1, 5})
	stamp := instant("2026-10-01T00:00:00Z")
	save := func(n, p string, h int, status string, correct *bool) {
		in, err := e.Load(ctx, owner, n, p)
		require.NoError(t, err)
		r := Result{Input: in, Revision: Revision(in), Rules: RulesVersion, Horizon: h, Status: status, DirectionCorrect: correct}
		raw, err := json.Marshal(r)
		require.NoError(t, err)
		stamp = stamp.Add(time.Second)
		_, err = q.SavePredictionEvaluation(ctx, store.SavePredictionEvaluationParams{ID: uuid.New().String(), UserID: owner, PredictionID: p, PredictionRevision: r.Revision, Horizon: int64(h), Fingerprint: uuid.New().String(), ResultJson: string(raw), AttemptedAt: stamp.Format("2006-01-02T15:04:05.000000000Z")})
		require.NoError(t, err)
	}
	yes, no := true, false
	save(n, p, 1, "validated", &yes)
	save(n, p, 3, "validated", &no)
	save(n, p, 10, "unavailable", nil)
	save(n, p, 20, "incomplete", nil)
	save(an, ap, 1, "validated", &no)
	got, err := e.Performance(ctx, owner, PerformanceFilter{})
	require.NoError(t, err)
	require.Equal(t, PerformanceCounts{Total: 7, Pending: 2, Validated: 3, Unavailable: 1, Incomplete: 1}, got.Counts)
	require.Len(t, got.BySource, 2)
	require.Equal(t, "ai", got.BySource[0].Source)
	require.Equal(t, 1, got.BySource[0].SampleCount)
	require.Equal(t, 0.0, *got.BySource[0].HitRate)
	require.Equal(t, 2, got.BySource[1].SampleCount)
	require.Equal(t, 50.0, *got.BySource[1].HitRate)
	require.Len(t, got.ByAsset, 2)
	require.Len(t, got.ByCategory, 2)
	require.Len(t, got.ByHorizon, 7)
	again, err := e.Performance(ctx, owner, PerformanceFilter{})
	require.NoError(t, err)
	require.Equal(t, got, again)
	for _, f := range []PerformanceFilter{{Source: "ai"}, {Symbol: "1306"}, {Category: "Macro"}, {From: "2026-09-02", To: "2026-09-02"}} {
		filtered, err := e.Performance(ctx, owner, f)
		require.NoError(t, err)
		require.Equal(t, 2, filtered.Counts.Total)
		require.Len(t, filtered.BySource, 1)
		require.Equal(t, "ai", filtered.BySource[0].Source)
	}
	filtered, err := e.Performance(ctx, owner, PerformanceFilter{Source: "user", Symbol: "285a", Category: "Industry", AssetType: "stock", Horizon: 1, From: "2026-09-01", To: "2026-09-01"})
	require.NoError(t, err)
	require.Equal(t, 1, filtered.Counts.Total)
	require.Equal(t, 100.0, *filtered.BySource[0].HitRate)
	empty, err := e.Performance(ctx, owner, PerformanceFilter{AssetType: "etf"})
	require.NoError(t, err)
	require.Zero(t, empty.Counts.Total)
	require.NotNil(t, empty.BySource)
	empty, err = e.Performance(ctx, "other", PerformanceFilter{})
	require.NoError(t, err)
	require.Zero(t, empty.Counts.Total)
	// A newer failed attempt retracts an older hit; retained history must not double count.
	save(n, p, 1, "unavailable", nil)
	got, err = e.Performance(ctx, owner, PerformanceFilter{Source: "user"})
	require.NoError(t, err)
	require.Equal(t, 1, got.BySource[0].SampleCount)
	require.Equal(t, 0.0, *got.BySource[0].HitRate)
	// Editing invalidates all results for the previous revision.
	_, err = q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: p, UserID: owner, Source: "user", Direction: "bearish"})
	require.NoError(t, err)
	got, err = e.Performance(ctx, owner, PerformanceFilter{Source: "user"})
	require.NoError(t, err)
	require.Equal(t, 5, got.Counts.Pending)
	require.Nil(t, got.BySource[0].HitRate)
	require.Zero(t, got.BySource[0].SampleCount)
	// A corrupt finalized result never contributes a percentage.
	save(n, p, 1, "validated", nil)
	got, err = e.Performance(ctx, owner, PerformanceFilter{Source: "user"})
	require.NoError(t, err)
	require.Equal(t, 1, got.Counts.Unavailable)
	require.Nil(t, got.BySource[0].HitRate)
	for _, f := range []PerformanceFilter{{Source: "bad"}, {AssetType: "coin"}, {Horizon: 2}, {From: "yesterday"}, {From: "2026-09-02", To: "2026-09-01"}} {
		_, err = e.Performance(ctx, owner, f)
		require.Error(t, err)
	}
}
