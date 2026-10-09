package importer_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/importer"
	"github.com/tradermemos/api/internal/store"
)

func TestCommitOccurrences(t *testing.T) {
	for _, kind := range []string{"generic", "option", "journal", "legacy", "stable"} {
		t.Run(kind, func(t *testing.T) {
			conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
			require.NoError(t, err)
			t.Cleanup(func() { conn.Close() })
			require.NoError(t, db.Migrate(conn))
			q, ctx := store.New(conn), context.Background()
			u, err := q.CreateUser(ctx, store.CreateUserParams{ID: uuid.New().String(), Email: "occ@x.com", PasswordHash: "x"})
			require.NoError(t, err)
			acc, err := q.CreateAccount(ctx, store.CreateAccountParams{ID: uuid.New().String(), UserID: u.ID, Name: "test", BaseCurrency: "USD"})
			require.NoError(t, err)
			row := map[string]string{"symbol": "AAPL", "side": "buy", "quantity": "100", "price": "10", "executed_at": "2026-01-01T10:00:00Z"}
			if kind == "option" {
				row["symbol"] = "AAPL260821C00120000"
			}
			mapping := map[string]string{"symbol": "symbol", "side": "side", "quantity": "quantity", "price": "price", "executed_at": "executed_at"}
			parsed := importer.NewGeneric(mapping).ParseRows([]map[string]string{row, row})
			if kind == "journal" {
				row = map[string]string{"Symbol": "AAPL", "Side": "LONG", "Qty": "100", "Entry": "10", "Exit": "11", "Open Date": "2026-01-01T10:00:00Z", "Date": "2026-01-01T11:00:00Z", "Market": "STOCK", "Status": "WIN", "Notes": "same trade"}
				parsed = importer.NewJournal().ParseRows([]map[string]string{row, row})
			}
			if kind == "stable" {
				parsed.Executions[0].DedupKey = "sbi|source-row|0|0"
				parsed.Executions[1].DedupKey = "sbi|source-row|1|0"
			}
			require.Empty(t, parsed.Errors)
			require.NotEmpty(t, parsed.Executions)
			if kind == "legacy" {
				pe := parsed.Executions[0]
				base := importer.DedupHash(pe.Symbol, pe.Side, pe.Quantity, pe.Price, pe.ExecutedAt)
				require.Equal(t, "e26cea7b4f5866f4ddee0adce4725409", base)
				require.Equal(t, base, importer.DedupHashOccurrence(pe.Symbol, pe.Side, pe.Quantity, pe.Price, pe.ExecutedAt, 0))
				_, err = q.InsertExecution(ctx, store.InsertExecutionParams{ID: uuid.New().String(), UserID: u.ID, AccountID: acc.ID, Symbol: pe.Symbol, InstrumentType: pe.InstrumentType, Side: pe.Side, Quantity: pe.Quantity, Price: pe.Price, ExecutedAt: pe.ExecutedAt, Multiplier: 1, DedupHash: base})
				require.NoError(t, err)
				single := parsed
				single.Executions = single.Executions[:1]
				res, err := importer.Commit(ctx, q, u.ID, acc.ID, sql.NullString{}, single)
				require.NoError(t, err)
				require.Zero(t, res.Inserted)
				require.Empty(t, res.Errors)
			}
			res, err := importer.Commit(ctx, q, u.ID, acc.ID, sql.NullString{}, parsed)
			require.NoError(t, err)
			if kind == "legacy" {
				require.Zero(t, res.Inserted)
				require.Equal(t, 2, res.Skipped)
				require.Len(t, res.Errors, 1)
				require.Contains(t, res.Errors[0].Message, "legacy import")
			} else {
				require.Equal(t, len(parsed.Executions), res.Inserted)
				require.Empty(t, res.Errors)
			}
			fills, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: acc.ID})
			require.NoError(t, err)
			if kind == "legacy" {
				require.Len(t, fills, 1)
			} else {
				require.Len(t, fills, len(parsed.Executions))
			}
			if kind == "stable" {
				expected := []string{}
				actual := []string{}
				for _, pe := range parsed.Executions {
					sum := sha256.Sum256([]byte(pe.DedupKey))
					expected = append(expected, fmt.Sprintf("%x", sum[:16]))
				}
				for _, fill := range fills {
					actual = append(actual, fill.DedupHash)
					var details map[string]any
					if fill.Details.Valid {
						require.NoError(t, json.Unmarshal([]byte(fill.Details.String), &details))
					}
					require.NotContains(t, details, "occ")
				}
				require.ElementsMatch(t, expected, actual)
			}
			if kind == "option" {
				require.Equal(t, "option", parsed.Executions[0].InstrumentType)
				require.NoError(t, importer.NormalizeOptionExecutions(ctx, q, slog.New(slog.NewTextHandler(io.Discard, nil))))
				after, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: u.ID, AccountID: acc.ID})
				require.NoError(t, err)
				require.Equal(t, fills, after)
			}
			res, err = importer.Commit(ctx, q, u.ID, acc.ID, sql.NullString{}, parsed)
			require.NoError(t, err)
			require.Zero(t, res.Inserted)
			require.Equal(t, len(parsed.Executions), res.Skipped)
		})
	}
}

func TestDedupOccurrenceInvalidDetails(t *testing.T) {
	for _, raw := range []string{"", "oops", "-1", "999999999999999999999999999999999999"} {
		require.Zero(t, importer.DedupOccurrence(map[string]string{"occ": raw}))
	}
	require.Equal(t, 2, importer.DedupOccurrence(map[string]string{"occ": "2"}))
}
