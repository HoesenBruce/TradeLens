package marketdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
)

type recordingProvider struct {
	requests []Request
}

func (p *recordingProvider) Name() string { return "test" }

func (p *recordingProvider) FetchBars(_ context.Context, req Request) ([]Bar, error) {
	p.requests = append(p.requests, req)
	return []Bar{{Time: req.From.Unix(), Close: 1}, {Time: req.To.Unix(), Close: 2}}, nil
}

func TestTransactionDailyCoverageIsIncremental(t *testing.T) {
	t.Run("forward only", func(t *testing.T) {
		service, provider, q := coverageFixture(t)
		today := marketDayOffset(dailyRequest(time.Now()), time.Now(), 0)
		seedCoverage(t, q, today.AddDate(0, 0, -365), today)

		_, err := service.GetTransactionBars(context.Background(), dailyRequest(today), today)
		require.NoError(t, err)
		require.Len(t, provider.requests, 1)
		require.Equal(t, today, provider.requests[0].From)
		require.Equal(t, today.AddDate(0, 0, 1), provider.requests[0].To)
	})

	t.Run("historical backfill and overlap dedupe", func(t *testing.T) {
		service, provider, q := coverageFixture(t)
		today := marketDayOffset(dailyRequest(time.Now()), time.Now(), 0)
		cachedFrom := today.AddDate(0, 0, -100)
		seedCoverage(t, q, cachedFrom, today.AddDate(0, 0, 1))

		earliest := today.AddDate(0, 0, -10)
		response, err := service.GetTransactionBars(context.Background(), dailyRequest(today), earliest)
		require.NoError(t, err)
		require.Len(t, provider.requests, 1)
		require.Equal(t, earliest.AddDate(0, 0, -DefaultDailyMarketDataLookbackDays), provider.requests[0].From)
		require.Equal(t, cachedFrom, provider.requests[0].To)

		cached, err := q.GetMarketBarsCache(context.Background(), coverageKey)
		require.NoError(t, err)
		var bars []Bar
		require.NoError(t, json.Unmarshal(cached.BarsJson, &bars))
		require.Len(t, bars, 3, "the overlapping boundary bar must be stored once")
		require.Equal(t, bars, response.Bars, "the chart must receive cached history so it can pan backwards")
	})

	t.Run("complete coverage does not fetch", func(t *testing.T) {
		service, provider, q := coverageFixture(t)
		today := marketDayOffset(dailyRequest(time.Now()), time.Now(), 0)
		seedCoverage(t, q, today.AddDate(0, 0, -365), today.AddDate(0, 0, 1))

		_, err := service.GetTransactionBars(context.Background(), dailyRequest(today), today)
		require.NoError(t, err)
		require.Empty(t, provider.requests)
	})
}

const coverageKey = "daily-coverage-v1|7203|stock"

func coverageFixture(t *testing.T) (*Service, *recordingProvider, *store.Queries) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "coverage.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, db.Migrate(conn))
	provider := &recordingProvider{}
	q := store.New(conn)
	return NewService(q, provider), provider, q
}

func dailyRequest(today time.Time) Request {
	return Request{
		Symbol: "7203", InstrumentType: "stock", Interval: "D",
		From: today.AddDate(0, 0, -2), To: today,
	}
}

func seedCoverage(t *testing.T, q *store.Queries, from, to time.Time) {
	t.Helper()
	bars, err := json.Marshal([]Bar{{Time: from.Unix(), Close: 1}, {Time: to.Unix(), Close: 2}})
	require.NoError(t, err)
	require.NoError(t, q.UpsertMarketBarsCache(context.Background(), store.UpsertMarketBarsCacheParams{
		CacheKey: coverageKey, Symbol: "7203", Interval: "D",
		FromTs: FormatTimeRFC3339(from), ToTs: FormatTimeRFC3339(to), BarsJson: bars,
		Provider: "test", FetchedAt: time.Now().UTC().Format(time.RFC3339), ExpiresAt: sql.NullString{},
	}))
}
