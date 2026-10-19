package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/importer"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

// CSV rows are the lossless parser input after decoding, not a copy of every upload.
// shortcut: statements, JSON, cash and enrichment need their own replay contexts before opting in.
type csvSourceContext struct {
	Version   int      `json:"version"`
	Headers   []string `json:"headers"`
	Format    string   `json:"format"`
	Mapping   string   `json:"mapping"`
	SourceTZ  string   `json:"source_tz"`
	DateOrder string   `json:"date_order"`
}
type csvSourceRow struct {
	Values      map[string]string `json:"values"`
	OptionRight string            `json:"option_right,omitempty"`
}

func sourceHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(h, "%d:%s", len(part), part)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func executionSnapshot(e store.Execution) string {
	// Compare editable/parsed fields; creation metadata and regroup-derived basis are not edits.
	details := map[string]json.RawMessage{}
	if e.Details.Valid {
		if json.Unmarshal([]byte(e.Details.String), &details) != nil {
			return ""
		}
	}
	for _, key := range []string{"transferred_cost_basis", "transferred_unit_cost", "cash_qty_disposed", "cash_cost_basis_used", "margin_short_qty_closed", "disposal_price", "applicable_costs", "realized_pnl"} {
		delete(details, key)
	}
	d, _ := json.Marshal(details)
	e.Details = sql.NullString{String: string(d), Valid: true}
	e.CreatedAt = time.Time{}
	data, _ := json.Marshal(e)
	return string(data)
}

func (s *Server) retainCSVSource(ctx context.Context, q store.Querier, c *echo.Context, batch store.ImportBatch, loaded loadedImport, parsed importer.ParseResult, hashes []string) error {
	if loaded.Source != "csv" || (loaded.Format != "executions" && loaded.Format != "journal_trades") {
		return nil
	}
	cfg := csvSourceContext{Version: 1, Headers: loaded.Headers, Format: loaded.Format, Mapping: c.FormValue("column_mapping"), SourceTZ: strings.TrimSpace(c.FormValue("source_tz")), DateOrder: c.FormValue("date_order")}
	// Canonical JSON avoids storing the same mapping twice for whitespace/key-order differences.
	mapping := map[string]string{}
	if cfg.Mapping != "" {
		if err := json.Unmarshal([]byte(cfg.Mapping), &mapping); err != nil {
			return err
		}
	}
	data, _ := json.Marshal(mapping)
	cfg.Mapping = string(data)
	if cfg.SourceTZ == "" {
		if _, _, tz, ok := importer.MatchBroker(loaded.Headers); ok {
			cfg.SourceTZ = tz
		}
	}
	if cfg.SourceTZ == "" {
		cfg.SourceTZ = "UTC"
	}
	if cfg.DateOrder == "" {
		stamps := importer.NewGeneric(mapping).DateStamps(loaded.Rows)
		if loaded.Format == "journal_trades" {
			stamps = importer.JournalDateStamps(loaded.Rows)
		}
		cfg.DateOrder = importer.DateOrderMonthFirst
		if importer.DetectDayFirst(stamps) {
			cfg.DateOrder = importer.DateOrderDayFirst
		}
	}
	contextData, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	contextID := sourceHash(batch.UserID, batch.AccountID, string(contextData))
	all, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: batch.UserID, AccountID: batch.AccountID})
	if err != nil {
		return err
	}
	byHash := map[string]store.Execution{}
	for _, e := range all {
		byHash[e.DedupHash] = e
	}
	byRow := map[int][]store.Execution{}
	invalid := map[int]bool{}
	for i, pe := range parsed.Executions {
		if pe.SourceRow < 1 || pe.SourceRow > len(loaded.Rows) || i >= len(hashes) {
			continue
		}
		e, ok := byHash[hashes[i]]
		if !ok || !e.ImportBatchID.Valid {
			invalid[pe.SourceRow] = true
			continue
		}
		byRow[pe.SourceRow] = append(byRow[pe.SourceRow], e)
	}
	overrides := journalOptionOverrides(c)
	sources := []store.RetainedImportSource{}
	for rowNumber, executions := range byRow {
		if invalid[rowNumber] {
			continue
		}
		row := csvSourceRow{Values: loaded.Rows[rowNumber-1]}
		if overrides != nil {
			row.OptionRight = overrides.OptionRightByRow[rowNumber]
		}
		rowData, err := json.Marshal(row)
		if err != nil {
			return err
		}
		ids := []string{}
		for _, e := range executions {
			ids = append(ids, e.ID)
		}
		sourceID := sourceHash(contextID, string(rowData), strings.Join(ids, ","))
		retained := store.RetainedImportSource{Row: store.PutImportSourceRowParams{ID: sourceID, UserID: batch.UserID, AccountID: batch.AccountID, BatchID: batch.ID, ContextID: contextID, RowNumber: int64(rowNumber), Payload: string(rowData)}}
		for leg, e := range executions {
			retained.Fills = append(retained.Fills, store.PutImportSourceFillParams{SourceID: sourceID, ExecutionID: e.ID, Leg: int64(leg), Snapshot: executionSnapshot(e)})
		}
		sources = append(sources, retained)
	}
	return store.BulkRetainImportSources(ctx, q, store.PutImportSourceContextParams{ID: contextID, UserID: batch.UserID, AccountID: batch.AccountID, Payload: string(contextData)}, sources)
}

