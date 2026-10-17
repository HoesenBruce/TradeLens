package importer_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/importer"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestSBIReviewImportPersistenceAndDedup(t *testing.T) {
	fixture, err := os.ReadFile("testdata/sbi-review-direction.csv")
	require.NoError(t, err)
	_, rows, ok, err := importer.ReadSBITradeCSV(fixture)
	require.NoError(t, err)
	require.True(t, ok)
	for _, reversed := range []bool{false, true} {
		t.Run(map[bool]string{false: "closing-first", true: "reversed"}[reversed], func(t *testing.T) {
			input := slices.Clone(rows)
			if reversed {
				slices.Reverse(input)
			}
			parsed := importer.ParseSBIRows(input, nil, "")
			require.Empty(t, parsed.Errors)
			conn, err := db.Open(filepath.Join(t.TempDir(), "qa.db"))
			require.NoError(t, err)
			defer conn.Close()
			require.NoError(t, db.Migrate(conn))
			q, ctx := store.New(conn), context.Background()
			u, err := q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "review@synthetic.invalid", PasswordHash: "x"})
			require.NoError(t, err)
			acc, err := q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: u.ID, Name: "SBI", Broker: "sbi", AccountType: "margin", BaseCurrency: "JPY"})
			require.NoError(t, err)
			result, err := importer.Commit(ctx, q, u.ID, acc.ID, sql.NullString{}, parsed)
			require.NoError(t, err)
			require.Equal(t, 9, result.Inserted)
			saved, err := q.ListTrades(ctx, store.ListTradesParams{UserID: u.ID})
			require.NoError(t, err)
			require.Len(t, saved, 4)
			fees := 0.0
			for _, tr := range saved {
				fees += tr.FeesTotal
				require.False(t, tr.TimeInTradeSecs.Valid)
				switch tr.QtyOpened {
				case 20:
					require.Equal(t, "long", tr.Direction)
					require.Equal(t, 101.0, tr.AvgEntryPrice)
					require.Equal(t, 111.0, tr.AvgExitPrice.Float64)
					require.Zero(t, tr.QtyRemaining)
					require.Equal(t, 200.0, tr.NetPnl.Float64)
				case 10:
					require.Equal(t, "long", tr.Direction)
					require.Equal(t, 100.0, tr.AvgEntryPrice)
					require.Equal(t, 110.0, tr.AvgExitPrice.Float64)
					require.Zero(t, tr.QtyRemaining)
				case 70:
					require.Equal(t, "short", tr.Direction)
					require.Equal(t, 10.0, tr.QtyRemaining)
					require.False(t, tr.NetPnl.Valid)
				case 0:
					require.Equal(t, "long", tr.Direction)
					require.False(t, tr.NetPnl.Valid)
					require.Contains(t, tr.AccountingWarning, "Incomplete SBI opening evidence")
				default:
					t.Fatalf("unexpected trade: %+v", tr)
				}
				require.NoError(t, q.UpdateTradeNotes(ctx, store.UpdateTradeNotesParams{ID: tr.ID, UserID: u.ID, Notes: "retained review"}))
			}
			require.Equal(t, 9.0, fees)
			before, err := q.ListTrades(ctx, store.ListTradesParams{UserID: u.ID})
			require.NoError(t, err)
			slices.Reverse(input)
			repeated := importer.ParseSBIRows(input, nil, "")
			result, err = importer.Commit(ctx, q, u.ID, acc.ID, sql.NullString{}, repeated)
			require.NoError(t, err)
			require.Zero(t, result.Inserted)
			require.Equal(t, 9, result.Skipped)
			require.NoError(t, trades.NewService(q).Regroup(ctx, u.ID, acc.ID))
			after, err := q.ListTrades(ctx, store.ListTradesParams{UserID: u.ID})
			require.NoError(t, err)
			// Regroup may advance its audit timestamp; all accounting and
			// execution-derived fields, including real trade times, must agree.
			require.Len(t, after, len(before))
			for i := range before {
				require.False(t, after[i].UpdatedAt.Before(before[i].UpdatedAt))
				before[i].UpdatedAt = after[i].UpdatedAt
			}
			require.Equal(t, before, after)
		})
	}
}
