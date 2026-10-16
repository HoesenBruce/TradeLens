package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"time"

	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/store"
)

// realizedTrades replaces only explicit SBI groups with their individual closes.
// Review summaries keep their closed-group population and all original fields.
func (s *Server) realizedTrades(ctx context.Context, uid string, f Filters, meta *currencyMetadata, closed []store.Trade) ([]analytics.ClosedTrade, error) {
	accounts, err := s.deps.Store.ListAccounts(ctx, uid)
	if err != nil {
		return nil, failLoad(err, "could not load realized accounts")
	}
	sbiIDs := map[string]bool{}
	out := []analytics.ClosedTrade{}
	rates := map[string]float64{meta.Currency: 1}
	for _, fx := range meta.FXRates {
		rates[fx.From] = fx.Rate
	}
	for _, a := range accounts {
		if !f.matchAccount(a.ID) || (len(f.AccountIDs) == 0 && !f.IncludeBacktest && a.AccountType == AccountTypeBacktest) {
			continue
		}
		rows, err := s.deps.Store.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: uid, AccountID: a.ID})
		if err != nil {
			return nil, failLoad(err, "could not load realized executions")
		}
		hasSBI := false
		dateOnly := map[string]bool{}
		for _, row := range rows {
			var d struct {
				Lot       string `json:"lot"`
				Precision string `json:"source_time_precision"`
			}
			_ = json.Unmarshal([]byte(row.Details.String), &d)
			if d.Lot == "sbi:cash" || d.Lot == "sbi:margin-long" || d.Lot == "sbi:margin-short" {
				sbiIDs[row.ID] = true
				dateOnly[row.ID] = d.Precision == "date"
				hasSBI = hasSBI || f.matchSymbol(row.Symbol)
			}
		}
		if !hasSBI {
			continue
		}
		if f.Symbol != "" {
			selected := rows[:0]
			for _, row := range rows {
				if f.matchSymbol(row.Symbol) {
					selected = append(selected, row)
				}
			}
			rows = selected
		}
		results, err := s.deps.Trades.SBIAccounting(ctx, rows)
		if err != nil {
			return nil, failLoad(err, "could not reconcile SBI realized results")
		}
		for _, result := range results {
			tradeIndex := map[string]int{}
			for i, tr := range result.Trades {
				for _, id := range tr.ExecutionIDs {
					tradeIndex[id] = i
				}
			}
			unavailable := map[string]bool{}
			for _, id := range result.UnavailableCloseIDs {
				unavailable[id] = true
			}
			for _, close := range result.RealizedCloses {
				if close.Source == "position_conversion" || close.Source == "position_settlement" {
					continue
				}
				if unavailable[close.ExecutionID] && close.Source != "broker_reported" {
					continue
				}
				index, exists := tradeIndex[close.ExecutionID]
				if !exists {
					continue
				}
				tr := result.Trades[index]
				sample := store.Trade{Direction: tr.Direction, OpenedAt: tr.OpenedAt}
				if tr.ClosedAt != nil {
					sample.ClosedAt = sql.NullTime{Time: *tr.ClosedAt, Valid: true}
				}
				if tr.TimeInTradeSecs != nil {
					sample.TimeInTradeSecs = sql.NullInt64{Int64: *tr.TimeInTradeSecs, Valid: true}
				}
				at := close.Date
				matchesDate := f.matchClosed(tr.Symbol, at)
				if dateOnly[close.ExecutionID] {
					// Date-only storage anchors never establish an intraday realization time.
					day := at.In(time.FixedZone("Asia/Tokyo", 9*3600)).Format(time.DateOnly)
					at, _ = time.ParseInLocation(time.DateOnly, day, f.Loc)
					matchesDate = f.matchSymbol(tr.Symbol) && (f.From == nil || day >= f.From.In(f.Loc).Format(time.DateOnly)) && (f.To == nil || day <= f.To.In(f.Loc).Format(time.DateOnly))
				}
				if !f.matchSideDuration(sample) || !matchesDate {
					continue
				}
				rate, err := s.analyticsRate(ctx, meta, rates, a.BaseCurrency)
				if err != nil {
					return nil, err
				}
				net, fees := close.Pnl*rate, close.Fees*rate
				if math.IsNaN(net) || math.IsInf(net, 0) || math.IsNaN(fees) || math.IsInf(fees, 0) {
					return nil, Fail(502, "invalid_money", "invalid realized monetary input", nil)
				}
				out = append(out, analytics.ClosedTrade{NetPnl: net, GrossPnl: net + fees, FeesTotal: fees, OpenedAt: tr.OpenedAt, ClosedAt: at})
			}
		}
	}
	for _, tr := range closed {
		if !sbiIDs[tr.ID] {
			out = append(out, toClosedTrades([]store.Trade{tr})...)
		}
	}
	return out, nil
}
