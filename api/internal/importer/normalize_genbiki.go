package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

// NormalizeGenbiki upgrades only the exact synthetic-leg signature emitted by
// the historical SBI parser. Ordinary rows have even microsecond offsets;
// only the second genbiki leg has an odd offset. IDs and dedup hashes survive.
// Each account's annotation and regroup commit together, including retries.
func NormalizeGenbiki(ctx context.Context, q store.Querier) error {
	users, err := q.ListUsers(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, user := range users {
		accounts, err := q.ListAccounts(ctx, user.ID)
		if err != nil {
			return err
		}
		for _, account := range accounts {
			err := store.InTx(ctx, q, func(tx store.Querier) error {
				rows, err := tx.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: user.ID, AccountID: account.ID})
				if err != nil {
					return err
				}
				affected := false
				for i := range rows {
					var d map[string]any
					if json.Unmarshal([]byte(rows[i].Details.String), &d) == nil && d["conversion_type"] == "genbiki" && d["lot"] == "sbi:cash" && d["transferred_cost_basis"] == nil {
						affected = true
					}
					if i == 0 || !legacyGenbikiPair(rows[i-1], rows[i]) {
						continue
					}
					id := "sbi|legacy-genbiki|" + rows[i-1].ID
					for _, index := range []int{i - 1, i} {
						row := &rows[index]
						details := map[string]any{}
						if err := json.Unmarshal([]byte(row.Details.String), &details); err != nil {
							return err
						}
						details["event_type"], details["conversion_type"], details["conversion_id"] = "position_conversion", "genbiki", id
						encoded, err := json.Marshal(details)
						if err != nil {
							return err
						}
						row.Details = sql.NullString{String: string(encoded), Valid: true}
						if err := tx.UpdateExecutionContract(ctx, store.UpdateExecutionContractParams{ID: row.ID, UserID: row.UserID, Symbol: row.Symbol, DedupHash: row.DedupHash, Details: row.Details}); err != nil {
							return err
						}
					}
					affected = true
				}
				// Existing #130 conversions also need their cash trade basis rebuilt.
				if affected {
					return trades.NewService(tx).Regroup(ctx, user.ID, account.ID)
				}
				return nil
			})
			if err != nil {
				failures = append(failures, fmt.Errorf("account %s: %w", account.ID, err))
			}
		}
	}
	return errors.Join(failures...)
}

func legacyGenbikiPair(margin, cash store.Execution) bool {
	var m, c map[string]any
	if json.Unmarshal([]byte(margin.Details.String), &m) != nil || json.Unmarshal([]byte(cash.Details.String), &c) != nil {
		return false
	}
	local := margin.ExecutedAt.In(time.FixedZone("JST", 9*3600))
	offset := local.Sub(time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()))
	return m["lot"] == "sbi:margin-long" && c["lot"] == "sbi:cash" &&
		m["position_type"] == "margin_long" && c["position_type"] == "cash" &&
		m["position_effect"] == "reduce" && c["position_effect"] == "increase" &&
		m["event_type"] == nil && c["event_type"] == nil && m["conversion_type"] == nil && c["conversion_type"] == nil &&
		m["broker_reported_realized_pnl"] == nil && c["broker_reported_realized_pnl"] == nil &&
		margin.UserID == cash.UserID && margin.AccountID == cash.AccountID && margin.ImportBatchID == cash.ImportBatchID &&
		margin.Symbol == cash.Symbol && margin.InstrumentType == "stock" && cash.InstrumentType == "stock" &&
		margin.Side == "sell" && cash.Side == "buy" && margin.Quantity > 0 && margin.Quantity == cash.Quantity && margin.Price == cash.Price &&
		margin.Multiplier == cash.Multiplier && cash.Fees == 0 && cash.Commission == 0 &&
		offset >= 0 && offset < time.Second && offset%(2*time.Microsecond) == 0 && cash.ExecutedAt.Sub(margin.ExecutedAt) == time.Microsecond
}
