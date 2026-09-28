package api

import (
	"context"
	"math"
	"net/http"
	"strings"

	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/marketdata"
)

type summaryResponse struct {
	analytics.Summary
	Currency       string                      `json:"currency"`
	TargetCurrency string                      `json:"target_currency"`
	FXPolicy       string                      `json:"fx_policy"`
	FXRates        []marketdata.FxRateResponse `json:"fx_rates"`
}

func (s *Server) summary(ctx context.Context, uid string, f Filters, requested string) (summaryResponse, error) {
	var out summaryResponse
	target := strings.ToUpper(strings.TrimSpace(requested))
	if target != "" && !validCurrency(target) {
		return out, Fail(http.StatusBadRequest, "bad_request", "target_currency must be a three-letter currency code", nil)
	}
	currencies, err := s.portfolioCurrencies(ctx, uid, f)
	if err != nil {
		return out, failLoad(err, "could not resolve summary currency")
	}
	if target == "" {
		if len(currencies) > 1 {
			return out, Fail(http.StatusBadRequest, "mixed_currencies", "target_currency is required for mixed-currency summary", nil)
		}
		if len(currencies) == 1 {
			target = currencies[0]
		}
	}
	target = strings.ToUpper(strings.TrimSpace(target))
	if target != "" && !validCurrency(target) {
		return out, failLoad(errUnknownCurrency, "could not resolve summary currency")
	}
	rows, err := s.loadClosedTradeRows(ctx, uid, f)
	if err != nil {
		return out, failLoad(err, "could not compute summary")
	}
	out.Currency, out.TargetCurrency = target, strings.ToUpper(strings.TrimSpace(requested))
	out.FXPolicy = "latest"
	out.FXRates = []marketdata.FxRateResponse{}
	rates := map[string]float64{target: 1}
	trades := make([]analytics.ClosedTrade, 0, len(rows))
	for _, row := range rows {
		if !row.NetPnl.Valid || !row.ClosedAt.Valid {
			continue
		}
		source := strings.ToUpper(strings.TrimSpace(row.PnlCurrency))
		if !validCurrency(source) {
			return out, failLoad(errUnknownCurrency, "trade P&L currency could not be resolved")
		}
		rate, ok := rates[source]
		if !ok {
			if s.deps.Market == nil {
				return out, Fail(http.StatusBadGateway, "fx_unavailable", "summary FX is unavailable", nil)
			}
			fx, fxErr := s.deps.Market.GetFxRate(ctx, source, target)
			if fxErr != nil || fx.Rate <= 0 || math.IsNaN(fx.Rate) || math.IsInf(fx.Rate, 0) {
				return out, Fail(http.StatusBadGateway, "fx_unavailable", "summary FX is unavailable", nil)
			}
			rate = fx.Rate
			rates[source] = rate
			out.FXRates = append(out.FXRates, fx)
		}
		net, gross, fees := row.NetPnl.Float64*rate, grossPnlOf(row)*rate, row.FeesTotal*rate
		for _, value := range []float64{net, gross, fees} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return out, Fail(http.StatusBadGateway, "fx_unavailable", "invalid normalized monetary input", nil)
			}
		}
		trades = append(trades, analytics.ClosedTrade{NetPnl: net, GrossPnl: gross, FeesTotal: fees, OpenedAt: row.OpenedAt, ClosedAt: row.ClosedAt.Time})
	}
	out.Summary = analytics.Summarize(trades)
	return out, nil
}

func validCurrency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}
