package api

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/accountvalue"
	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/store"
)

func (s *Server) analyticsRoutes(g *echo.Group) {
	g.GET("/analytics/summary", s.handleSummary)
	g.GET("/analytics/r-summary", s.handleRSummary)
	g.GET("/analytics/equity-curve", s.handleEquityCurve)
	g.GET("/analytics/account-value", s.handleAccountValue)
	g.GET("/analytics/daily", s.handleDaily)
	g.GET("/analytics/breakdown", s.handleBreakdown)
	g.GET("/analytics/compliance", s.handleCompliance)
	g.GET("/analytics/behavior", s.handleBehavior)
	g.GET("/analytics/montecarlo", s.handleMonteCarlo)
	g.GET("/analytics/execution-score", s.handleExecScore)
}

type accountValueResponse struct {
	Currency         string               `json:"currency"`
	Timezone         string               `json:"timezone"`
	AdjustmentStatus string               `json:"adjustment_status"`
	Points           []accountvalue.Point `json:"points"`
}

func (s *Server) handleAccountValue(c *echo.Context) error {
	if s.deps.AccountValue == nil {
		return Fail(http.StatusServiceUnavailable, "unavailable", "market data not configured", nil)
	}
	ctx, uid := c.Request().Context(), auth.UserID(c)
	from, err := dateParam(c.QueryParam("from"))
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid 'from' date (want YYYY-MM-DD)", nil)
	}
	to, err := dateParam(c.QueryParam("to"))
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid 'to' date (want YYYY-MM-DD)", nil)
	}
	if from != nil && to != nil && from.After(*to) {
		return Fail(http.StatusBadRequest, "bad_request", "from must not be after to", nil)
	}

	accounts, currency, err := s.accountValueAccounts(ctx, uid, parseAccountIDs(c))
	if err != nil {
		return err
	}
	response := accountValueResponse{Currency: currency, Timezone: "Asia/Tokyo", AdjustmentStatus: "unadjusted", Points: []accountvalue.Point{}}
	if len(accounts) == 0 {
		return c.JSON(http.StatusOK, response)
	}

	executions := make([]store.Execution, 0)
	selected := make(map[string]bool, len(accounts))
	for _, account := range accounts {
		selected[account.ID] = true
		rows, loadErr := s.deps.Store.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: uid, AccountID: account.ID})
		if loadErr != nil {
			return Fail(http.StatusInternalServerError, "internal", "could not load executions", nil)
		}
		executions = append(executions, rows...)
	}
	cashRows, err := s.deps.Store.ListCashTransactions(ctx, store.ListCashTransactionsParams{UserID: uid})
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load cash flows", nil)
	}
	cash := cashRows[:0]
	for _, row := range cashRows {
		if selected[row.AccountID] {
			cash = append(cash, row)
		}
	}
	if len(executions) == 0 && len(cash) == 0 {
		return c.JSON(http.StatusOK, response)
	}

	from, to = reconstructionRange(from, to, executions, cash)
	if from.After(*to) {
		return c.JSON(http.StatusOK, response)
	}
	sessions, err := s.deps.AccountValue.MarketSessions(ctx, *from, *to)
	if err != nil {
		return Fail(http.StatusBadGateway, "upstream_error", "could not load market sessions", nil).Wrap(err)
	}
	result, err := s.deps.AccountValue.Reconstruct(ctx, accountvalue.Request{
		Executions: executions, CashTransactions: cash, MarketSessions: sessions,
		ConfirmedSuspensions: ignoredMissingPrices(c.QueryParam("ignored_missing_prices")),
	})
	if err != nil {
		return Fail(http.StatusBadGateway, "upstream_error", "could not reconstruct account value", nil).Wrap(err)
	}
	response.Points = accountvalue.Combine(result)
	return c.JSON(http.StatusOK, response)
}

func ignoredMissingPrices(raw string) map[accountvalue.Instrument]map[string]bool {
	out := map[accountvalue.Instrument]map[string]bool{}
	for _, item := range strings.Split(raw, ",") {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			continue
		}
		instrument := accountvalue.Instrument{Symbol: strings.TrimSpace(parts[0]), InstrumentType: "stock"}
		date := strings.TrimSpace(parts[1])
		if _, err := time.Parse(time.DateOnly, date); instrument.Symbol != "" && err == nil {
			if out[instrument] == nil {
				out[instrument] = map[string]bool{}
			}
			out[instrument][date] = true
		}
	}
	return out
}

