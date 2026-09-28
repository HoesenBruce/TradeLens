package positions

import (
	"encoding/json"
	"math"
	"time"

	"github.com/tradermemos/api/internal/money"
	"github.com/tradermemos/api/internal/store"
)

// Settlement is recorded on both legs; only the cash disposal realizes P&L.
type Settlement struct {
	CashQtyDisposed      float64 `json:"cash_qty_disposed"`
	CashCostBasisUsed    float64 `json:"cash_cost_basis_used"`
	MarginShortQtyClosed float64 `json:"margin_short_qty_closed"`
	DisposalPrice        float64 `json:"disposal_price"`
	ApplicableCosts      float64 `json:"applicable_costs"`
	RealizedPnL          float64 `json:"realized_pnl"`
	Source               string  `json:"realized_pnl_source"`
}

type settlementMetadata struct {
	Type     string   `json:"settlement_type"`
	ID       string   `json:"settlement_id"`
	Proceeds *float64 `json:"broker_reported_settlement_proceeds"`
	Pnl      *float64 `json:"broker_reported_realized_pnl"`
	Basis    *float64 `json:"broker_reported_close_basis"`
}

func settlementDetails(ex store.Execution) settlementMetadata {
	var d settlementMetadata
	if ex.Details.Valid {
		_ = json.Unmarshal([]byte(ex.Details.String), &d)
	}
	return d
}

func settlementKey(ex store.Execution) string {
	return ex.UserID + "|" + ex.AccountID + "|" + settlementDetails(ex).ID
}

func settlementPairs(executions []store.Execution) map[string][]store.Execution {
	pairs := map[string][]store.Execution{}
	for _, ex := range executions {
		d := settlementDetails(ex)
		if d.Type == "genwatashi" && d.ID != "" {
			pairs[settlementKey(ex)] = append(pairs[settlementKey(ex)], ex)
		}
	}
	return pairs
}

// SettlementResults runs the same replay used by historical account valuation.
// Missing results signal incomplete history, never a guessed short lot.
func SettlementResults(executions []store.Execution) map[string]Settlement {
	results := map[string]Settlement{}
	var last time.Time
	for _, ex := range executions {
		if ex.ExecutedAt.After(last) {
			last = ex.ExecutedAt
		}
	}
	replay(executions, []time.Time{last}, nil, nil, results)
	return results
}

func (s *state) applyGenwatashi(ex store.Execution) {
	// The first (cash) leg settles both positions atomically.
	if lotFromDetails(ex) == "sbi:margin-short" {
		if len(s.settlementLegs[settlementKey(ex)]) != 2 {
			s.warn(ex, "invalid_genwatashi_settlement", "cash settlement leg is missing")
		}
		return
	}
	legs := s.settlementLegs[settlementKey(ex)]
	invalid := func() {
		s.warn(ex, "invalid_genwatashi_settlement", "missing or ambiguous cash/short history or settlement pair")
	}
	if len(legs) != 2 || lotFromDetails(ex) != "sbi:cash" || ex.Side != "sell" || ex.Quantity <= 0 || ex.Price < 0 || ex.Multiplier != 1 || ex.InstrumentType != "stock" {
		invalid()
		return
	}
	var short store.Execution
	for _, leg := range legs {
		if leg.ID != ex.ID {
			short = leg
		}
	}
	if lotFromDetails(short) != "sbi:margin-short" || short.Side != "buy" || short.Symbol != ex.Symbol || short.InstrumentType != ex.InstrumentType || short.Quantity != ex.Quantity || short.Multiplier != ex.Multiplier || short.Price != ex.Price || short.Fees != 0 || short.Commission != 0 || !tokyoDate(short.ExecutedAt).Equal(tokyoDate(ex.ExecutedAt)) {
		invalid()
		return
	}
	cashKey := positionKey{ex.AccountID, ex.Symbol, ex.InstrumentType, "sbi:cash", CashLong}
	shortKey := positionKey{ex.AccountID, ex.Symbol, ex.InstrumentType, "sbi:margin-short", MarginShort}
	cash, margin := s.positions[cashKey], s.positions[shortKey]
	if cash.Quantity+epsilon < ex.Quantity || margin.Quantity+epsilon < ex.Quantity || cash.Multiplier != 1 || margin.Multiplier != 1 {
		invalid()
		return
	}
	d := settlementDetails(ex)
	// A history export's price alone cannot select one of several short entries.
	if d.Proceeds == nil && abs(ex.Price-margin.AverageCost) > epsilon {
		invalid()
		return
	}
	basis := cash.AverageCost
	if d.Basis != nil {
		basis = *d.Basis
	} else {
		basis = math.Ceil(basis - epsilon)
	}
	costs := ex.Fees + ex.Commission
	proceeds := ex.Price*ex.Quantity - costs
	if d.Proceeds != nil {
		proceeds = *d.Proceeds
		costs = ex.Price*ex.Quantity - proceeds
	}
	if costs < 0 || basis < 0 || margin.AverageCost*margin.Quantity+epsilon < ex.Price*ex.Quantity {
		invalid()
		return
	}
	openFees := s.marginOpenFees[shortKey] * ex.Quantity / margin.Quantity
	if costs+epsilon < openFees {
		invalid()
		return
	}
	pnl, source := money.Round2(proceeds-basis*ex.Quantity), "calculated_sbi_cash_basis"
	if d.Pnl != nil {
		pnl, source = *d.Pnl, "broker_reported"
	}
	result := Settlement{ex.Quantity, money.Round2(basis * ex.Quantity), ex.Quantity, ex.Price, money.Round2(costs), pnl, source}
	if s.settlements != nil {
		s.settlements[ex.ID], s.settlements[short.ID] = result, result
	}
	// SBI settles opening fees again inside the aggregate receipt. Reverse their
	// earlier cash booking so the aggregate costs are counted exactly once.
	account := s.account(ex.AccountID)
	account.RealizedPnL += pnl
	account.CashDelta += proceeds + openFees
	account.Fees += costs - openFees
	s.marginOpenFees[shortKey] -= openFees
	cash.Quantity -= ex.Quantity
	remainingShortCost := margin.AverageCost*margin.Quantity - ex.Price*ex.Quantity
	margin.Quantity -= ex.Quantity
	if margin.Quantity > epsilon {
		margin.AverageCost = remainingShortCost / margin.Quantity
	}
	for key, p := range map[positionKey]Position{cashKey: cash, shortKey: margin} {
		if p.Quantity < epsilon {
			delete(s.positions, key)
			delete(s.marginOpenFees, key)
			delete(s.basisDay, key)
			delete(s.convertedCash, key)
		} else {
			s.positions[key] = p
		}
	}
}
