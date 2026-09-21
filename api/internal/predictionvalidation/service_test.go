package predictionvalidation

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	conn, err := db.Open(target)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn, driver))

	q := store.NewForDriver(conn, driver)
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "u@x.test", PasswordHash: "x"})
	require.NoError(t, err)
	n, err := q.CreateNews(ctx, store.CreateNewsParams{ID: "n", UserID: "u", Title: "original", Source: "manual", PublishedAt: instant("2026-09-01T00:00:00Z"), Tags: "[]"})
	require.NoError(t, err)
	a, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{ID: "a", NewsID: n.ID, UserID: "u", AssetType: "stock", Symbol: "285A", Market: "JP", Source: "user"})
	require.NoError(t, err)
	p, err := q.CreatePrediction(ctx, store.CreatePredictionParams{ID: "p", NewsAssetID: a.ID, UserID: "u", Source: "user", Direction: "bullish", Reasoning: "original reasoning"})
	require.NoError(t, err)
	_, err = conn.Exec("UPDATE predictions SET created_at=$1,updated_at=$2 WHERE id='p'", instant("2026-09-13T00:00:00Z"), instant("2026-09-13T00:00:00Z"))
	require.NoError(t, err)
	for _, h := range []int64{1, 3, 5, 10, 20} {
		require.NoError(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: p.ID, UserID: "u", TradingDays: h}))
	}
	provider := &evidenceProvider{name: "http"}
	e := Engine{Store: q, Market: marketdata.NewService(q, provider)}
	now := instant("2026-12-01T00:00:00Z")
	first, err := e.Validate(ctx, "u", "n", "p", now)
	require.NoError(t, err)
	require.Len(t, first, 5)
	require.Equal(t, "validated", first[0].Result.Status)
	again, err := e.Validate(ctx, "u", "n", "p", now.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, first[0].ID, again[0].ID)
	history, err := e.History(ctx, "u", "n", "p")
	require.NoError(t, err)
	require.Len(t, history, 5)
	provider.mutate = func(r *marketdata.Response) { r.Bars = r.Bars[1:] }
	revised, err := e.Validate(ctx, "u", "n", "p", now.Add(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "incomplete", revised[0].Result.Status)
	require.Equal(t, first[0].ID, revised[0].PreviousID)
	history, err = e.History(ctx, "u", "n", "p")
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
	original, err := q.GetPrediction(ctx, store.GetPredictionParams{ID: "p", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, "original reasoning", original.Reasoning)
	beforeRevision, err := q.GetPredictionRevision(ctx, store.GetPredictionRevisionParams{PredictionID: "p", UserID: "u"})
	require.NoError(t, err)
	_, err = e.Validate(ctx, "other", "n", "p", now)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = e.History(ctx, "u", "wrong-news", "p")
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: "p", UserID: "u", Source: "user", Direction: "bearish", Reasoning: "edited"})
	require.NoError(t, err)
	afterRevision, err := q.GetPredictionRevision(ctx, store.GetPredictionRevisionParams{PredictionID: "p", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, beforeRevision+1, afterRevision)
	history, err = e.History(ctx, "u", "n", "p")
	require.NoError(t, err)
	for _, item := range history {
		require.False(t, item.Current)
	}
	_, err = q.DeletePrediction(ctx, store.DeletePredictionParams{ID: "p", UserID: "u", Source: "user"})
	require.NoError(t, err)
	var count int
	require.NoError(t, conn.QueryRow("SELECT count(*) FROM prediction_evaluations").Scan(&count))
	require.Zero(t, count)
}
