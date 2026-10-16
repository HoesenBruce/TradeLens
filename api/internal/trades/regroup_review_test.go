package trades_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestReviewedRegroupPreviewApplyRollbackAndAssociations(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "synthetic.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.NewForDriver(conn, "sqlite")
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "synthetic@qa.test", PasswordHash: "x"})
	require.NoError(t, err)
	_, err = q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: "u", Name: "Synthetic", BaseCurrency: "JPY"})
	require.NoError(t, err)
	for i, side := range []string{"buy", "sell", "sell"} {
		qty, price, details := 1.0, 110.0, `{"lot":"sbi:cash","position_effect":"reduce","broker_reported_realized_pnl":120}`
		if i == 0 {
			qty, price, details = 2, 100, `{"lot":"sbi:cash","position_effect":"increase"}`
		}
		if i == 2 {
			details = `{"lot":"sbi:cash","position_effect":"reduce","broker_reported_realized_pnl":-30}`
		}
		_, err = q.InsertExecution(ctx, store.InsertExecutionParams{ID: fmt.Sprint(i), UserID: "u", AccountID: "a", Symbol: "1515", InstrumentType: "stock", Side: side, Quantity: qty, Price: price, ExecutedAt: time.Date(2026, 9, 1+i, 1, 0, 0, 0, time.UTC), Multiplier: 1, DedupHash: fmt.Sprint(i), Details: sql.NullString{String: details, Valid: true}})
		require.NoError(t, err)
	}
	svc := trades.NewService(q)
	require.NoError(t, svc.Regroup(ctx, "u", "a"))
	err = q.UpdateTradeNotes(ctx, store.UpdateTradeNotesParams{ID: "0", UserID: "u", Notes: "synthetic review"})
	require.NoError(t, err)
	_, err = conn.Exec("UPDATE trades SET net_pnl=777 WHERE id='0'")
	require.NoError(t, err)
	before, err := q.GetTrade(ctx, store.GetTradeParams{ID: "0", UserID: "u"})
	require.NoError(t, err)
	require.NoError(t, q.UpsertTradeJournal(ctx, store.UpsertTradeJournalParams{TradeID: "0", UserID: "u", Notes: "synthetic journal"}))
	fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
	require.NoError(t, err)
	preview, err := svc.ReviewedRegroup(ctx, "u", "a", true, "", nil)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 1)
	require.Equal(t, 777.0, *preview.Changes[0].Before.NetPnL)
	require.Equal(t, 90.0, *preview.Changes[0].After.NetPnL)
	require.Empty(t, preview.RemovedIDs)
	require.Equal(t, []string{"0"}, preview.PreservedIDs)
	after, err := q.GetTrade(ctx, store.GetTradeParams{ID: "0", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, before, after)
	// The market loader must use the transaction store rather than the root connection.
	bound := false
	svc.GetBars = func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{}, fmt.Errorf("root loader must not run")
	}
	_, err = svc.ReviewedRegroup(ctx, "u", "a", true, "", func(tx store.Querier) trades.BarsGetter {
		require.NotSame(t, q, tx)
		bound = true
		return func(context.Context, marketdata.Request) (marketdata.Response, error) {
			return marketdata.Response{Timezone: "Asia/Tokyo"}, nil
		}
	})
	require.NoError(t, err)
	require.True(t, bound)
	svc.GetBars = nil
	_, err = svc.ReviewedRegroup(ctx, "u", "a", false, "stale", nil)
	require.ErrorIs(t, err, trades.ErrRegroupReview)
	svc.GetBars = func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{}, fmt.Errorf("synthetic failure")
	}
	_, err = svc.ReviewedRegroup(ctx, "u", "a", false, preview.Token, nil)
	require.Error(t, err)
	after, err = q.GetTrade(ctx, store.GetTradeParams{ID: "0", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, before, after)
	svc.GetBars = nil
	for range 2 {
		applied, err := svc.ReviewedRegroup(ctx, "u", "a", false, preview.Token, nil)
		require.NoError(t, err)
		require.True(t, applied.Applied)
	}
	after, err = q.GetTrade(ctx, store.GetTradeParams{ID: "0", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, 90.0, after.NetPnl.Float64)
	require.Equal(t, before.Notes, after.Notes)
	journal, err := q.GetTradeJournal(ctx, store.GetTradeJournalParams{TradeID: "0", UserID: "u"})
	require.NoError(t, err)
	require.Equal(t, "synthetic journal", journal.Notes)
	afterFills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"})
	require.NoError(t, err)
	require.Equal(t, fills, afterFills)
	// A changed grouping may not silently cascade-delete any old review identity.
	_, err = conn.Exec("INSERT INTO trades SELECT 'legacy',user_id,account_id,symbol,instrument_type,direction,status,opened_at,closed_at,qty_opened,avg_entry_price,avg_exit_price,gross_pnl,fees_total,net_pnl,pnl_currency,return_pct,r_multiple,time_in_trade_secs,notes,created_at,updated_at,qty_remaining,accounting_warning FROM trades WHERE id='0'")
	require.NoError(t, err)
	preview, err = svc.ReviewedRegroup(ctx, "u", "a", true, "", nil)
	require.NoError(t, err)
	require.Contains(t, preview.RemovedIDs, "legacy")
	_, err = svc.ReviewedRegroup(ctx, "u", "a", false, preview.Token, nil)
	require.ErrorIs(t, err, trades.ErrRegroupReview)
	_, err = q.GetTrade(ctx, store.GetTradeParams{ID: "legacy", UserID: "u"})
	require.NoError(t, err)
}