func (s *Server) accountValueAccounts(ctx context.Context, userID string, requested []string) ([]store.Account, string, error) {
	rows, err := s.deps.Store.ListAccounts(ctx, userID)
	if err != nil {
		return nil, "", Fail(http.StatusInternalServerError, "internal", "could not load accounts", nil)
	}
	wanted := map[string]bool{}
	for _, accountID := range requested {
		wanted[accountID] = true
	}
	selected, currency := make([]store.Account, 0), ""
	for _, account := range rows {
		if len(wanted) > 0 && !wanted[account.ID] {
			continue
		}
		if len(wanted) == 0 && account.AccountType == AccountTypeBacktest {
			continue
		}
		if currency != "" && account.BaseCurrency != currency {
			return nil, "", Fail(http.StatusBadRequest, "mixed_currencies", errMixedCurrencies.Error(), nil)
		}
		currency = account.BaseCurrency
		selected = append(selected, account)
	}
	return selected, currency, nil
}

func dateParam(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	date, err := time.ParseInLocation(time.DateOnly, value, time.FixedZone("Asia/Tokyo", 9*60*60))
	return &date, err
}

func reconstructionRange(from, to *time.Time, executions []store.Execution, cash []store.CashTransaction) (*time.Time, *time.Time) {
	tokyo := time.FixedZone("Asia/Tokyo", 9*60*60)
	if from == nil {
		var earliest time.Time
		for _, execution := range executions {
			if earliest.IsZero() || execution.ExecutedAt.Before(earliest) {
				earliest = execution.ExecutedAt
			}
		}
		for _, row := range cash {
			if earliest.IsZero() || row.OccurredAt.Before(earliest) {
				earliest = row.OccurredAt
			}
		}
		date := earliest.In(tokyo)
		value := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, tokyo)
		from = &value
	}
	if to == nil {
		now := time.Now().In(tokyo)
		value := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tokyo)
		to = &value
	}
	return from, to
}

// grossPnlOf reads the stored gross P&L, reconstructing it from net + fees
// for rows imported before gross was recorded.
func grossPnlOf(t store.Trade) float64 {
	if t.GrossPnl.Valid {
		return t.GrossPnl.Float64
	}
	return t.NetPnl.Float64 + t.FeesTotal
}

func toClosedTrades(rows []store.Trade) []analytics.ClosedTrade {
	out := make([]analytics.ClosedTrade, 0, len(rows))
	for _, t := range rows {
		if !t.NetPnl.Valid || !t.ClosedAt.Valid {
			continue
		}
		out = append(out, analytics.ClosedTrade{
			NetPnl:    t.NetPnl.Float64,
			GrossPnl:  grossPnlOf(t),
			FeesTotal: t.FeesTotal,
			OpenedAt:  t.OpenedAt,
			ClosedAt:  t.ClosedAt.Time,
		})
	}
	return out
}

func (s *Server) handleSummary(c *echo.Context) error {
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	response, err := s.summary(c.Request().Context(), auth.UserID(c), f, c.QueryParam("target_currency"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response)
}

func (s *Server) handleRSummary(c *echo.Context) error {
	ctx := c.Request().Context()
	uid := auth.UserID(c)
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	rows, err := s.loadClosedTrades(ctx, uid, f)
	if err != nil {
		return failLoad(err, "could not compute r-summary")
	}
	risks, err := s.deps.Store.ListJournalRisks(ctx, uid)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load risk", nil)
	}
	riskByTrade := make(map[string]float64, len(risks))
	for _, r := range risks {
		if r.InitialRisk.Valid {
			riskByTrade[r.TradeID] = r.InitialRisk.Float64
		}
	}
	var withRisk []analytics.RiskTrade
	excluded := 0
	for _, t := range rows {
		if !t.NetPnl.Valid {
			continue
		}
		risk, ok := riskByTrade[t.ID]
		if !ok || risk <= 0 {
			excluded++
			continue
		}
		withRisk = append(withRisk, analytics.RiskTrade{
			NetPnl: t.NetPnl.Float64, InitialRisk: risk, FeesTotal: t.FeesTotal,
		})
	}
	return c.JSON(http.StatusOK, analytics.SummarizeR(withRisk, excluded))
}

