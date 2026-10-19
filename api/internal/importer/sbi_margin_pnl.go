package importer

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

var ErrSBIEnrichmentUnresolved = errors.New("resolve invalid, ambiguous or conflicting enrichment rows before applying")

type SBIMarginPnLRow struct {
	Date            string                 `json:"date"`
	Symbol          string                 `json:"symbol"`
	Transaction     string                 `json:"transaction"`
	Quantity        float64                `json:"quantity"`
	Price           float64                `json:"price"`
	Basis           float64                `json:"basis"`
	PnL             *float64               `json:"pnl,omitempty"`
	Status          string                 `json:"status"`
	Message         string                 `json:"message,omitempty"`
	ExecutionID     string                 `json:"execution_id,omitempty"`
	MatchProvenance string                 `json:"match_provenance,omitempty"` // report row to close execution only
	OpeningMatch    string                 `json:"opening_match_provenance,omitempty"`
	Allocations     []SBIOpeningAllocation `json:"opening_allocations,omitempty"`
	Evidence        map[string]string      `json:"evidence,omitempty"`
}

type SBIMarginPnLImport struct {
	Headers []string
	Rows    []map[string]string
	Parsed  []SBIMarginPnLRow
}

var sbiPnLCode = regexp.MustCompile(`^[0-9A-Z]{4,5}$`)

func ReadSBIMarginPnLCSV(data []byte) (SBIMarginPnLImport, bool, error) {
	decoded, err := decodeSBI(data)
	if err != nil {
		return SBIMarginPnLImport{}, false, nil
	}
	r := csv.NewReader(bytes.NewReader(decoded))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return SBIMarginPnLImport{}, false, nil
	}
	for i, record := range records {
		headers := trimHeaders(record)
		if !hasSBIHeaderFields(headers, []string{"約定日", "取引", "平均取得価額"}) ||
			(!hasSBIHeaderFields(headers, []string{"銘柄コード"}) && !hasSBIHeaderFields(headers, []string{"銘柄名"})) ||
			(!hasSBIHeaderFields(headers, []string{"数量", "単価"}) && !hasSBIHeaderFields(headers, []string{"約定数量", "約定単価"})) ||
			(!hasSBIHeaderFields(headers, []string{"実現損益"}) && !hasSBIHeaderFields(headers, []string{"決済損益"}) && !hasSBIHeaderFields(headers, []string{"実現損益(税引前・円)"})) {
			continue
		}
		// Execution history remains the primary importer even if it happens to carry basis.
		if bytes.Contains(decoded, []byte("約定履歴照会")) {
			return SBIMarginPnLImport{}, false, nil
		}
		out := SBIMarginPnLImport{Headers: headers}
		for _, values := range records[i+1:] {
			if len(values) == 0 || strings.TrimSpace(strings.Join(values, "")) == "" {
				continue
			}
			row := map[string]string{}
			for j, h := range headers {
				if j < len(values) {
					row[h] = strings.TrimSpace(values[j])
				}
			}
			out.Rows = append(out.Rows, row)
			transaction := row["取引"]
			switch transaction {
			case "返済売":
				transaction = "信用返済売"
			case "返済買":
				transaction = "信用返済買"
			}
			if transaction != "信用返済売" && transaction != "信用返済買" && transaction != "現渡" && transaction != "売却" {
				continue
			}
			symbol := row["銘柄コード"]
			if symbol == "" {
				parts := strings.Fields(row["銘柄名"])
				if len(parts) > 0 && sbiPnLCode.MatchString(parts[len(parts)-1]) {
					symbol = parts[len(parts)-1]
				}
			}
			p := SBIMarginPnLRow{Date: row["約定日"], Symbol: symbol, Transaction: transaction}
			if len(sbiEvidence(row)) > 0 {
				p.Evidence = sbiEvidence(row)
			}
			quantity, qerr := sbiPnLNumber(firstValue(row, "数量", "約定数量"))
			price, perr := sbiPnLNumber(firstValue(row, "単価", "約定単価"))
			basis, berr := sbiPnLNumber(row["平均取得価額"])
			pnl, nerr := sbiPnLNumber(firstValue(row, "実現損益", "決済損益", "実現損益(税引前・円)"))
			if _, err := sbiPnLDate(p.Date); err != nil || p.Symbol == "" || qerr != nil || perr != nil || berr != nil || nerr != nil || quantity <= 0 || price <= 0 || basis <= 0 {
				p.Status, p.Message = "invalid_row", "invalid date, symbol, quantity, price, basis or realized P&L"
			} else {
				p.Quantity, p.Price, p.Basis, p.PnL = quantity, price, basis, &pnl
			}
			out.Parsed = append(out.Parsed, p)
		}
		return out, true, nil
	}
	return SBIMarginPnLImport{}, false, nil
}

