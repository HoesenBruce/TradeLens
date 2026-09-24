package trades

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/money"
)

type BarsGetter func(context.Context, marketdata.Request) (marketdata.Response, error)

// Open partial closes keep nil trade P&L and separate realized results.
func reconcileReportedTrades(fills []Execution, result *AccountingResult) {
	reported := map[string]bool{}
	closes := map[string]RealizedClose{}
	missing := map[string]bool{}
	for _, f := range fills {
		if f.BrokerReportedPnl != nil {
			reported[f.ID] = true
		}
	}
	for _, c := range result.RealizedCloses {
		closes[c.ExecutionID] = c
	}
	for _, id := range result.UnavailableCloseIDs {
		missing[id] = true
	}
	for i := range result.Trades {
		tr := &result.Trades[i]
		if tr.Status != "closed" {
			continue
		}
		hasReported, unavailable := false, false
		net := 0.0
		for _, id := range tr.ExecutionIDs {
			hasReported = hasReported || reported[id]
			unavailable = unavailable || missing[id]
			net += closes[id].Pnl
		}
		if !hasReported {
			continue
		}
		if unavailable {
			tr.NetPnl, tr.GrossPnl, tr.ReturnPct = nil, nil, nil
			tr.AccountingWarning = "P&L unavailable: incomplete SBI settlement evidence."
			continue
		}
		net = money.Round2(net)
		gross := money.Round2(net + tr.FeesTotal)
		tr.NetPnl, tr.GrossPnl = &net, &gross
		if base := tr.AvgEntryPrice * tr.QtyOpened; base != 0 {
			tr.ReturnPct = f64(money.Round2(net / base * 100))
		}
	}
}

func (s *Service) checkSplitBoundaries(ctx context.Context, fills []Execution, result *AccountingResult) error {
	if len(fills) == 0 || fills[0].InstrumentType != "stock" {
		return nil
	}
	from, to := fills[0].ExecutedAt, fills[0].ExecutedAt
	byID := map[string]Execution{}
	for _, f := range fills {
		byID[f.ID] = f
		if f.ExecutedAt.Before(from) {
			from = f.ExecutedAt
		}
		if f.ExecutedAt.After(to) {
			to = f.ExecutedAt
		}
	}
	zone, _ := time.LoadLocation(marketdata.MarketTimezone(marketdata.Request{Symbol: fills[0].Symbol, InstrumentType: "stock"}))
	if from.In(zone).Format("2006-01-02") == to.In(zone).Format("2006-01-02") {
		return nil
	}
	if s.GetBars == nil {
		for i := range result.Trades {
			result.Trades[i].AccountingWarning += " Corporate-action check unavailable: market data is not configured."
		}
		return nil
	}
	response, err := s.GetBars(ctx, marketdata.Request{Symbol: fills[0].Symbol, InstrumentType: "stock", Interval: "D", From: from.AddDate(0, 0, -7), To: to.AddDate(0, 0, 1)})
	if err != nil {
		return fmt.Errorf("check corporate actions for %s: %w", fills[0].Symbol, err)
	}
	candidates := append([]marketdata.CorporateActionCandidate(nil), response.CorporateActions...)
	for _, detected := range marketdata.FindCorporateActionCandidates(response) {
		known := false
		for _, candidate := range response.CorporateActions {
			known = known || candidate.EffectiveDate == detected.EffectiveDate
		}
		if !known {
			candidates = append(candidates, detected)
		}
	}
	loc := time.UTC
	if zone, err := time.LoadLocation(response.Timezone); err == nil {
		loc = zone
	}
	for i := range result.Trades {
		tr := &result.Trades[i]
		last := tr.OpenedAt
		reliable := tr.Status == "closed" && strings.HasPrefix(fills[0].LotKey, "sbi:margin-")
		closeCount := 0
		for _, id := range tr.ExecutionIDs {
			f := byID[id]
			if f.ExecutedAt.After(last) {
				last = f.ExecutedAt
			}
			if (tr.Direction == "long" && f.Side == "sell") || (tr.Direction == "short" && f.Side == "buy") {
				closeCount++
				reliable = reliable && f.BrokerReportedPnl != nil && f.ConversionType == ""
			}
		}
		reliable = reliable && closeCount > 0 && tr.NetPnl != nil
		for _, c := range candidates {
			if c.Status == "rejected" {
				continue
			}
			date, err := time.Parse("2006-01-02", c.EffectiveDate)
			if err != nil {
				continue
			}
			day := date.Format("2006-01-02")
			if day <= tr.OpenedAt.In(loc).Format("2006-01-02") || day > last.In(loc).Format("2006-01-02") {
				continue
			}
			tr.AccountingWarning = fmt.Sprintf("Corporate-action boundary: %s on %s (ratio %.3g). Quantities and cost basis are not split-adjusted.", c.CandidateType, day, c.SuspectedRatio)
			tr.ReturnPct = nil
			if reliable {
				tr.AccountingWarning += " Net P&L uses SBI settlement values."
			} else {
				tr.NetPnl, tr.GrossPnl = nil, nil
				tr.AccountingWarning += " P&L unavailable: reliable settlement evidence is missing."
			}
			break
		}
	}
	return nil
}
