package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/importer"
	"github.com/tradermemos/api/internal/store"
)

func fptr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	return &n.Float64
}
func tptr(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}
func iptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}
func sptr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	s := n.String
	return &s
}

// tradeDTO flattens sqlc's sql.Null* fields into JSON-friendly nullable values
// (a number or null), so clients see net_pnl: 200 rather than {Float64,Valid}.
type tradeDTO struct {
	SourcePnlCurrency string      `json:"source_pnl_currency,omitempty"`
	AccountingWarning string      `json:"accounting_warning,omitempty"`
	ID                string      `json:"id"`
	AccountID         string      `json:"account_id"`
	Symbol            string      `json:"symbol"`
	InstrumentType    string      `json:"instrument_type"`
	Direction         string      `json:"direction"`
	Status            string      `json:"status"`
	OpenedAt          time.Time   `json:"opened_at"`
	ClosedAt          *time.Time  `json:"closed_at"`
	QtyOpened         float64     `json:"qty_opened"`
	QtyRemaining      float64     `json:"qty_remaining"`
	AvgEntryPrice     float64     `json:"avg_entry_price"`
	AvgExitPrice      *float64    `json:"avg_exit_price"`
	GrossPnl          *float64    `json:"gross_pnl"`
	FeesTotal         float64     `json:"fees_total"`
	NetPnl            *float64    `json:"net_pnl"`
	PnlCurrency       string      `json:"pnl_currency"`
	ReturnPct         *float64    `json:"return_pct"`
	TimeInTradeSecs   *int64      `json:"time_in_trade_secs"`
	Notes             string      `json:"notes"`
	Tags              []store.Tag `json:"tags"`
	// call/put for option trades, resolved from the fills' contract details
	// (OCC symbol as fallback). Absent for non-options and unresolvable rows.
	OptionRight *string  `json:"option_right,omitempty"`
	StockName   *string  `json:"stock_name,omitempty"`
	InitialRisk *float64 `json:"initial_risk,omitempty"`
	// Journal quick-filter fields, filled on list rows so clients can filter by
	// setup/emotion/ratings without a detail fetch. The detail DTO's own fields
	// shadow the emotion/rating ones (same JSON keys at shallower depth).
	SetupID        *string `json:"setup_id,omitempty"`
	EmotionalState string  `json:"emotional_state,omitempty"`
	Confidence     *int64  `json:"confidence,omitempty"`
	TradeQuality   *int64  `json:"trade_quality,omitempty"`
}

func toTradeDTO(t store.Trade, tags []store.Tag) tradeDTO {
	if tags == nil {
		tags = []store.Tag{}
	}
	return tradeDTO{
		AccountingWarning: t.AccountingWarning,
		ID:                t.ID, AccountID: t.AccountID, Symbol: t.Symbol,
		InstrumentType: t.InstrumentType, Direction: t.Direction, Status: t.Status,
		OpenedAt: t.OpenedAt, ClosedAt: tptr(t.ClosedAt), QtyOpened: t.QtyOpened,
		QtyRemaining: t.QtyRemaining, AvgEntryPrice: t.AvgEntryPrice, AvgExitPrice: fptr(t.AvgExitPrice),
		GrossPnl: fptr(t.GrossPnl), FeesTotal: t.FeesTotal, NetPnl: fptr(t.NetPnl),
		PnlCurrency: t.PnlCurrency, ReturnPct: fptr(t.ReturnPct),
		TimeInTradeSecs: iptr(t.TimeInTradeSecs), Notes: t.Notes, Tags: tags,
	}
}

// optionRightFrom resolves call/put from one fill's contract details, falling
// back to the OCC/word-marked symbol when the broker left details sparse —
// the same order the grouping service uses to partition contracts.
func optionRightFrom(details sql.NullString, symbol string) string {
	if details.Valid && details.String != "" {
		var m map[string]any
		if json.Unmarshal([]byte(details.String), &m) == nil {
			if s, ok := m["option_right"].(string); ok {
				if r := strings.ToLower(strings.TrimSpace(s)); r == "call" || r == "put" {
					return r
				}
			}
		}
	}
	return importer.InferOptionRight(symbol)
}

