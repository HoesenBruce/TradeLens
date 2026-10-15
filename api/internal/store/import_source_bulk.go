package store

import (
	"context"
	"database/sql"
	"errors"
)

// RetainedImportSource keeps a complete CSV row/leg group together during deduplication.
type RetainedImportSource struct {
	Row   PutImportSourceRowParams
	Fills []PutImportSourceFillParams
}

func BulkRetainImportSources(ctx context.Context, q Querier, cfg PutImportSourceContextParams, sources []RetainedImportSource) error {
	if b, ok := q.(interface {
		RetainImportSourcesBulk(context.Context, PutImportSourceContextParams, []RetainedImportSource) error
	}); ok {
		return b.RetainImportSourcesBulk(ctx, cfg, sources)
	}
	for _, source := range sources {
		if _, err := q.GetImportSourceRow(ctx, GetImportSourceRowParams{ID: source.Row.ID, UserID: cfg.UserID}); err == nil {
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := q.PutImportSourceContext(ctx, cfg); err != nil {
			return err
		}
		for _, fill := range source.Fills {
			if err := q.SupersedeImportSources(ctx, fill.ExecutionID); err != nil {
				return err
			}
		}
		if err := q.PutImportSourceRow(ctx, source.Row); err != nil {
			return err
		}
		for _, fill := range source.Fills {
			if err := q.PutImportSourceFill(ctx, fill); err != nil {
				return err
			}
		}
	}
	return nil
}

func retainImportSources(ctx context.Context, db DBTX, d bulkDialect, cfg PutImportSourceContextParams, sources []RetainedImportSource) error {
	ids := make([]string, 0, len(sources))
	for _, s := range sources {
		ids = append(ids, s.Row.ID)
	}
	existing := map[string]bool{}
	chunk := max(d.maxParams-1, 1)
	for start := 0; start < len(ids); start += chunk {
		stmt, args := inClause(d, "SELECT id FROM import_source_rows WHERE user_id = "+d.placeholder(1)+" AND id IN", []any{cfg.UserID}, ids[start:min(start+chunk, len(ids))])
		rows, err := db.QueryContext(ctx, stmt, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			existing[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	var sourceRows, fillRows [][]any
	var executions []string
	for _, s := range sources {
		if existing[s.Row.ID] {
			continue
		}
		existing[s.Row.ID] = true
		r := s.Row
		sourceRows = append(sourceRows, []any{r.ID, r.UserID, r.AccountID, r.BatchID, r.ContextID, r.RowNumber, r.Payload})
		for _, f := range s.Fills {
			executions = append(executions, f.ExecutionID)
			fillRows = append(fillRows, []any{f.SourceID, f.ExecutionID, f.Leg, f.Snapshot})
		}
	}
	if len(sourceRows) == 0 {
		return nil
	}
	if err := execValues(ctx, db, d, "INSERT INTO import_source_contexts (id,user_id,account_id,payload)", "", " ON CONFLICT(id) DO NOTHING", [][]any{{cfg.ID, cfg.UserID, cfg.AccountID, cfg.Payload}}); err != nil {
		return err
	}
	for start := 0; start < len(executions); start += d.maxParams {
		stmt, args := inClause(d, "UPDATE import_source_rows SET active=0 WHERE id IN (SELECT source_id FROM import_source_fills WHERE execution_id IN", nil, executions[start:min(start+d.maxParams, len(executions))])
		if _, err := db.ExecContext(ctx, stmt+")", args...); err != nil {
			return err
		}
	}
	if err := execValues(ctx, db, d, "INSERT INTO import_source_rows (id,user_id,account_id,batch_id,context_id,row_number,payload)", "", "", sourceRows); err != nil {
		return err
	}
	return execValues(ctx, db, d, "INSERT INTO import_source_fills (source_id,execution_id,leg,snapshot)", "", "", fillRows)
}

func (q *Queries) RetainImportSourcesBulk(ctx context.Context, cfg PutImportSourceContextParams, sources []RetainedImportSource) error {
	return retainImportSources(ctx, q.db, sqliteBulk, cfg, sources)
}
func (p *PG) RetainImportSourcesBulk(ctx context.Context, cfg PutImportSourceContextParams, sources []RetainedImportSource) error {
	return retainImportSources(ctx, p.db, postgresBulk, cfg, sources)
}