func firstValue(row map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := row[key]; v != "" {
			return v
		}
	}
	return ""
}

func sbiPnLNumber(raw string) (float64, error) {
	v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(raw), ",", ""), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid number %q", raw)
	}
	return v, nil
}

func sbiPnLDate(raw string) (string, error) {
	for _, layout := range []string{"2006/01/02", "2006/1/2", "2006-01-02"} {
		if d, err := time.Parse(layout, raw); err == nil {
			return d.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("invalid date %q", raw)
}

// MatchSBIMarginPnL uses explicit SBI close provenance; indistinguishable results
// may match by multiplicity, but differing results never imply an intraday order.
func MatchSBIMarginPnL(rows []SBIMarginPnLRow, executions []store.Execution) []SBIMarginPnLRow {
	out := append([]SBIMarginPnLRow(nil), rows...)
	claimed := map[string]int{}
	scopes := map[string]bool{}
	for _, execution := range executions {
		scopes[execution.UserID+"|"+execution.AccountID] = true
	}
	for i := range out {
		row := &out[i]
		if row.Status == "invalid_row" {
			continue
		}
		row.ExecutionID, row.Message, row.MatchProvenance, row.OpeningMatch, row.Allocations = "", "", "", "", nil
		date, _ := sbiPnLDate(row.Date)
		var candidates, compatible []store.Execution
		credit := isSBICredit(*row)
		closeMatch := "exact"
		if raw := row.Evidence["約定数量"]; credit && raw != "" && !sbiDecimal(raw, strconv.FormatFloat(row.Quantity, 'f', -1, 64)) {
			row.Status, row.Message = "invalid_row", "SBI quantity cannot be represented safely; reconcile the source quantity before applying."
			continue
		}
		if credit && len(scopes) > 1 {
			row.Status, row.MatchProvenance, row.OpeningMatch, row.Message = "ambiguous_match", "unresolved", "unresolved", "Select one account before matching SBI settlement evidence."
			continue
		}
		for _, execution := range executions {
			var details map[string]any
			if !execution.Details.Valid || json.Unmarshal([]byte(execution.Details.String), &details) != nil {
				continue
			}
			lot, _ := details["lot"].(string)
			wantLot, wantSide := "sbi:margin-long", "sell"
			if row.Transaction == "現渡" || row.Transaction == "売却" {
				wantLot, wantSide = "sbi:cash", "sell"
			}
			if row.Transaction == "信用返済買" {
				wantLot, wantSide = "sbi:margin-short", "buy"
			}
			if (row.Transaction == "現渡" && details["settlement_type"] != "genwatashi") || (row.Transaction != "現渡" && details["settlement_type"] != nil) {
				continue
			}
			if execution.InstrumentType != "stock" || execution.Multiplier != 1 || lot != wantLot || details["position_effect"] != "reduce" || execution.Side != wantSide ||
				execution.Symbol != row.Symbol || (credit && (execution.Quantity != row.Quantity || execution.Price != row.Price)) || (!credit && (math.Abs(execution.Quantity-row.Quantity) > 0.000001 || math.Abs(execution.Price-row.Price) > 0.0001)) ||
				execution.ExecutedAt.In(time.FixedZone("JST", 9*3600)).Format("2006-01-02") != date {
				continue
			}
			candidates = append(candidates, execution)
			if credit && sbiConsistent(*row, execution) {
				compatible = append(compatible, execution)
			}
		}
		if credit && len(candidates) > 0 {
			if len(compatible) == 0 {
				row.Status, row.Message = "conflict", "Stored SBI settlement evidence disagrees; inspect the original reports."
				continue
			}
			candidates = compatible
		}
		if len(candidates) > 1 {
			// Only identical results with complete multiplicity can be assigned safely.
			count, identical := 0, true
			for _, other := range rows {
				otherDate, _ := sbiPnLDate(other.Date)
				if other.Status == "invalid_row" || otherDate != date || other.Symbol != row.Symbol || other.Transaction != row.Transaction || other.Quantity != row.Quantity || other.Price != row.Price {
					continue
				}
				if credit && !slices.ContainsFunc(candidates, func(e store.Execution) bool { return sbiConsistent(other, e) }) {
					continue
				}
				count++
				identical = identical && sameSBIResult(other, *row)
			}
			if identical && count == len(candidates) {
				sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
				for _, candidate := range candidates {
					if _, used := claimed[candidate.ID]; !used {
						candidates = []store.Execution{candidate}
						closeMatch = "equivalent"
						if credit {
							row.Message = "Equivalent close results matched by complete multiplicity; individual repayment identity is unknown."
						}
						break
					}
				}
			}
		}
		switch len(candidates) {
		case 0:
			row.Status = "no_matching_execution"
		case 1:
			candidate := candidates[0]
			row.ExecutionID = candidate.ID
			if credit {
				row.MatchProvenance = closeMatch
			}
			if priorIndex, exists := claimed[candidate.ID]; exists {
				prior := &out[priorIndex]
				if !sameSBIResult(*prior, *row) {
					prior.Status, prior.Message = "conflict", "duplicate source rows disagree for the same execution"
					row.Status, row.Message = "conflict", "duplicate source rows disagree for the same execution"
				} else {
					row.Status = "already_enriched"
				}
				continue
			}
			var details map[string]any
			_ = json.Unmarshal([]byte(candidate.Details.String), &details)
			if existing, ok := details["broker_reported_close_basis"].(float64); ok && existing != row.Basis {
				row.Status, row.Message = "conflict", fmt.Sprintf("existing basis %.4f", existing)
			} else if existing, ok := details["broker_reported_realized_pnl"].(float64); ok && row.PnL != nil && existing != *row.PnL {
				row.Status, row.Message = "conflict", fmt.Sprintf("existing realized P&L %.2f", existing)
			} else if _, ok := details["broker_reported_close_basis"].(float64); ok && details["broker_reported_realized_pnl"] != nil {
				row.Status = "already_enriched"
			} else {
				row.Status = "enrichable"
			}
			claimed[candidate.ID] = i
		default:
			sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
			row.Status, row.Message = "ambiguous_match", fmt.Sprintf("%d matching executions; provide distinguishing SBI settlement evidence", len(candidates))
			for _, candidate := range candidates {
				row.Message += " " + candidate.ID
			}
		}
	}
	matchSBIOpenings(out, executions)
	return out
}

func sameSBIResult(a, b SBIMarginPnLRow) bool {
	if a.Basis != b.Basis || a.PnL == nil || b.PnL == nil || *a.PnL != *b.PnL {
		return false
	}
	if isSBICredit(a) {
		for key, value := range a.Evidence {
			if !sbiTermEqual(key, value, b.Evidence[key]) {
				return false
			}
		}
		for key := range b.Evidence {
			if a.Evidence[key] == "" {
				return false
			}
		}
	}
	return true
}

func CommitSBIMarginPnL(ctx context.Context, q store.Querier, userID, accountID string, rows []SBIMarginPnLRow, getters ...trades.BarsGetter) (int, error) {
	for _, row := range rows {
		if row.Status != "enrichable" && row.Status != "already_enriched" && (isSBICredit(row) || row.Status != "no_matching_execution") {
			return 0, ErrSBIEnrichmentUnresolved
		}
	}
	all, err := q.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: userID, AccountID: accountID})
	if err != nil {
		return 0, err
	}
	fresh := MatchSBIMarginPnL(rows, all)
	for i := range rows {
		a, _ := json.Marshal(rows[i])
		b, _ := json.Marshal(fresh[i])
		if string(a) != string(b) {
			return 0, fmt.Errorf("matched execution changed; preview again")
		}
	}
	updated := 0
	for _, row := range rows {
		if row.Status != "enrichable" {
			continue
		}
		execution, err := q.GetExecution(ctx, store.GetExecutionParams{ID: row.ExecutionID, UserID: userID})
		if err != nil || execution.AccountID != accountID {
			return updated, fmt.Errorf("matched execution changed")
		}
		var details map[string]any
		if err := json.Unmarshal([]byte(execution.Details.String), &details); err != nil {
			return updated, err
		}
		details["broker_reported_close_basis"] = row.Basis
		if isSBICredit(row) {
			details["sbi_report_match_provenance"] = row.MatchProvenance
			details["sbi_opening_match_provenance"] = row.OpeningMatch
			allocations, err := json.Marshal(row.Allocations)
			if err != nil {
				return updated, err
			}
			details["sbi_opening_allocations"] = string(allocations)
		}
		details["close_basis_source"] = "broker_reported"
		details["realized_pnl_tax_basis"] = "pretax"
		details["realized_pnl_report"] = "sbi_realized_pnl_detail"
		if _, exists := details["broker_reported_realized_pnl"]; !exists && row.PnL != nil {
			details["broker_reported_realized_pnl"] = *row.PnL
			details["realized_pnl_source"] = "broker_reported"
		}
		value, err := json.Marshal(details)
		if err != nil {
			return updated, err
		}
		if err := q.UpdateExecutionContract(ctx, store.UpdateExecutionContractParams{ID: execution.ID, UserID: userID, Symbol: execution.Symbol, DedupHash: execution.DedupHash, Details: sql.NullString{String: string(value), Valid: true}}); err != nil {
			return updated, err
		}
		updated++
	}
	if updated > 0 {
		if err := trades.NewService(q, getters...).Regroup(ctx, userID, accountID); err != nil {
			return updated, err
		}
	}
	return updated, nil
}
