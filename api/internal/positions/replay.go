package positions

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
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
	Instrument  string `json:"instrument"`
	ExecutionID string `json:"execution_id"`
	Date        string `json:"date"`
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

type Split struct {
	Symbol, InstrumentType string
	EffectiveDate          time.Time
	Ratio                  float64
}

type positionKey struct {
	accountID, symbol, instrument, lot string
	kind                               Kind
}

type state struct {
	positions      map[positionKey]Position
	basisDay       map[positionKey]string
	marginOpenings map[positionKey]int
	marginOpenFees map[positionKey]float64
	accounts       map[string]*AccountSnapshot
	warnings       []Warning
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
	return ReplayWithSplits(executions, dates, nil)
}

func ReplayWithSplits(executions []store.Execution, dates []time.Time, splits []Split) []Snapshot {
	executions = append([]store.Execution(nil), executions...)
	sort.SliceStable(executions, func(i, j int) bool {
		iTime, jTime := replayTime(executions[i]), replayTime(executions[j])
		if iTime.Equal(jTime) {
			return executions[i].ID < executions[j].ID
		}
		return iTime.Before(jTime)
	})
	dates = uniqueDates(dates)
	splits = append([]Split(nil), splits...)
	sort.SliceStable(splits, func(i, j int) bool { return splits[i].EffectiveDate.Before(splits[j].EffectiveDate) })

	s := state{
		positions: map[positionKey]Position{}, basisDay: map[positionKey]string{},
		marginOpenings: map[positionKey]int{}, marginOpenFees: map[positionKey]float64{},
		accounts: map[string]*AccountSnapshot{},
	}
	out := make([]Snapshot, 0, len(dates))
	nextExecution := 0
	nextSplit := 0
	for _, date := range dates {
		end := tokyoDate(date).AddDate(0, 0, 1)
		for nextExecution < len(executions) && executions[nextExecution].ExecutedAt.Before(end) {
			for nextSplit < len(splits) && !tokyoDate(splits[nextSplit].EffectiveDate).After(replayTime(executions[nextExecution])) {
				s.applySplit(splits[nextSplit])
				nextSplit++
			}
			s.apply(executions[nextExecution])
			nextExecution++
		}
		for nextSplit < len(splits) && tokyoDate(splits[nextSplit].EffectiveDate).Before(end) {
			s.applySplit(splits[nextSplit])
			nextSplit++
		}
		out = append(out, s.snapshot(date))
	}
	return out
}

func (s *state) applySplit(split Split) {
	if split.Ratio <= 0 || split.Ratio == 1 {
		return
	}
	for key, position := range s.positions {
		if key.symbol == split.Symbol && key.instrument == split.InstrumentType {
			position.Quantity *= split.Ratio
			position.AverageCost /= split.Ratio
			s.positions[key] = position
		}
	}
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
	sbiCash := lot == "sbi:cash"
	if sbiCash && open && s.basisDay[key] != tokyoDate(ex.ExecutedAt).Format(time.DateOnly) {
		position.AverageCost = math.Ceil(position.AverageCost - epsilon)
	}
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
		buyCost := ex.Price * ex.Quantity
		if sbiCash {
			buyCost += ex.Fees + ex.Commission
			s.basisDay[key] = tokyoDate(ex.ExecutedAt).Format(time.DateOnly)
		}
		position.AverageCost = (position.AverageCost*position.Quantity + buyCost) / (position.Quantity + ex.Quantity)
		position.Quantity += ex.Quantity
		s.positions[key] = position
		if kind != CashLong && strings.HasPrefix(lot, "sbi:margin-") {
			s.marginOpenings[key]++
			s.marginOpenFees[key] += ex.Fees + ex.Commission
		}
		s.bookOpen(ex, kind, multiplier, sbiCash)
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
	if sbiCash {
		position.AverageCost = math.Ceil(position.AverageCost - epsilon)
	}
	gross := (ex.Price - position.AverageCost) * ex.Quantity * multiplier
	if kind == MarginShort {
		gross = -gross
	}
	fees := ex.Fees + ex.Commission
	account := s.account(ex.AccountID)
	account.Fees += fees
	settlement := gross - fees
	cashSettlement := settlement
	if kind != CashLong && strings.HasPrefix(lot, "sbi:margin-") {
		openFees := s.marginOpenFees[key] * ex.Quantity / position.Quantity
		s.marginOpenFees[key] -= openFees
		if reported := reportedPnlFromDetails(ex); reported != nil {
			settlement = *reported
			cashSettlement = settlement + openFees // opening fee was paid earlier
		} else if s.marginOpenings[key] > 1 {
			s.warn(ex, "margin_realized_pnl_unavailable", "multiple margin openings have no broker-reported close result or lot match")
			settlement = 0
			cashSettlement = -fees
		} else {
			settlement -= openFees
		}
	}
	account.RealizedPnL += settlement
	if kind == CashLong {
		account.CashDelta += ex.Price*ex.Quantity*multiplier - fees
	} else {
		account.CashDelta += cashSettlement
	}
	position.Quantity -= ex.Quantity
	if position.Quantity < epsilon {
		delete(s.positions, key)
		delete(s.basisDay, key)
		delete(s.marginOpenings, key)
		delete(s.marginOpenFees, key)
	} else {
		s.positions[key] = position
	}
}

func reportedPnlFromDetails(ex store.Execution) *float64 {
	if !ex.Details.Valid {
		return nil
	}
	var details struct {
		Pnl *float64 `json:"broker_reported_realized_pnl"`
	}
	if json.Unmarshal([]byte(ex.Details.String), &details) != nil {
		return nil
	}
	return details.Pnl
}

func (s *state) bookOpen(ex store.Execution, kind Kind, multiplier float64, sbiCash bool) {
	fees := ex.Fees + ex.Commission
	account := s.account(ex.AccountID)
	account.Fees += fees
	if !sbiCash && !(kind != CashLong && strings.HasPrefix(lotFromDetails(ex), "sbi:margin-")) {
		account.RealizedPnL -= fees
	}
	account.CashDelta -= fees
	if kind == CashLong {
		account.CashDelta -= ex.Price * ex.Quantity * multiplier
	}
}

func (s *state) warn(ex store.Execution, code, message string) {
	s.account(ex.AccountID)
	s.warnings = append(s.warnings, Warning{
		AccountID: ex.AccountID, Code: code, Instrument: ex.Symbol, ExecutionID: ex.ID,
		Date: tokyoDate(ex.ExecutedAt).Format(time.DateOnly), Message: message,
	})
}

func replayTime(ex store.Execution) time.Time {
	if !strings.HasPrefix(lotFromDetails(ex), "sbi:") {
		return ex.ExecutedAt
	}
	day := tokyoDate(ex.ExecutedAt)
	if !opensPosition(ex) {
		day = day.Add(12 * time.Hour)
	}
	return day
}

func opensPosition(ex store.Execution) bool {
	if kindFromLot(lotFromDetails(ex)) == MarginShort {
		return ex.Side == "sell"
	}
	return ex.Side == "buy"
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
		if position.Lot == "sbi:cash" {
			position.AverageCost = math.Ceil(position.AverageCost - epsilon)
		} else {
			position.AverageCost = money.Round2(position.AverageCost)
		}
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