// optionRightsByTrade maps trade id → call/put from the user's option fills;
// the first fill that resolves wins (fills arrive ordered by execution time).
func optionRightsByTrade(rows []store.ListOptionExecutionDetailsForUserRow) map[string]string {
	out := make(map[string]string)
	for _, r := range rows {
		if _, done := out[r.TradeID]; done {
			continue
		}
		if right := optionRightFrom(r.Details, r.Symbol); right != "" {
			out[r.TradeID] = right
		}
	}
	return out
}

func stockNamesByTrade(rows []store.ListOptionExecutionDetailsForUserRow) map[string]string {
	out := make(map[string]string)
	for _, r := range rows {
		if out[r.TradeID] != "" || !r.Details.Valid || r.Details.String == "" {
			continue
		}
		var details map[string]string
		if json.Unmarshal([]byte(r.Details.String), &details) == nil {
			if name := strings.TrimSpace(details["stock_name"]); name != "" {
				out[r.TradeID] = name
			}
		}
	}
	return out
}

// optionRightFromFills is the single-trade variant used by the detail DTO,
// where the fills are already loaded.
func optionRightFromFills(fills []store.Execution) string {
	for _, f := range fills {
		if right := optionRightFrom(f.Details, f.Symbol); right != "" {
			return right
		}
	}
	return ""
}

// executionDTO flattens sql.Null* and parses details JSON so clients get
// details: {option_right, strike, expiry} rather than {String, Valid}.
type executionDTO struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	AccountID      string            `json:"account_id"`
	ExternalID     *string           `json:"external_id"`
	Symbol         string            `json:"symbol"`
	InstrumentType string            `json:"instrument_type"`
	Side           string            `json:"side"`
	TradeType      string            `json:"trade_type"`
	Quantity       float64           `json:"quantity"`
	Price          float64           `json:"price"`
	Fees           float64           `json:"fees"`
	Commission     float64           `json:"commission"`
	ExecutedAt     time.Time         `json:"executed_at"`
	Multiplier     float64           `json:"multiplier"`
	Details        map[string]string `json:"details"`
	ImportBatchID  *string           `json:"import_batch_id"`
	DedupHash      string            `json:"dedup_hash"`
	CreatedAt      time.Time         `json:"created_at"`
}

func toExecutionDTO(e store.Execution) executionDTO {
	details := map[string]string{}
	if e.Details.Valid && e.Details.String != "" {
		_ = json.Unmarshal([]byte(e.Details.String), &details)
		if details == nil {
			details = map[string]string{}
		}
	}
	return executionDTO{
		ID: e.ID, UserID: e.UserID, AccountID: e.AccountID,
		ExternalID: sptr(e.ExternalID), Symbol: e.Symbol, InstrumentType: e.InstrumentType,
		Side: e.Side, TradeType: tradeType(e.Side, details), Quantity: e.Quantity, Price: e.Price, Fees: e.Fees, Commission: e.Commission,
		ExecutedAt: e.ExecutedAt, Multiplier: e.Multiplier, Details: details,
		ImportBatchID: sptr(e.ImportBatchID), DedupHash: e.DedupHash, CreatedAt: e.CreatedAt,
	}
}

func tradeType(side string, details map[string]string) string {
	if details["settlement_type"] == "genwatashi" && (details["lot"] == "sbi:cash" || details["lot"] == "sbi:margin-short") {
		return "genwatashi"
	}
	switch details["lot"] {
	case "sbi:cash":
		return "cash_" + side
	case "sbi:margin-long":
		if side == "buy" {
			return "margin_long_open"
		}
		return "margin_long_close"
	case "sbi:margin-short":
		if side == "sell" {
			return "margin_short_open"
		}
		return "margin_short_close"
	default:
		return side
	}
}

func toExecutionDTOs(rows []store.Execution) []executionDTO {
	out := make([]executionDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, toExecutionDTO(e))
	}
	return out
}

// Mixed scopes are rejected unless the endpoint normalizes each monetary input.
var errMixedCurrencies = errors.New("selected accounts mix base currencies")