type reprocessResult struct {
	RetainedRows  int    `json:"retained_rows"`
	Executions    int    `json:"executions"`
	Changed       int    `json:"changed"`
	Trades        int    `json:"trades"`
	TradesChanged int    `json:"trades_changed"`
	Fingerprint   string `json:"fingerprint"`
	Applied       bool   `json:"applied"`
}

var errReprocessPreview = errors.New("read-only reprocess preview")

type reprocessConflict string

func (e reprocessConflict) Error() string { return string(e) }
func replayConflict(format string, args ...any) error {
	return reprocessConflict(fmt.Sprintf(format, args...))
}

func (s *Server) handleReprocessImport(c *echo.Context) error {
	uid := auth.UserID(c)
	ctx := c.Request().Context()
	batch, err := s.deps.Store.GetImportBatch(ctx, store.GetImportBatchParams{ID: c.Param("id"), UserID: uid})
	if err != nil {
		return Fail(http.StatusNotFound, "not_found", "import batch not found", nil)
	}
	if batch.Status != "committed" {
		return Fail(http.StatusConflict, "conflict", "only committed imports can be reprocessed", nil)
	}
	var in struct {
		Apply       bool   `json:"apply"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid reprocess request", nil)
	}
	if in.Apply && in.Fingerprint == "" {
		return Fail(http.StatusBadRequest, "bad_request", "preview is required before reprocessing", nil)
	}
	// A preview must always roll back, including when a future caller supplies a partial store.
	runner, ok := s.deps.Store.(store.SerializableTxRunner)
	if !ok {
		return Fail(http.StatusServiceUnavailable, "unavailable", "transactional store required", nil)
	}
	var result reprocessResult
	err = runner.InSerializableTx(ctx, func(q store.Querier) error {
		current, err := q.GetImportBatch(ctx, store.GetImportBatchParams{ID: batch.ID, UserID: uid})
		if err != nil {
			return err
		}
		if current.Status != "committed" {
			return replayConflict("import is no longer committed")
		}
		result, err = s.reprocessCSV(ctx, q, current)
		if err != nil {
			return err
		}
		if !in.Apply {
			return errReprocessPreview
		}
		if in.Fingerprint != result.Fingerprint {
			return replayConflict("data or processing rules changed since preview; preview again")
		}
		result.Applied = true
		return nil
	})
	if errors.Is(err, errReprocessPreview) {
		err = nil
	}
	if err != nil {
		var conflict reprocessConflict
		if errors.As(err, &conflict) {
			return Fail(http.StatusConflict, "reprocess_conflict", conflict.Error(), nil)
		}
		return Fail(http.StatusInternalServerError, "internal", "could not reprocess import; existing data was kept", nil)
	}
	return c.JSON(http.StatusOK, result)
}

// shortcut: replay is synchronous; batch updates or a job are needed if hosted batches exceed request deadlines.
func (s *Server) reprocessCSV(ctx context.Context, q store.Querier, batch store.ImportBatch) (reprocessResult, error) {
	result := reprocessResult{}
	rows, err := q.ListImportSourceRows(ctx, store.ListImportSourceRowsParams{BatchID: batch.ID, UserID: batch.UserID})
	if err != nil {
		return result, err
	}
	if len(rows) == 0 {
		return result, replayConflict("no retained CSV source rows; legacy or unsupported imports need the original file")
	}
	activeIDs := map[string]bool{}
	activeRows := rows[:0]
	for _, row := range rows {
		if row.Active != 0 {
			activeRows = append(activeRows, row)
			activeIDs[row.ID] = true
		}
	}
	rows = activeRows
	if len(rows) == 0 {
		return result, replayConflict("newer source data exists; use its import batch")
	}
	fills, err := q.ListImportSourceFills(ctx, store.ListImportSourceFillsParams{BatchID: batch.ID, UserID: batch.UserID})
	if err != nil {
		return result, err
	}
	activeFills := fills[:0]
	for _, fill := range fills {
		if activeIDs[fill.SourceID] {
			activeFills = append(activeFills, fill)
		}
	}
	fills = activeFills
	bySource := map[string][]store.ImportSourceFill{}
	for _, fill := range fills {
		bySource[fill.SourceID] = append(bySource[fill.SourceID], fill)
	}
	before, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: batch.UserID, AccountID: batch.AccountID})
	if err != nil {
		return result, err
	}
	original := map[string]store.Execution{}
	for _, e := range before {
		original[e.ID] = e
	}
	beforeTrades, err := tradeSnapshots(ctx, q, batch.UserID, batch.AccountID)
	if err != nil {
		return result, err
	}
	memberships, err := tradeMemberships(ctx, q, batch.UserID, batch.AccountID)
	if err != nil {
		return result, err
	}
	input, _ := json.Marshal(struct {
		Rows        any
		Fills       any
		Executions  any
		Memberships any
		Trades      any
	}{rows, fills, before, memberships, beforeTrades})
	seen := map[string]bool{}
	updates := []store.ReplaceImportedExecutionParams{}
	for _, row := range rows {
		var cfg csvSourceContext
		var source csvSourceRow
		if err := json.Unmarshal([]byte(row.ContextPayload), &cfg); err != nil {
			return result, err
		}
		if cfg.Version != 1 {
			return result, replayConflict("unsupported retained source version")
		}
		if err := json.Unmarshal([]byte(row.Payload), &source); err != nil {
			return result, err
		}
		// Reuse the same parser entry point and effective choices, with one complete source row/leg group.
		values := url.Values{"column_mapping": {cfg.Mapping}, "source_tz": {cfg.SourceTZ}, "date_order": {cfg.DateOrder}}
		if source.OptionRight != "" {
			values.Set("journal_option_overrides", `{"1":`+strconv.Quote(source.OptionRight)+`}`)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		parserContext := s.Echo.NewContext(req, nil)
		parsed, _, err := parseImport(parserContext, loadedImport{Source: "csv", Format: cfg.Format, Headers: cfg.Headers, Rows: []map[string]string{source.Values}}, nil, true)
		if err != nil {
			return result, replayConflict("row %d cannot be parsed with current rules; no data changed", row.RowNumber)
		}
		links := bySource[row.ID]
		if len(parsed.Errors) > 0 || len(parsed.Executions) != len(links) || len(links) == 0 {
			return result, replayConflict("row %d no longer maps to the same executions; no data changed", row.RowNumber)
		}
		for leg, pe := range parsed.Executions {
			link := links[leg]
			old, ok := original[link.ExecutionID]
			if !ok || old.UserID != batch.UserID || old.AccountID != batch.AccountID || !old.ImportBatchID.Valid {
				return result, replayConflict("row %d has deleted or unavailable executions", row.RowNumber)
			}
			if seen[old.ID] {
				return result, replayConflict("source rows overlap; review required")
			}
			seen[old.ID] = true
			if executionSnapshot(old) != link.Snapshot {
				return result, replayConflict("row %d was edited or enriched since import; existing edits were kept", row.RowNumber)
			}
			if importer.IsSBI(cfg.Headers) {
				pe.ExecutedAt = pe.ExecutedAt.Add(time.Duration((row.RowNumber-1)*2) * time.Microsecond)
			}
			update, err := importer.ReprocessedExecution(ctx, q, old, pe)
			if err != nil {
				return result, err
			}
			updates = append(updates, update)
			n, err := q.ReplaceImportedExecution(ctx, update)
			if err != nil {
				return result, replayConflict("row %d conflicts with existing data; no data changed", row.RowNumber)
			}
			if n != 1 {
				return result, replayConflict("execution changed during processing")
			}
			after, err := q.GetExecution(ctx, store.GetExecutionParams{ID: old.ID, UserID: batch.UserID})
			if err != nil {
				return result, err
			}
			if executionSnapshot(after) != executionSnapshot(old) {
				result.Changed++
			}
		}
	}
	getBars := s.deps.Trades.GetBars
	if getBars != nil && s.deps.Market != nil {
		getBars = marketdata.NewService(q, s.deps.Market.Provider).GetBars
	}
	if err := trades.NewService(q, getBars).Regroup(ctx, batch.UserID, batch.AccountID); err != nil {
		return result, err
	}
	afterMemberships, err := tradeMemberships(ctx, q, batch.UserID, batch.AccountID)
	if err != nil {
		return result, err
	}
	// Even an unannotated trade may have a review link added concurrently. Keep every existing identity/membership.
	if !reflect.DeepEqual(memberships, afterMemberships) {
		return result, replayConflict("new grouping would change existing trade associations; no data changed")
	}
	for _, fill := range fills {
		after, err := q.GetExecution(ctx, store.GetExecutionParams{ID: fill.ExecutionID, UserID: batch.UserID})
		if err != nil {
			return result, err
		}
		if err := q.UpdateImportSourceSnapshot(ctx, store.UpdateImportSourceSnapshotParams{SourceID: fill.SourceID, ExecutionID: fill.ExecutionID, Snapshot: executionSnapshot(after)}); err != nil {
			return result, err
		}
	}
	afterTrades, err := tradeSnapshots(ctx, q, batch.UserID, batch.AccountID)
	if err != nil {
		return result, err
	}
	for id, value := range afterTrades {
		if beforeTrades[id] != value {
			result.TradesChanged++
		}
	}
	output, _ := json.Marshal(struct {
		Executions any
		Trades     any
	}{updates, afterTrades})
	result.RetainedRows = len(rows)
	result.Executions = len(fills)
	result.Trades = len(afterMemberships)
	result.Fingerprint = sourceHash(string(input), string(output))
	return result, nil
}

func tradeMemberships(ctx context.Context, q store.Querier, uid, accountID string) (map[string][]string, error) {
	list, err := q.ListTrades(ctx, store.ListTradesParams{UserID: uid, AccountID: accountID})
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, t := range list {
		fills, err := q.ListExecutionsForTrade(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		ids := []string{}
		for _, e := range fills {
			ids = append(ids, e.ID)
		}
		sort.Strings(ids)
		out[t.ID] = ids
	}
	return out, nil
}

func tradeSnapshots(ctx context.Context, q store.Querier, uid, accountID string) (map[string]string, error) {
	list, err := q.ListTrades(ctx, store.ListTradesParams{UserID: uid, AccountID: accountID})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, t := range list {
		t.UpdatedAt = time.Time{}
		data, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		out[t.ID] = string(data)
	}
	return out, nil
}
