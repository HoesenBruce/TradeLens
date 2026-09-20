package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
)

func TestPredictionsSQLite(t *testing.T) {
	q, done := newStore(t)
	defer done()
	testPredictions(t, q)
}

func TestPredictionsPostgres(t *testing.T) {
	url := os.Getenv("TM_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TM_TEST_DATABASE_URL for Postgres")
	}
	conn, err := db.Open(url)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn, db.DriverPostgres))
	testPredictions(t, store.NewForDriver(conn, db.DriverPostgres))
}

func testPredictions(t *testing.T, q store.Querier) {
	ctx := context.Background()
	uid := uuid.New().String()
	_, err := q.CreateUser(ctx, store.CreateUserParams{ID: uid, Email: uid + "@example.com", PasswordHash: "x"})
	require.NoError(t, err)
	n, err := q.CreateNews(ctx, store.CreateNewsParams{ID: uuid.New().String(), UserID: uid, Title: "Thesis", Source: "manual", PublishedAt: time.Now(), Tags: "[]"})
	require.NoError(t, err)
	a, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{ID: uuid.New().String(), NewsID: n.ID, UserID: uid, AssetType: "stock", Symbol: "285A", Source: "user"})
	require.NoError(t, err)
	p := store.CreatePredictionParams{ID: uuid.New().String(), NewsAssetID: a.ID, UserID: uid, Source: "user", Direction: "bullish", Confidence: sql.NullInt64{Int64: 80, Valid: true}, Reasoning: "Demand", Catalysts: "Launch", Risks: "Delay", Invalidation: "Cancelled"}
	user, err := q.CreatePrediction(ctx, p)
	require.NoError(t, err)
	for _, days := range []int64{1, 3, 5, 10, 20} {
		require.NoError(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: user.ID, TradingDays: days, UserID: uid}))
	}
	h, err := q.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: user.ID, UserID: uid})
	require.NoError(t, err)
	require.Len(t, h, 5)
	require.Error(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: user.ID, TradingDays: 2, UserID: uid}))
	require.Error(t, q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: user.ID, TradingDays: 1, UserID: uid}))
	p.ID = uuid.New().String()
	p.Source = "ai"
	p.Confidence = sql.NullInt64{}
	ai, err := q.CreatePrediction(ctx, p)
	require.NoError(t, err)
	require.False(t, ai.Confidence.Valid)
	rows, err := q.ListPredictions(ctx, store.ListPredictionsParams{NewsID: n.ID, UserID: uid})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	_, err = q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: ai.ID, UserID: uid, Source: "user", Direction: "bearish"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	updated, err := q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: user.ID, UserID: uid, Source: "user", Direction: "neutral", Reasoning: "Updated"})
	require.NoError(t, err)
	require.Equal(t, "neutral", updated.Direction)
	got, err := q.GetPrediction(ctx, store.GetPredictionParams{ID: user.ID, UserID: uid})
	require.NoError(t, err)
	require.Equal(t, "Updated", got.Reasoning)
	_, err = q.GetPrediction(ctx, store.GetPredictionParams{ID: user.ID, UserID: "other"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	for _, confidence := range []int64{-1, 101} {
		p.ID = uuid.New().String()
		p.Confidence = sql.NullInt64{Int64: confidence, Valid: true}
		_, err = q.CreatePrediction(ctx, p)
		require.Error(t, err)
	}
	p.Confidence = sql.NullInt64{}
	p.Direction = "up"
	_, err = q.CreatePrediction(ctx, p)
	require.Error(t, err)
	p.Direction = "bullish"
	p.Source = "invalid"
	_, err = q.CreatePrediction(ctx, p)
	require.Error(t, err)
	for _, confidence := range []int64{0, 100} {
		p.ID = uuid.New().String()
		p.Source = "user"
		p.Confidence = sql.NullInt64{Int64: confidence, Valid: true}
		_, err = q.CreatePrediction(ctx, p)
		require.NoError(t, err)
	}
	p.Source = "user"
	p.UserID = "other"
	_, err = q.CreatePrediction(ctx, p)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, q.DeletePredictionHorizons(ctx, store.DeletePredictionHorizonsParams{PredictionID: user.ID, UserID: uid}))
	h, err = q.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: user.ID, UserID: uid})
	require.NoError(t, err)
	require.Empty(t, h)
	count, err := q.DeletePrediction(ctx, store.DeletePredictionParams{ID: ai.ID, UserID: uid, Source: "user"})
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = q.DeletePrediction(ctx, store.DeletePredictionParams{ID: user.ID, UserID: uid, Source: "user"})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	_, err = q.DeleteNews(ctx, store.DeleteNewsParams{ID: n.ID, UserID: uid})
	require.NoError(t, err)
	_, err = q.GetPrediction(ctx, store.GetPredictionParams{ID: ai.ID, UserID: uid})
	require.ErrorIs(t, err, sql.ErrNoRows)
}
