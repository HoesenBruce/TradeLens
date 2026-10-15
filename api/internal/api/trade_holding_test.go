package api

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func TestTradeHoldingPrecisionAndMarketDates(t *testing.T) {
	parse := func(s string) time.Time { at, err := time.Parse(time.RFC3339, s); require.NoError(t, err); return at }
	for _, tc := range []struct {
		name, symbol, start, end, startPrecision, endPrecision, kind string
		days                                                         int
		seconds                                                      int64
	}{
		{"precise", "AAPL", "2026-10-13T09:30:00-04:00", "2026-10-13T10:45:00-04:00", "", "", "elapsed", 0, 4500},
		{"same day", "6920", "2026-10-13T09:00:00+09:00", "2026-10-13T15:00:00+09:00", "date", "date", "calendar", 0, 0},
		{"next day", "6920", "2026-10-13T09:00:00+09:00", "2026-10-14T09:00:00+09:00", "date", "date", "calendar", 1, 0},
		{"three days", "6920", "2026-10-13T09:00:00+09:00", "2026-10-16T09:00:00+09:00", "date", "date", "calendar", 3, 0},
		{"mixed start", "6920", "2026-10-13T09:00:00+09:00", "2026-10-14T10:45:00+09:00", "date", "", "calendar", 1, 0},
		{"mixed end", "AAPL", "2026-10-13T23:30:00-04:00", "2026-10-14T00:00:00-04:00", "", "date", "calendar", 1, 0},
		{"Tokyo midnight across UTC date", "6920", "2026-10-13T14:55:00Z", "2026-10-13T15:05:00Z", "date", "", "calendar", 1, 0},
		{"New York same date across UTC midnight", "AAPL", "2026-10-13T23:55:00Z", "2026-10-14T00:05:00Z", "date", "", "calendar", 0, 0},
		{"DST spring", "AAPL", "2026-03-07T12:00:00-05:00", "2026-03-09T12:00:00-04:00", "date", "date", "calendar", 2, 0},
		{"DST fall", "AAPL", "2026-10-31T12:00:00-04:00", "2026-11-02T12:00:00-05:00", "date", "date", "calendar", 2, 0},
		{"precise DST", "AAPL", "2026-03-07T12:00:00-05:00", "2026-03-09T12:00:00-04:00", "", "", "elapsed", 0, 47 * 3600},
		{"invalid precision", "AAPL", "2026-10-13T09:30:00Z", "2026-10-13T10:45:00Z", "unknown", "", "unknown", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, end := parse(tc.start), parse(tc.end)
			tr := store.Trade{Symbol: tc.symbol, InstrumentType: "stock", Status: "closed", QtyOpened: 100, OpenedAt: start, ClosedAt: sql.NullTime{Time: end, Valid: true}}
			fills := []store.ListOptionExecutionDetailsForUserRow{
				{ExecutedAt: start, Details: sql.NullString{Valid: true, String: `{"source_time_precision":"` + tc.startPrecision + `"}`}},
				{ExecutedAt: end, Details: sql.NullString{Valid: true, String: `{"source_time_precision":"` + tc.endPrecision + `"}`}},
			}
			got := tradeHolding(tr, fills, end)
			require.Equal(t, tc.kind, got.Kind)
			if tc.kind == "calendar" {
				require.Equal(t, tc.days, *got.CalendarDays)
				require.Nil(t, got.Seconds)
			}
			if tc.kind == "elapsed" {
				require.Equal(t, tc.seconds, *got.Seconds)
				require.Nil(t, got.CalendarDays)
			}
			tr.AccountingWarning = "Incomplete SBI opening evidence"
			require.Equal(t, "unknown", tradeHolding(tr, fills, end).Kind)
			tr.AccountingWarning = ""
			require.Equal(t, "unknown", tradeHolding(tr, nil, end).Kind)
			tr.Status = "open"
			ongoing := tradeHolding(tr, fills, end)
			require.True(t, ongoing.Ongoing)
			if tc.startPrecision == "date" {
				require.Equal(t, tc.days, *ongoing.CalendarDays)
			}
		})
	}
}

func TestTradeHoldingKeepsPreciseEndpointsWithInteriorDateOnlyFill(t *testing.T) {
	start := time.Date(2026, 10, 13, 14, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	tr := store.Trade{Symbol: "AAPL", InstrumentType: "stock", Status: "closed", QtyOpened: 100, OpenedAt: start, ClosedAt: sql.NullTime{Time: end, Valid: true}}
	fills := []store.ListOptionExecutionDetailsForUserRow{{ExecutedAt: start}, {ExecutedAt: end}, {ExecutedAt: start.Add(24 * time.Hour), Details: sql.NullString{Valid: true, String: `{"source_time_precision":"date"}`}}}
	got := tradeHolding(tr, fills, end)
	require.Equal(t, "elapsed", got.Kind)
	require.Equal(t, int64(48*3600), *got.Seconds)
}
