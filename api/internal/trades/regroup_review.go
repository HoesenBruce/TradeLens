package trades

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/tradermemos/api/internal/store"
)

var ErrRegroupReview = errors.New("SBI regroup requires a current preview token and cannot remove existing review identities")
var previewRollback = errors.New("regroup preview rollback")

type RegroupTrade struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	EntryBasis   float64  `json:"entry_basis"`
	Quantity     float64  `json:"quantity"`
	Remaining    float64  `json:"remaining"`
	NetPnL       *float64 `json:"net_pnl"`
	Warning      string   `json:"accounting_warning"`
	ExecutionIDs []string `json:"execution_ids"`
}

type RegroupChange struct {
	Before *RegroupTrade `json:"before"`
	After  *RegroupTrade `json:"after"`
}

type RegroupReview struct {
	Token        string          `json:"preview_token"`
	Applied      bool            `json:"applied"`
	Changes      []RegroupChange `json:"changed_groups"`
	RemovedIDs   []string        `json:"removed_trade_ids"`
	PreservedIDs []string        `json:"preserved_trade_ids"`
}

func regroupState(ctx context.Context, q store.Querier, userID, accountID string) ([]RegroupTrade, error) {
	rows, err := q.ListTrades(ctx, store.ListTradesParams{UserID: userID, AccountID: sql.NullString{String: accountID, Valid: true}})
	if err != nil {
		return nil, err
	}
	out := make([]RegroupTrade, 0, len(rows))
	for _, row := range rows {
		fills, err := q.ListExecutionsForTrade(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(fills))
		for _, f := range fills {
			ids = append(ids, f.ID)
		}
		sort.Strings(ids)
		var net *float64
		if row.NetPnl.Valid {
			v := row.NetPnl.Float64
			net = &v
		}
		out = append(out, RegroupTrade{ID: row.ID, Status: row.Status, EntryBasis: row.AvgEntryPrice, Quantity: row.QtyOpened, Remaining: row.QtyRemaining, NetPnL: net, Warning: row.AccountingWarning, ExecutionIDs: ids})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ReviewedRegroup stages the actual regroup in a transaction. Preview always
// rolls back; apply verifies the resulting state before committing any writes.
func (s *Service) ReviewedRegroup(ctx context.Context, userID, accountID string, preview bool, token string, bindBars func(store.Querier) BarsGetter) (RegroupReview, error) {
	out := RegroupReview{Changes: []RegroupChange{}, RemovedIDs: []string{}, PreservedIDs: []string{}}
	runner, ok := s.q.(store.TxRunner)
	if !ok {
		return out, fmt.Errorf("reviewed regroup requires transactions")
	}
	err := runner.InTx(ctx, func(q store.Querier) error {
		before, err := regroupState(ctx, q, userID, accountID)
		if err != nil {
			return err
		}
		getBars := s.GetBars
		if bindBars != nil {
			getBars = bindBars(q)
		}
		if err := NewService(q, getBars).Regroup(ctx, userID, accountID); err != nil {
			return err
		}
		after, err := regroupState(ctx, q, userID, accountID)
		if err != nil {
			return err
		}
		executions, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: userID, AccountID: accountID})
		if err != nil {
			return err
		}
		data, err := json.Marshal(struct {
			Executions []store.Execution
			Trades     []RegroupTrade
		}{executions, after})
		if err != nil {
			return err
		}
		out.Token = fmt.Sprintf("%x", sha256.Sum256(data))
		old := map[string]RegroupTrade{}
		for _, v := range before {
			old[v.ID] = v
		}
		for _, v := range after {
			if b, exists := old[v.ID]; exists {
				out.PreservedIDs = append(out.PreservedIDs, v.ID)
				if !reflect.DeepEqual(b, v) {
					a := v
					out.Changes = append(out.Changes, RegroupChange{Before: &b, After: &a})
				}
				delete(old, v.ID)
			} else {
				a := v
				out.Changes = append(out.Changes, RegroupChange{After: &a})
			}
		}
		for _, b := range before {
			if _, exists := old[b.ID]; exists {
				b := b
				out.RemovedIDs = append(out.RemovedIDs, b.ID)
				out.Changes = append(out.Changes, RegroupChange{Before: &b})
			}
		}
		if preview {
			return previewRollback
		}
		if token == "" || token != out.Token || len(out.RemovedIDs) > 0 {
			return ErrRegroupReview
		}
		out.Applied = true
		return nil
	})
	if errors.Is(err, previewRollback) {
		err = nil
	}
	if err == nil && out.Applied && s.AfterRegroup != nil {
		s.AfterRegroup(userID, accountID)
	}
	return out, err
}
