package positions

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/tradermemos/api/internal/money"
	"github.com/tradermemos/api/internal/store"
)

const epsilon = 1e-9

var tokyo = time.FixedZone("Asia/Tokyo", 9*60*60)

type Kind string

const (
	CashLong    Kind = "cash_long"
	MarginLong  Kind = "margin_long"
	MarginShort Kind = "margin_short"
)

type Position struct {
	AccountID      string  `json:"account_id"`
	Symbol         string  `json:"symbol"`
	InstrumentType string  `json:"instrument_type"`
	Kind           Kind    `json:"kind"`
	Lot            string  `json:"lot,omitempty"`
	Quantity       float64 `json:"quantity"`
	AverageCost    float64 `json:"average_cost"`
	Multiplier     float64 `json:"multiplier"`
}

type Warning struct {
	AccountID   string `json:"account_id"`
	Code        string `json:"code"`
	ExecutionID string `json:"execution_id"`
	Message     string `json:"message"`
}

type AccountSnapshot struct {
	AccountID   string     `json:"account_id"`
	Positions   []Position `json:"positions"`
	CashDelta   float64    `json:"cash_delta"`
	RealizedPnL float64    `json:"realized_pnl"`
	Fees        float64    `json:"fees"`
}

type Snapshot struct {
	Date     string            `json:"date"`
	Accounts []AccountSnapshot `json:"accounts"`
	Warnings []Warning         `json:"warnings"`
}

type positionKey struct {
	accountID, symbol, instrument, lot string
	kind                               Kind
}

type state struct {
	positions map[positionKey]Position
	accounts  map[string]*AccountSnapshot
	warnings  []Warning
}

func (s *state) account(accountID string) *AccountSnapshot {
	account := s.accounts[accountID]
	if account == nil {
		account = &AccountSnapshot{AccountID: accountID}
		s.accounts[accountID] = account
	}
	return account
}

// Replay applies every execution up to each requested Tokyo market date. Dates
// select output only: executions before the first date still establish opening state.
func Replay(executions []store.Execution, dates []time.Time) []Snapshot {
	executions = append([]store.Execution(nil), executions...)
	sort.SliceStable(executions, func(i, j int) bool {
		if executions[i].ExecutedAt.Equal(executions[j].ExecutedAt) {
			return executions[i].ID < executions[j].ID
		}
		return executions[i].ExecutedAt.Before(executions[j].ExecutedAt)
	})
	dates = uniqueDates(dates)

	s := state{positions: map[positionKey]Position{}, accounts: map[string]*AccountSnapshot{}}
	out := make([]Snapshot, 0, len(dates))
	nextExecution := 0
	for _, date := range dates {
		end := tokyoDate(date).AddDate(0, 0, 1)
		for nextExecution < len(executions) && executions[nextExecution].ExecutedAt.Before(end) {
			s.apply(executions[nextExecution])
			nextExecution++
		}
		out = append(out, s.snapshot(date))
	}
	return out
}

