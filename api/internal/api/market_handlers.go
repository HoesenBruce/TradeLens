package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

func (s *Server) marketRoutes(g *echo.Group) {
	g.GET("/market/bars", s.handleMarketBars)
	g.GET("/market/corporate-actions", s.handleMarketCorporateActions)
	g.GET("/market/fx", s.handleMarketFx)
}

func (s *Server) handleMarketCorporateActions(c *echo.Context) error {
	symbol := strings.TrimSpace(c.QueryParam("symbol"))
	if symbol == "" {
		return Fail(http.StatusBadRequest, "bad_request", "symbol is required", nil)
	}
	from, err := marketdata.ParseTimeParam(strings.TrimSpace(c.QueryParam("from")))
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid from: "+err.Error(), nil)
	}
	to, err := marketdata.ParseTimeParam(strings.TrimSpace(c.QueryParam("to")))
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid to: "+err.Error(), nil)
	}
	instrumentType := strings.TrimSpace(c.QueryParam("instrument_type"))
	if instrumentType == "" {
		instrumentType = "stock"
	}
	if !marketdata.SupportedInstrument(instrumentType) {
		return Fail(http.StatusBadRequest, "bad_request", "unsupported instrument_type", nil)
	}
	if s.deps.Market == nil {
		return Fail(http.StatusServiceUnavailable, "unavailable", "market data not configured", nil)
	}
	if !marketdata.ChartableSymbol(symbol) {
		return c.JSON(http.StatusOK, []marketdata.CorporateActionCandidate{})
	}

	out, err := s.deps.Market.DetectCorporateActions(c.Request().Context(), marketdata.Request{
		Symbol: symbol, InstrumentType: instrumentType, From: from, To: to,
	})
	if err != nil {
		c.Logger().Warn("corporate action detection failed", "symbol", symbol, "err", err)
		return Fail(http.StatusBadGateway, "upstream_error", "failed to detect corporate actions", nil)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) handleMarketFx(c *echo.Context) error {
	from := strings.TrimSpace(c.QueryParam("from"))
	to := strings.TrimSpace(c.QueryParam("to"))
	if from == "" || to == "" {
		return Fail(http.StatusBadRequest, "bad_request", "from and to are required", nil)
	}
	if s.deps.Market == nil {
		return Fail(http.StatusServiceUnavailable, "unavailable", "market data not configured", nil)
	}
	out, err := s.deps.Market.GetFxRate(c.Request().Context(), from, to)
	if err != nil {
		c.Logger().Warn("fx rate fetch failed", "from", from, "to", to, "err", err)
		return Fail(http.StatusBadGateway, "upstream_error", "failed to fetch FX rate", nil)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) handleMarketBars(c *echo.Context) error {
	symbol := strings.TrimSpace(c.QueryParam("symbol"))
	if symbol == "" {
		return Fail(http.StatusBadRequest, "bad_request", "symbol is required", nil)
	}
	instrumentType := strings.TrimSpace(c.QueryParam("instrument_type"))
	if instrumentType == "" {
		instrumentType = "stock"
	}
	if !marketdata.SupportedInstrument(instrumentType) {
		return Fail(http.StatusBadRequest, "bad_request", "unsupported instrument_type", nil)
	}

	interval, err := marketdata.ParseInterval(c.QueryParam("interval"))
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}

	fromRaw := strings.TrimSpace(c.QueryParam("from"))
	toRaw := strings.TrimSpace(c.QueryParam("to"))
	if fromRaw == "" || toRaw == "" {
		return Fail(http.StatusBadRequest, "bad_request", "from and to are required (RFC3339)", nil)
	}
	from, err := marketdata.ParseTimeParam(fromRaw)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid from: "+err.Error(), nil)
	}
	to, err := marketdata.ParseTimeParam(toRaw)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid to: "+err.Error(), nil)
	}

	if s.deps.Market == nil {
		return Fail(http.StatusServiceUnavailable, "unavailable", "market data not configured", nil)
	}

	if !marketdata.ChartableSymbol(symbol) {
		req := marketdata.Request{
			Symbol: symbol, InstrumentType: instrumentType, Interval: interval,
			From: from, To: to,
		}
		return c.JSON(http.StatusOK, marketdata.EmptyResponse(req, "skipped"))
	}

	if interval == "" {
		interval = marketdata.DefaultInterval(from, to)
	}

	// Stable window: snap to minute boundaries before padding.
	from = marketdata.SnapChartTime(from)
	to = marketdata.SnapChartTime(to)

	// Pad chart window so entries/exits near edges have context.
	pad := chartPadding(interval)
	from = from.Add(-pad)
	to = to.Add(pad)
	now := marketdata.SnapChartTime(time.Now().UTC())
	if to.After(now) {
		to = now
	}

	req := marketdata.Request{
		Symbol:         symbol,
		InstrumentType: instrumentType,
		Interval:       interval,
		From:           from,
		To:             to,
	}
	var out marketdata.Response
	if interval == "D" {
		earliest, rangeErr := s.earliestExecution(c.Request().Context(), auth.UserID(c), symbol, instrumentType)
		if rangeErr != nil {
			c.Logger().Warn("execution range lookup failed", "symbol", symbol, "err", rangeErr)
			out, err = s.deps.Market.GetBars(c.Request().Context(), req)
		} else {
			out, err = s.deps.Market.GetTransactionBars(c.Request().Context(), req, earliest)
		}
	} else {
		out, err = s.deps.Market.GetBars(c.Request().Context(), req)
	}
	if err != nil {
		// Chart widget should degrade gracefully — return empty bars, not 502.
		c.Logger().Warn("market bars fetch failed", "symbol", symbol, "err", err)
		return c.JSON(http.StatusOK, marketdata.EmptyResponse(req, "unavailable"))
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) earliestExecution(ctx context.Context, userID, symbol, instrumentType string) (time.Time, error) {
	accounts, err := s.deps.Store.ListAccounts(ctx, userID)
	if err != nil {
		return time.Time{}, err
	}
	var earliest time.Time
	for _, account := range accounts {
		executions, err := s.deps.Store.ListExecutionsForAccount(ctx, store.ListExecutionsForAccountParams{UserID: userID, AccountID: account.ID})
		if err != nil {
			return time.Time{}, err
		}
		for _, execution := range executions {
			if strings.EqualFold(strings.TrimSpace(execution.Symbol), strings.TrimSpace(symbol)) && execution.InstrumentType == instrumentType && (earliest.IsZero() || execution.ExecutedAt.Before(earliest)) {
				earliest = execution.ExecutedAt
			}
		}
	}
	return earliest, nil
}

func chartPadding(interval string) time.Duration {
	switch interval {
	case "1":
		return 30 * time.Minute
	case "5":
		return time.Hour
	case "15":
		return 2 * time.Hour
	case "30":
		return 4 * time.Hour
	case "60", "240":
		return 6 * time.Hour
	case "W":
		return 14 * 24 * time.Hour
	case "M":
		return 60 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}