// portfolioCurrencies resolves exactly the accounts included by the trade loaders.
func (s *Server) portfolioCurrencies(ctx context.Context, userID string, f Filters) ([]string, error) {
	accounts, err := s.deps.Store.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, id := range f.AccountIDs {
		if !slices.ContainsFunc(accounts, func(a store.Account) bool { return a.ID == id }) {
			return nil, errUnknownCurrency
		}
	}
	currencies := []string{}
	for _, a := range accounts {
		if !f.matchAccount(a.ID) || (len(f.AccountIDs) == 0 && !f.IncludeBacktest && a.AccountType == AccountTypeBacktest) {
			continue
		}
		if a.BaseCurrency == "" {
			return nil, errUnknownCurrency
		}
		if !slices.Contains(currencies, a.BaseCurrency) {
			currencies = append(currencies, a.BaseCurrency)
		}
	}
	return currencies, nil
}

func (s *Server) portfolioCurrency(ctx context.Context, userID string, f Filters) (string, error) {
	currencies, err := s.portfolioCurrencies(ctx, userID, f)
	if err != nil {
		return "", err
	}
	if len(currencies) > 1 {
		return "", errMixedCurrencies
	}
	if len(currencies) == 0 {
		return "", nil
	}
	return currencies[0], nil
}

var errUnknownCurrency = errors.New("account scope currency could not be resolved")

func (s *Server) checkPortfolioCurrency(ctx context.Context, userID string, f Filters) error {
	// Full exports preserve individual rows, rather than summing money.
	if f.IncludeBacktest {
		return nil
	}
	_, err := s.portfolioCurrency(ctx, userID, f)
	return err
}

// failLoad maps a loader error to an API error: the mixed-currency guard is
// the caller's mistake (400), anything else is internal.
func failLoad(err error, msg string) error {
	if errors.Is(err, errUnknownCurrency) {
		return Fail(http.StatusBadRequest, "unknown_currency", errUnknownCurrency.Error(), nil)
	}
	if errors.Is(err, errMixedCurrencies) {
		return Fail(http.StatusBadRequest, "mixed_currencies", errMixedCurrencies.Error(), nil)
	}
	return Fail(http.StatusInternalServerError, "internal", msg, nil)
}

// loadClosedTrades fetches a user's closed trades (optionally account-scoped in
// SQL) and applies symbol/date filters in Go. Unscoped results skip backtest
// accounts unless f.IncludeBacktest is set.
func (s *Server) loadClosedTrades(ctx context.Context, userID string, f Filters) ([]store.Trade, error) {
	if err := s.checkPortfolioCurrency(ctx, userID, f); err != nil {
		return nil, err
	}
	return s.loadClosedTradeRows(ctx, userID, f)
}

// Only callers that validate scope and normalize money may bypass the currency guard.
func (s *Server) loadClosedTradeRows(ctx context.Context, userID string, f Filters) ([]store.Trade, error) {
	rows, err := s.deps.Store.ListClosedTrades(ctx, store.ListClosedTradesParams{
		UserID:    userID,
		AccountID: f.accountNarg(),
	})
	if err != nil {
		return nil, err
	}
	excluded, err := s.backtestAccountIDs(ctx, userID, f)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, t := range rows {
		if !excluded[t.AccountID] && f.matchTrade(t) {
			out = append(out, t)
		}
	}
	return out, nil
}

// loadTrades fetches open and/or closed trades for the trade log / home page.
// Date filters default to opened_at (legacy). Pass date_basis=close to filter
// by closed_at so calendar / realized-day views stay consistent.
func (s *Server) loadTrades(ctx context.Context, userID string, f Filters) ([]store.Trade, error) {
	if err := s.checkPortfolioCurrency(ctx, userID, f); err != nil {
		return nil, err
	}
	return s.loadTradeRows(ctx, userID, f)
}

func (s *Server) loadTradeRows(ctx context.Context, userID string, f Filters) ([]store.Trade, error) {
	rows, err := s.deps.Store.ListTrades(ctx, store.ListTradesParams{
		UserID:    userID,
		AccountID: f.accountNarg(),
		Status:    statusArg(f.Status),
	})
	if err != nil {
		return nil, err
	}
	excluded, err := s.backtestAccountIDs(ctx, userID, f)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, t := range rows {
		if !excluded[t.AccountID] && f.matchListDate(t) && f.matchSideDuration(t) {
			out = append(out, t)
		}
	}
	return out, nil
}