func (s *state) apply(ex store.Execution) {
	lot := lotFromDetails(ex)
	kind := kindFromLot(lot)
	openingSide := "buy"
	if kind == MarginShort {
		openingSide = "sell"
	}
	multiplier := ex.Multiplier
	if multiplier == 0 {
		multiplier = 1
	}
	if (ex.Side != "buy" && ex.Side != "sell") || ex.Quantity <= 0 || ex.Price < 0 || multiplier <= 0 {
		s.warn(ex, "invalid_execution", "side, quantity, price, or multiplier is invalid")
		return
	}

	key := positionKey{ex.AccountID, ex.Symbol, ex.InstrumentType, lot, kind}
	position, open := s.positions[key]
	if ex.Side == openingSide {
		if open && abs(position.Multiplier-multiplier) > epsilon {
			s.warn(ex, "invalid_execution_sequence", "position multiplier changed while open")
			return
		}
		if !open {
			position = Position{
				AccountID: ex.AccountID, Symbol: ex.Symbol, InstrumentType: ex.InstrumentType,
				Kind: kind, Lot: lot, Multiplier: multiplier,
			}
		}
		position.AverageCost = (position.AverageCost*position.Quantity + ex.Price*ex.Quantity) / (position.Quantity + ex.Quantity)
		position.Quantity += ex.Quantity
		s.positions[key] = position
		s.bookOpen(ex, kind, multiplier)
		return
	}

	if !open || ex.Quantity-position.Quantity > epsilon {
		s.warn(ex, "invalid_execution_sequence", "close quantity exceeds the open position")
		return
	}
	if abs(position.Multiplier-multiplier) > epsilon {
		s.warn(ex, "invalid_execution_sequence", "position multiplier changed while open")
		return
	}
	gross := (ex.Price - position.AverageCost) * ex.Quantity * multiplier
	if kind == MarginShort {
		gross = -gross
	}
	fees := ex.Fees + ex.Commission
	account := s.account(ex.AccountID)
	account.Fees += fees
	account.RealizedPnL += gross - fees
	if kind == CashLong {
		account.CashDelta += ex.Price*ex.Quantity*multiplier - fees
	} else {
		account.CashDelta += gross - fees
	}
	position.Quantity -= ex.Quantity
	if position.Quantity < epsilon {
		delete(s.positions, key)
	} else {
		s.positions[key] = position
	}
}

func (s *state) bookOpen(ex store.Execution, kind Kind, multiplier float64) {
	fees := ex.Fees + ex.Commission
	account := s.account(ex.AccountID)
	account.Fees += fees
	account.RealizedPnL -= fees
	account.CashDelta -= fees
	if kind == CashLong {
		account.CashDelta -= ex.Price * ex.Quantity * multiplier
	}
}

func (s *state) warn(ex store.Execution, code, message string) {
	s.account(ex.AccountID)
	s.warnings = append(s.warnings, Warning{AccountID: ex.AccountID, Code: code, ExecutionID: ex.ID, Message: message})
}

func (s *state) snapshot(date time.Time) Snapshot {
	accounts := make(map[string]AccountSnapshot, len(s.accounts))
	for accountID, totals := range s.accounts {
		accounts[accountID] = AccountSnapshot{
			AccountID: accountID, CashDelta: money.Round2(totals.CashDelta),
			RealizedPnL: money.Round2(totals.RealizedPnL), Fees: money.Round2(totals.Fees),
		}
	}
	for _, position := range s.positions {
		position.AverageCost = money.Round2(position.AverageCost)
		position.Quantity = money.Round2(position.Quantity)
		account := accounts[position.AccountID]
		account.Positions = append(account.Positions, position)
		accounts[position.AccountID] = account
	}
	out := make([]AccountSnapshot, 0, len(accounts))
	for _, account := range accounts {
		sort.Slice(account.Positions, func(i, j int) bool {
			a, b := account.Positions[i], account.Positions[j]
			if a.Symbol != b.Symbol {
				return a.Symbol < b.Symbol
			}
			if a.InstrumentType != b.InstrumentType {
				return a.InstrumentType < b.InstrumentType
			}
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			return a.Lot < b.Lot
		})
		out = append(out, account)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountID < out[j].AccountID })
	return Snapshot{
		Date: tokyoDate(date).Format(time.DateOnly), Accounts: out,
		Warnings: append([]Warning(nil), s.warnings...),
	}
}

func uniqueDates(dates []time.Time) []time.Time {
	byDate := map[string]time.Time{}
	for _, date := range dates {
		day := tokyoDate(date)
		byDate[day.Format(time.DateOnly)] = day
	}
	out := make([]time.Time, 0, len(byDate))
	for _, date := range byDate {
		out = append(out, date)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func tokyoDate(t time.Time) time.Time {
	t = t.In(tokyo)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, tokyo)
}

func lotFromDetails(ex store.Execution) string {
	if !ex.Details.Valid {
		return ""
	}
	var details struct {
		Lot string `json:"lot"`
	}
	_ = json.Unmarshal([]byte(ex.Details.String), &details)
	return details.Lot
}

func kindFromLot(lot string) Kind {
	switch lot {
	case "sbi:margin-long":
		return MarginLong
	case "sbi:margin-short":
		return MarginShort
	default:
		return CashLong
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
