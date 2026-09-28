package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/auth"
)

// handleBehavior runs the behavioral-pattern detectors over closed trades.
func (s *Server) handleBehavior(c *echo.Context) error {
	ctx := c.Request().Context()
	uid := auth.UserID(c)
	f, err := parseFilters(c)
	if err != nil {
		return Fail(http.StatusBadRequest, "bad_request", err.Error(), nil)
	}

	meta, rows, err := s.normalizedClosedRows(ctx, uid, f, c.QueryParam("target_currency"))
	if err != nil {
		return err
	}
	journals, err := s.deps.Store.ListTradeJournalsForUser(ctx, uid)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load journals", nil)
	}
	mfeByTrade := make(map[string]float64, len(journals))
	for _, j := range journals {
		if j.Mfe.Valid {
			mfeByTrade[j.TradeID] = j.Mfe.Float64
		}
	}

	rates := map[string]float64{meta.Currency: 1}
	for _, fx := range meta.FXRates {
		rates[fx.From] = fx.Rate
	}
	trades := make([]analytics.BehaviorTrade, 0, len(rows))
	for _, t := range rows {
		if !t.NetPnl.Valid || !t.ClosedAt.Valid {
			continue
		}
		bt := analytics.BehaviorTrade{
			ID:            t.ID,
			Symbol:        t.Symbol,
			QtyOpened:     t.QtyOpened,
			AvgEntryPrice: t.AvgEntryPrice * rates[strings.ToUpper(strings.TrimSpace(t.PnlCurrency))],
			NetPnl:        t.NetPnl.Float64,
			OpenedAt:      t.OpenedAt,
			ClosedAt:      t.ClosedAt.Time,
		}
		if t.TimeInTradeSecs.Valid {
			bt.TimeInTradeSecs = t.TimeInTradeSecs.Int64
		}
		if mfe, ok := mfeByTrade[t.ID]; ok {
			v := mfe * rates[strings.ToUpper(strings.TrimSpace(t.PnlCurrency))]
			bt.Mfe = &v
		}
		trades = append(trades, bt)
	}
	return c.JSON(http.StatusOK, struct {
		analytics.BehaviorReport
		currencyMetadata
	}{analytics.Behavior(trades, analytics.DefaultBehaviorConfig(), f.Loc), meta})
}
