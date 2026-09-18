package accountvalue

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/money"
	"github.com/tradermemos/api/internal/positions"
	"github.com/tradermemos/api/internal/store"
)

var tokyo = time.FixedZone("Asia/Tokyo", 9*60*60)

type Request struct {
	Executions       []store.Execution
	CashTransactions []store.CashTransaction
	// MarketSessions is the authoritative exchange calendar; weekends and JPX holidays must be omitted.
	MarketSessions       []time.Time
	ConfirmedSuspensions map[Instrument]map[string]bool
}

type Instrument struct {
	Symbol         string
	InstrumentType string
}

type Warning struct {
	Code       string `json:"code"`
	Instrument string `json:"instrument,omitempty"`
	Date       string `json:"date"`
	Message    string `json:"message"`
}

type Point struct {
	Date                  string    `json:"date"`
	EstimatedAccountValue *float64  `json:"estimated_account_value"`
	ContributedCapital    float64   `json:"contributed_capital"`
	CashBalance           float64   `json:"cash_balance"`
	OpenPositionValue     *float64  `json:"open_position_value"`
	RealizedPnL           float64   `json:"realized_pnl"`
	UnrealizedPnL         *float64  `json:"unrealized_pnl"`
	Status                string    `json:"status"`
	Warnings              []Warning `json:"warnings"`
}

type Account struct {
	AccountID string  `json:"account_id"`
	Points    []Point `json:"points"`
}

type Result struct {
	Timezone         string    `json:"timezone"`
	AdjustmentStatus string    `json:"adjustment_status"`
	Accounts         []Account `json:"accounts"`
}

type Service struct {
	getBars func(context.Context, marketdata.Request) (marketdata.Response, error)
}

func NewService(getBars func(context.Context, marketdata.Request) (marketdata.Response, error)) *Service {
	return &Service{getBars: getBars}
}

// MarketSessions returns authoritative JPX sessions from a broad-market ETF
// fetched through the same shared market-data path used for valuation.
func (s *Service) MarketSessions(ctx context.Context, from, to time.Time) ([]time.Time, error) {
	response, err := s.getBars(ctx, marketdata.Request{
		Symbol: "1306", InstrumentType: "stock", Interval: "D",
		From: from, To: to.AddDate(0, 0, 1),
	})
	if err != nil {
		return nil, err
	}
	dates := map[string]time.Time{}
	for _, bar := range response.Bars {
		date, err := time.ParseInLocation(time.DateOnly, bar.MarketDate, tokyo)
		if err == nil && !date.Before(from) && !date.After(to) {
			dates[bar.MarketDate] = date
		}
	}
	out := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		out = append(out, date)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out, nil
}