// handleMonteCarlo bootstrap-resamples the filtered closed trades' net P&L.
// Optional params: paths, horizon, seed (reproducible runs), ruin_threshold
// (currency drawdown counted as ruin; defaults to the historical max
// drawdown).
func (s *Server) handleMonteCarlo(c *echo.Context) error {
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	params := analytics.MonteCarloParams{}
	if params.Seed, err = uintParam(c, "seed"); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid 'seed' (want unsigned integer)", nil)
	}
	intParams := map[string]*int{"paths": &params.Paths, "horizon": &params.Horizon}
	for name, dst := range intParams {
		if v := c.QueryParam(name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return Fail(http.StatusBadRequest, "bad_request", "invalid '"+name+"' (want positive integer)", nil)
			}
			*dst = n
		}
	}
	if v := c.QueryParam("ruin_threshold"); v != "" {
		t, err := strconv.ParseFloat(v, 64)
		if err != nil || t < 0 {
			return Fail(http.StatusBadRequest, "bad_request", "invalid 'ruin_threshold' (want positive number)", nil)
		}
		params.RuinThreshold = t
	}

	rows, err := s.loadClosedTrades(c.Request().Context(), auth.UserID(c), f)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not run simulation", nil)
	}
	trades := toClosedTrades(rows)
	// Chronological order matters only for the historical-drawdown anchor;
	// the bootstrap itself samples i.i.d.
	sort.Slice(trades, func(i, j int) bool { return trades[i].ClosedAt.Before(trades[j].ClosedAt) })
	pnls := make([]float64, len(trades))
	for i, t := range trades {
		pnls[i] = t.NetPnl
	}
	return c.JSON(http.StatusOK, analytics.MonteCarlo(pnls, params))
}

func uintParam(c *echo.Context, name string) (uint64, error) {
	v := c.QueryParam(name)
	if v == "" {
		return 0, nil
	}
	return strconv.ParseUint(v, 10, 64)
}

func (s *Server) handleDaily(c *echo.Context) error {
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}
	meta, trades, err := s.normalizedTrades(c.Request().Context(), auth.UserID(c), f, c.QueryParam("target_currency"))
	if err != nil {
		return err
	}
	// Preserve the native map contract for existing clients without a target.
	if c.QueryParam("target_currency") == "" {
		return c.JSON(http.StatusOK, analytics.DailyPnl(trades, f.DateBasis, f.Loc))
	}
	return c.JSON(http.StatusOK, struct {
		currencyMetadata
		Pnl map[string]float64 `json:"pnl"`
	}{meta, analytics.DailyPnl(trades, f.DateBasis, f.Loc)})
}

func (s *Server) handleEquityCurve(c *echo.Context) error {
	ctx := c.Request().Context()
	uid := auth.UserID(c)
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}

	meta, trades, err := s.normalizedTrades(ctx, uid, f, c.QueryParam("target_currency"))
	if err != nil {
		return err
	}
	cashRows, err := s.deps.Store.ListCashTransactions(ctx, store.ListCashTransactionsParams{
		UserID: uid, AccountID: f.accountNarg(),
	})
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load cash flows", nil)
	}
	excluded, err := s.backtestAccountIDs(ctx, uid, f)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load cash flows", nil)
	}
	cashRows = filterCashAccounts(cashRows, excluded)
	rates := map[string]float64{meta.Currency: 1}
	for _, fx := range meta.FXRates {
		rates[fx.From] = fx.Rate
	}
	flows := make([]analytics.CashFlow, 0, len(cashRows))
	cash := make([]cashDTO, 0, len(cashRows))
	for _, ct := range cashRows {
		if !f.matchAccount(ct.AccountID) {
			continue
		}
		rate, err := s.analyticsRate(ctx, &meta, rates, ct.Currency)
		if err != nil {
			return err
		}
		amount := ct.Amount * rate
		if math.IsNaN(amount) || math.IsInf(amount, 0) {
			return Fail(http.StatusBadGateway, "fx_unavailable", "invalid normalized cash input", nil)
		}
		flows = append(flows, analytics.CashFlow{Amount: amount, OccurredAt: ct.OccurredAt})
		ct.Amount, ct.Currency = amount, meta.Currency
		cash = append(cash, toCashDTO(ct))
	}
	// The curve starts at zero: accounts.starting_balance is metadata only, it is
	// already seeded into the ledger as the "Opening balance" deposit
	// (ensureOpeningDeposit), so adding it here would count that money twice.
	return c.JSON(http.StatusOK, struct {
		analytics.Equity
		currencyMetadata
		CashTransactions []cashDTO `json:"cash_transactions"`
	}{analytics.EquityCurve(0, flows, trades), meta, cash})
}
