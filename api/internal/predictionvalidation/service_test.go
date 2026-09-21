package predictionvalidation

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

func TestPersistenceRevisionsIsolationAndRetraction(t *testing.T) {
	testPersistence(t, filepath.Join(t.TempDir(), "validation.db"), db.DriverSQLite)
}
func TestPersistencePostgres(t *testing.T) {
	url := os.Getenv("TM_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TM_TEST_DATABASE_URL for Postgres")
	}
	testPersistence(t, url, db.DriverPostgres)
}
func testPersistence(t *testing.T, target, driver string) {
	ctx := context.Background()
	uid, nid, aid, pid := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	conn, err := db.Open(target)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn, driver))

	q := store.NewForDriver(conn, driver)
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: uid, Email: uid + "@test.example", PasswordHash: "x"})
	require.NoError(t, err)
	n, err := q.CreateNews(ctx, store.CreateNewsParams{ID: nid, UserID: uid, Title: "original", Source: "manual", PublishedAt: instant("2026-09-01T00:00:00Z"), Tags: "[]"})
	require.NoError(t, err)
	a, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{ID: aid, NewsID: n.ID, UserID: uid, AssetType: "stock", Symbol: "285A", Market: "JP", Source: "user"})
	require.NoError(t, err)
	p, err := q.CreatePrediction(ctx, store.CreatePredictionParams{ID: pid, NewsAssetID: a.ID, UserID: uid, Source: "user", Direction: "bullish", Reasoning: "original reasoning"})
	require.NoError(t, err)
	_, err = conn.Exec("UPDATE predictions SET created_at=$1,updated_at=$2 WHERE id=$3", instant("2026-09-13T00:00:00Z"), instant("2026-09-13T00:00:00Z"), pid)
	require.NoError(t, err)
	for _, h := range []int64{1, 3, 5, 10, 20} {
		require.NoError(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: p.ID, UserID: uid, TradingDays: h}))
	}
	provider := &evidenceProvider{name: "http"}
	e := Engine{Store: q, Market: marketdata.NewService(q, provider)}
	now := instant("2026-12-01T00:00:00Z")
	first, err := e.Validate(ctx, uid, nid, pid, now)
	require.NoError(t, err)
	require.Len(t, first, 5)
	require.Equal(t, "validated", first[0].Result.Status)
	again, err := e.Validate(ctx, uid, nid, pid, now.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, first[0].ID, again[0].ID)
	history, err := e.History(ctx, uid, nid, pid)
	require.NoError(t, err)
	require.Len(t, history, 5)
	provider.mutate = func(r *marketdata.Response) { r.Bars = r.Bars[1:] }
	revised, err := e.Validate(ctx, uid, nid, pid, now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "incomplete", revised[0].Result.Status)
	require.Equal(t, first[0].ID, revised[0].PreviousID)
	history, err = e.History(ctx, uid, nid, pid)
	require.NoError(t, err)
	require.Len(t, history, 10)
	current := 0
	for _, item := range history {
		if item.Current {
			current++
			require.Equal(t, "incomplete", item.Result.Status)
		}
		raw, err := json.Marshal(item)
		require.NoError(t, err)
		require.Contains(t, string(raw), "original reasoning")
	}
	require.Equal(t, 5, current)
	original, err := q.GetPrediction(ctx, store.GetPredictionParams{ID: pid, UserID: uid})
	require.NoError(t, err)
	require.Equal(t, "original reasoning", original.Reasoning)
	beforeRevision, err := q.GetPredictionRevision(ctx, store.GetPredictionRevisionParams{PredictionID: pid, UserID: uid})
	require.NoError(t, err)
	_, err = e.Validate(ctx, "other", nid, pid, now)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = e.History(ctx, uid, "wrong-news", pid)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: pid, UserID: uid, Source: "user", Direction: "bearish", Reasoning: "edited"})
	require.NoError(t, err)
	afterRevision, err := q.GetPredictionRevision(ctx, store.GetPredictionRevisionParams{PredictionID: pid, UserID: uid})
	require.NoError(t, err)
	require.Equal(t, beforeRevision+1, afterRevision)
	history, err = e.History(ctx, uid, nid, pid)
	require.NoError(t, err)
	for _, item := range history {
		require.False(t, item.Current)
	}
	_, err = conn.Exec("UPDATE predictions SET updated_at=$1 WHERE id=$2", instant("2026-09-13T01:00:00Z"), pid)
	require.NoError(t, err)
	provider.mutate = func(r *marketdata.Response) {
		stamp := now.Add(-time.Hour).Add(time.Duration(provider.calls) * time.Second)
		r.FetchedAt = &stamp
		if r.Symbol == "1306" {
			for i := range r.Bars {
				r.Bars[i].Close = 103
			}
		}
	}
	selection := &Benchmark{Symbol: "1306", Market: "JP", Currency: "JPY"}
	withBenchmark, err := e.Validate(ctx, uid, nid, pid, now.Add(3*time.Second), selection)
	require.NoError(t, err)
	require.Equal(t, "validated", withBenchmark[0].Result.BenchmarkStatus)
	require.Equal(t, -.01, *withBenchmark[0].Result.ExcessReturn)
	sameBenchmark, err := e.Validate(ctx, uid, nid, pid, now.Add(4*time.Second), selection)
	require.NoError(t, err)
	require.Equal(t, withBenchmark[0].ID, sameBenchmark[0].ID)
	reset, err := e.Validate(ctx, uid, nid, pid, now.Add(5*time.Second), nil)
	require.NoError(t, err)
	require.Equal(t, "not_requested", reset[0].Result.BenchmarkStatus)
	require.Nil(t, reset[0].Result.ExcessReturn)
	history, err = e.History(ctx, uid, nid, pid)
	require.NoError(t, err)
	for _, item := range history {
		if item.Current {
			require.Equal(t, "not_requested", item.Result.BenchmarkStatus)
		}
	}
	_, err = q.DeletePrediction(ctx, store.DeletePredictionParams{ID: pid, UserID: uid, Source: "user"})
	require.NoError(t, err)
	var count int
	require.NoError(t, conn.QueryRow("SELECT count(*) FROM prediction_evaluations WHERE prediction_id=$1", pid).Scan(&count))
	require.Zero(t, count)
}