// Combine sums same-currency account series into the public portfolio series.
func Combine(result Result) []Point {
	points := map[string]Point{}
	complete := map[string]bool{}
	for _, account := range result.Accounts {
		for _, point := range account.Points {
			combined, exists := points[point.Date]
			if !exists {
				combined = Point{Date: point.Date, Status: "complete"}
				complete[point.Date] = true
			}
			combined.ContributedCapital += point.ContributedCapital
			combined.CashBalance += point.CashBalance
			combined.RealizedPnL += point.RealizedPnL
			combined.Status = worseStatus(combined.Status, point.Status)
			combined.Warnings = append(combined.Warnings, point.Warnings...)
			if point.EstimatedAccountValue == nil || point.OpenPositionValue == nil || point.UnrealizedPnL == nil {
				complete[point.Date] = false
			} else {
				add(&combined.EstimatedAccountValue, *point.EstimatedAccountValue)
				add(&combined.OpenPositionValue, *point.OpenPositionValue)
				add(&combined.UnrealizedPnL, *point.UnrealizedPnL)
			}
			points[point.Date] = combined
		}
	}
	out := make([]Point, 0, len(points))
	for date, point := range points {
		point.ContributedCapital = money.Round2(point.ContributedCapital)
		point.CashBalance = money.Round2(point.CashBalance)
		point.RealizedPnL = money.Round2(point.RealizedPnL)
		if !complete[date] {
			point.EstimatedAccountValue = nil
			point.OpenPositionValue = nil
			point.UnrealizedPnL = nil
		} else {
			*point.EstimatedAccountValue = money.Round2(*point.EstimatedAccountValue)
			*point.OpenPositionValue = money.Round2(*point.OpenPositionValue)
			*point.UnrealizedPnL = money.Round2(*point.UnrealizedPnL)
		}
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func add(total **float64, value float64) {
	if *total == nil {
		zero := 0.0
		*total = &zero
	}
	**total += value
}

func worseStatus(a, b string) string {
	priority := map[string]int{
		"complete": 0, "incomplete_missing_price": 1,
		"incomplete_invalid_sequence": 2, "unsupported_corporate_action": 3,
	}
	if priority[b] > priority[a] {
		return b
	}
	return a
}

func (s *Service) Reconstruct(ctx context.Context, req Request) (Result, error) {
	snapshots := positions.Replay(req.Executions, req.MarketSessions)
	result := Result{Timezone: "Asia/Tokyo", AdjustmentStatus: "unadjusted"}
	if len(snapshots) == 0 {
		return result, nil
	}

	responses, fetchErrors := s.loadPrices(ctx, req.Executions, snapshots)
	if len(responses) == 0 && len(fetchErrors) > 0 {
		return Result{}, fmt.Errorf("load historical prices: %w", firstError(fetchErrors))
	}

	accountIDs := collectAccounts(req.Executions, req.CashTransactions)
	ledgers := ledgerStates(req.CashTransactions, snapshots)
	for _, accountID := range accountIDs {
		account := Account{AccountID: accountID, Points: make([]Point, 0, len(snapshots))}
		lastClose := map[Instrument]float64{}
		for i, snapshot := range snapshots {
			replay := replayAccount(snapshot, accountID)
			ledger := ledgerAt(ledgers, accountID, i)
			point := Point{
				Date: snapshot.Date, CashBalance: money.Round2(replay.CashDelta + ledger.cash),
				ContributedCapital: money.Round2(ledger.contributed),
				RealizedPnL:        money.Round2(replay.RealizedPnL + ledger.realized), Status: "complete",
			}
			for _, warning := range snapshot.Warnings {
				if warning.AccountID == accountID {
					point.invalid(warning.Code, "", warning.Message)
				}
			}
			for _, warning := range ledger.warnings {
				point.invalid(warning.Code, warning.Instrument, warning.Message)
			}
			s.valuePositions(&point, replay.Positions, responses, fetchErrors, lastClose, req.ConfirmedSuspensions)
			account.Points = append(account.Points, point)
		}
		result.Accounts = append(result.Accounts, account)
	}
	return result, nil
}

func (s *Service) loadPrices(ctx context.Context, executions []store.Execution, snapshots []positions.Snapshot) (map[Instrument]marketdata.Response, map[Instrument]error) {
	instruments := map[Instrument]positions.Position{}
	for _, snapshot := range snapshots {
		for _, account := range snapshot.Accounts {
			for _, position := range account.Positions {
				instruments[Instrument{position.Symbol, position.InstrumentType}] = position
			}
		}
	}
	responses := map[Instrument]marketdata.Response{}
	errs := map[Instrument]error{}
	to, _ := time.ParseInLocation(time.DateOnly, snapshots[len(snapshots)-1].Date, tokyo)
	for key, instrument := range instruments {
		from := earliestExecution(executions, instrument.Symbol, instrument.InstrumentType, to)
		response, err := s.getBars(ctx, marketdata.Request{
			Symbol: instrument.Symbol, InstrumentType: instrument.InstrumentType,
			Interval: "D", From: from, To: to.AddDate(0, 0, 1),
		})
		if err != nil {
			errs[key] = err
			continue
		}
		responses[key] = response
	}
	return responses, errs
}

func (s *Service) valuePositions(point *Point, held []positions.Position, responses map[Instrument]marketdata.Response, fetchErrors map[Instrument]error, lastClose map[Instrument]float64, suspensions map[Instrument]map[string]bool) {
	openValue, unrealized := 0.0, 0.0
	for _, position := range held {
		key := Instrument{position.Symbol, position.InstrumentType}
		response, ok := responses[key]
		if !ok {
			message := "historical market data is unavailable"
			if err := fetchErrors[key]; err != nil {
				message = err.Error()
			}
			point.missing(position.Symbol, message)
			continue
		}
		if response.AdjustmentStatus != "unadjusted" {
			point.unsupported(position.Symbol, "market data is not unadjusted")
			continue
		}
		for _, candidate := range marketdata.FindCorporateActionCandidates(response) {
			if candidate.Status != "rejected" && candidate.EffectiveDate <= point.Date {
				point.unsupported(position.Symbol, fmt.Sprintf("unconfirmed %s candidate on %s", candidate.CandidateType, candidate.EffectiveDate))
			}
		}
		close, found := closeOn(response.Bars, point.Date)
		if !found && suspensions[key][point.Date] {
			close, found = lastClose[key]
			if !found {
				close, found = lastCloseBefore(response.Bars, point.Date)
			}
			if found {
				point.warn("carried_forward_suspension_price", position.Symbol, "carried forward the last close for a confirmed suspension")
			}
		}
		if !found {
			point.missing(position.Symbol, "no unadjusted close is available for this market session")
			continue
		}
		lastClose[key] = close
		pnl := (close - position.AverageCost) * position.Quantity * position.Multiplier
		value := pnl
		if position.Kind == positions.CashLong {
			value = close * position.Quantity * position.Multiplier
		} else if position.Kind == positions.MarginShort {
			pnl = -pnl
			value = pnl
		}
		openValue += value
		unrealized += pnl
	}
	if point.Status == "complete" {
		openValue = money.Round2(openValue)
		unrealized = money.Round2(unrealized)
		estimated := money.Round2(point.CashBalance + openValue)
		point.OpenPositionValue = &openValue
		point.UnrealizedPnL = &unrealized
		point.EstimatedAccountValue = &estimated
	}
}

func (p *Point) warn(code, instrument, message string) {
	p.Warnings = append(p.Warnings, Warning{Code: code, Instrument: instrument, Date: p.Date, Message: message})
}

func (p *Point) missing(instrument, message string) {
	if p.Status == "complete" {
		p.Status = "incomplete_missing_price"
	}
	p.warn("missing_price", instrument, message)
}

func (p *Point) invalid(code, instrument, message string) {
	if p.Status != "unsupported_corporate_action" {
		p.Status = "incomplete_invalid_sequence"
	}
	p.warn(code, instrument, message)
}

func (p *Point) unsupported(instrument, message string) {
	p.Status = "unsupported_corporate_action"
	p.warn("unsupported_corporate_action", instrument, message)
}

type ledgerState struct {
	cash, contributed, realized float64
	warnings                    []Warning
}

func ledgerStates(rows []store.CashTransaction, snapshots []positions.Snapshot) map[string][]ledgerState {
	byAccount := map[string][]store.CashTransaction{}
	for _, row := range rows {
		byAccount[row.AccountID] = append(byAccount[row.AccountID], row)
	}
	out := map[string][]ledgerState{}
	for accountID, accountRows := range byAccount {
		sort.SliceStable(accountRows, func(i, j int) bool {
			if accountRows[i].OccurredAt.Equal(accountRows[j].OccurredAt) {
				return accountRows[i].ID < accountRows[j].ID
			}
			return accountRows[i].OccurredAt.Before(accountRows[j].OccurredAt)
		})
		state, next := ledgerState{}, 0
		for _, snapshot := range snapshots {
			date, _ := time.ParseInLocation(time.DateOnly, snapshot.Date, tokyo)
			end := date.AddDate(0, 0, 1)
			for next < len(accountRows) && accountRows[next].OccurredAt.Before(end) {
				row := accountRows[next]
				state.cash += row.Amount
				switch row.Type {
				case "deposit", "withdrawal":
					state.contributed += row.Amount
				case "fee":
					state.realized += row.Amount
				case "dividend", "interest", "adjustment":
				default:
					state.warnings = append(state.warnings, Warning{Code: "invalid_cash_transaction", Message: "unsupported cash transaction type " + row.Type})
				}
				next++
			}
			copy := state
			copy.warnings = append([]Warning(nil), state.warnings...)
			out[accountID] = append(out[accountID], copy)
		}
	}
	return out
}

func collectAccounts(executions []store.Execution, cash []store.CashTransaction) []string {
	set := map[string]bool{}
	for _, execution := range executions {
		set[execution.AccountID] = true
	}
	for _, row := range cash {
		set[row.AccountID] = true
	}
	out := make([]string, 0, len(set))
	for accountID := range set {
		out = append(out, accountID)
	}
	sort.Strings(out)
	return out
}

func replayAccount(snapshot positions.Snapshot, accountID string) positions.AccountSnapshot {
	for _, account := range snapshot.Accounts {
		if account.AccountID == accountID {
			return account
		}
	}
	return positions.AccountSnapshot{AccountID: accountID}
}

func ledgerAt(ledgers map[string][]ledgerState, accountID string, index int) ledgerState {
	if states := ledgers[accountID]; index < len(states) {
		return states[index]
	}
	return ledgerState{}
}

func earliestExecution(executions []store.Execution, symbol, instrument string, fallback time.Time) time.Time {
	earliest := fallback
	for _, execution := range executions {
		if execution.Symbol == symbol && execution.InstrumentType == instrument && execution.ExecutedAt.Before(earliest) {
			earliest = execution.ExecutedAt
		}
	}
	return earliest
}

func closeOn(bars []marketdata.Bar, date string) (float64, bool) {
	for _, bar := range bars {
		if bar.MarketDate == date && bar.Close > 0 {
			return bar.Close, true
		}
	}
	return 0, false
}

func firstError(errs map[Instrument]error) error {
	for _, err := range errs {
		return err
	}
	return nil
}

func lastCloseBefore(bars []marketdata.Bar, date string) (float64, bool) {
	latestDate, latestClose := "", 0.0
	for _, bar := range bars {
		if bar.MarketDate < date && bar.MarketDate > latestDate && bar.Close > 0 {
			latestDate, latestClose = bar.MarketDate, bar.Close
		}
	}
	return latestClose, latestDate != ""
}
