package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
	"uuid"
)

func TestNewsAndAssetsRoundTripSQLite(t *testing.T) {
	q, done := newStore(t)
	defer done()
	testNewsAndAssetsRoundTrip(t, q)
}

func TestNewsAndAssetsRoundTripPostgres(t *testing.T) {
	url := os.Getenv("TM_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TM_TEST_DATABASE_URL to run Postgres store tests")
	}
	conn, err := db.Open(url)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn, db.DriverPostgres))
	testNewsAndAssetsRoundTrip(t, store.NewForDriver(conn, db.DriverPostgres))
}

func testNewsAndAssetsRoundTrip(t *testing.T, q store.Querier) {
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		ID: uuid.New().String(), Email: "news@example.com", PasswordHash: "x",
	})
	require.NoError(t, err)
	published := time.Date(2026, 9, 20, 9, 30, 0, 0, time.UTC)
	item, err := q.CreateNews(ctx, store.CreateNewsParams{
		ID: uuid.New().String(), UserID: user.ID, Title: "Japan ETF launch", Source: "manual",
		PublishedAt: published, OriginalText: "Original", Notes: "Watch flows", Tags: `["ETF"]`,
	})
	require.NoError(t, err)
	item, err = q.UpdateNews(ctx, store.UpdateNewsParams{
		Title: "Japan ETF launched", Source: item.Source, Url: item.Url, PublishedAt: item.PublishedAt,
		OriginalText: item.OriginalText, Notes: item.Notes, Summary: "New listing",
		Category: "markets", Tags: item.Tags, ID: item.ID, UserID: user.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "Japan ETF launched", item.Title)
	items, err := q.ListNews(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, item.ID, items[0].ID)

	first, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{
		ID: uuid.New().String(), NewsID: item.ID, AssetType: "stock", Symbol: "285A",
		Market: "JP", Source: "user", UserID: user.ID,
	})
	require.NoError(t, err)
	second, err := q.CreateNewsAsset(ctx, store.CreateNewsAssetParams{
		ID: uuid.New().String(), NewsID: item.ID, AssetType: "index", Symbol: "N225",
		Source: "user", UserID: user.ID,
	})
	require.NoError(t, err)

	assets, err := q.ListNewsAssets(ctx, store.ListNewsAssetsParams{NewsID: item.ID, UserID: user.ID})
	require.NoError(t, err)
	require.Len(t, assets, 2)
	require.ElementsMatch(t, []string{"285A", "N225"}, []string{assets[0].Symbol, assets[1].Symbol})

	updated, err := q.UpdateNewsAsset(ctx, store.UpdateNewsAssetParams{
		AssetType: first.AssetType, Symbol: first.Symbol, Market: first.Market,
		Exchange: "TSE", DisplayName: "Global X Japan", Relation: "affected",
		Source: "user", ID: first.ID, UserID: user.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "TSE", updated.Exchange)
	rows, err := q.DeleteNewsAsset(ctx, store.DeleteNewsAssetParams{ID: second.ID, UserID: user.ID})
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)

	rows, err = q.DeleteNews(ctx, store.DeleteNewsParams{ID: item.ID, UserID: user.ID})
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)
	_, err = q.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: first.ID, UserID: user.ID})
	require.ErrorIs(t, err, sql.ErrNoRows)
}
