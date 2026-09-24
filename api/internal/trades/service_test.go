package trades_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestRegroupPersistsClosedTrade(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	require.NoError(t, err)
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := context.Background()

	u, _ := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.New().String(), Email: "a@b.com", PasswordHash: "x"})
	acc, _ := q.CreateAccount(ctx, store.CreateAccountParams{ID: uuid.New().String(), UserID: u.ID, Name: "M", BaseCurrency: "USD"})

	mk := func(side string, qty, price float64, ts string) {
		tt, _ := time.Parse(time.RFC3339, ts)
		_, err := q.InsertExecution(ctx, store.InsertExecutionParams{
			ID: uuid.New().String(), UserID: u.ID, AccountID: acc.ID, Symbol: "AAPL",
			InstrumentType: "stock", Side: side, Quantity: qty, Price: price,
			ExecutedAt: tt, Multiplier: 1, DedupHash: uuid.New().String(),
		})
		require.NoError(t, err)
	}
	mk("buy", 100, 10, "2026-01-01T10:00:00Z")
	mk("sell", 100, 12, "2026-01-01T11:00:00Z")

	svc := trades.NewService(q)
	require.NoError(t, svc.Regroup(ctx, u.ID, acc.ID))

	closed, err := q.ListClosedTrades(ctx, store.ListClosedTradesParams{UserID: u.ID})
	require.NoError(t, err)
	require.Len(t, closed, 1)
	require.True(t, closed[0].NetPnl.Valid)
	require.Equal(t, 200.0, closed[0].NetPnl.Float64)
}

func TestRegroupPersistsSplitSettlementAndWarning(t *testing.T) {
	for _, reported := range []bool{true, false} {
		t.Run(fmt.Sprint(reported), func(t *testing.T) {
			conn, err := db.Open(filepath.Join(t.TempDir(), "split.db"))
			require.NoError(t, err)
			defer conn.Close()
			require.NoError(t, db.Migrate(conn))
			q := store.New(conn)
			ctx := context.Background()
			u, err := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.New().String(), Email: "split@test.com", PasswordHash: "x"})
			require.NoError(t, err)
			acc, err := q.CreateAccount(ctx, store.CreateAccountParams{ID: uuid.New().String(), UserID: u.ID, Name: "SBI", BaseCurrency: "JPY"})
			require.NoError(t, err)
			for i, price := range []float64{17980, 17510, 2614.5, 2611.5} {
				day, side := 25+i, "buy"
				details := `{"lot":"sbi:margin-long"}`
				if i >= 2 {
					day = 29
					side = "sell"
					if reported {
						details = fmt.Sprintf(`{"lot":"sbi:margin-long","broker_reported_realized_pnl":%d}`, []int{11331, 11031}[i-2])
					}
				}
				_, err = q.InsertExecution(ctx, store.InsertExecutionParams{ID: fmt.Sprint(i), UserID: u.ID, AccountID: acc.ID, Symbol: "7013", InstrumentType: "stock", Side: side, Quantity: 100, Price: price, Fees: 9.5, Multiplier: 1, ExecutedAt: time.Date(2025, 9, day, i, 0, 0, 0, time.UTC), DedupHash: fmt.Sprint(i), Details: sql.NullString{Valid: true, String: details}})
				require.NoError(t, err)
			}
			before, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: acc.ID})
			require.NoError(t, err)
			svc := trades.NewService(q, func(context.Context, marketdata.Request) (marketdata.Response, error) {
				return marketdata.Response{Timezone: "Asia/Tokyo", CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2025-09-29", CandidateType: "stock_split", SuspectedRatio: 7}}}, nil
			})
			for range 2 {
				require.NoError(t, svc.Regroup(ctx, u.ID, acc.ID))
			}
			rows, err := q.ListClosedTrades(ctx, store.ListClosedTradesParams{UserID: u.ID})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, reported, rows[0].NetPnl.Valid)
			if reported {
				require.Equal(t, 22362.0, rows[0].NetPnl.Float64)
				require.Equal(t, 22400.0, rows[0].GrossPnl.Float64)
			}
			require.Contains(t, rows[0].AccountingWarning, "stock_split on 2025-09-29")
			require.False(t, rows[0].ReturnPct.Valid)
			after, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: acc.ID})
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}
