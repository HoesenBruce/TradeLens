package api

import (
	"context"
	"math"
	"net/http"
	"strings"

	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/marketdata"
)

type currencyMetadata struct {
	Currency       string                      `json:"currency"`
	TargetCurrency string                      `json:"target_currency"`
	FXPolicy       string                      `json:"fx_policy"`
	FXRates        []marketdata.FxRateResponse `json:"fx_rates"`
}

type summaryResponse struct {
	analytics.Summary
	currencyMetadata
}

// normalizedTrades is shared by monetary analytics; stored accounting inputs stay native.
func (s *Server) normalizedTrades(ctx context.Context, uid string, f Filters, requested string) (out currencyMetadata, trades []analytics.ClosedTrade, err error) {
	target := strings.ToUpper(strings.TrimSpace(requested))
	if target != "" && !validCurrency(target) {
		return out, nil, Fail(http.StatusBadRequest, "bad_request", "target_currency must be a three-letter currency code", nil)
	}
	currencies, err := s.portfolioCurrencies(ctx, uid, f)
	if err != nil {
		return out, nil, failLoad(err, "could not resolve analytics currency")
	}
	if target == "" {
		if len(currencies) > 1 {
			return out, nil, Fail(http.StatusBadRequest, "mixed_currencies", "target_currency is required for mixed-currency analytics", nil)
		}
		if len(currencies) == 1 {
			target = currencies[0]
		}
	}
	target = strings.ToUpper(strings.TrimSpace(target))
	if target != "" && !validCurrency(target) {
		return out, nil, failLoad(errUnknownCurrency, "could not resolve analytics currency")
	}
	rows, err := s.loadClosedTradeRows(ctx, uid, f)
	if err != nil {
		return out, nil, failLoad(err, "could not compute summary")
	}
	out.Currency, out.TargetCurrency = target, strings.ToUpper(strings.TrimSpace(requested))
	out.FXPolicy = "latest"
	out.FXRates = []marketdata.FxRateResponse{}
	rates := map[string]float64{target: 1}
	trades = make([]analytics.ClosedTrade, 0, len(rows))
	for _, row := range rows {
		if !row.NetPnl.Valid || !row.ClosedAt.Valid {
			continue
		}
		rate, rateErr := s.analyticsRate(ctx, &out, rates, row.PnlCurrency)
		if rateErr != nil {
			return out, nil, rateErr
		}
		net, gross, fees := row.NetPnl.Float64*rate, grossPnlOf(row)*rate, row.FeesTotal*rate
		for _, value := range []float64{net, gross, fees} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return out, nil, Fail(http.StatusBadGateway, "fx_unavailable", "invalid normalized monetary input", nil)
			}
		}
		trades = append(trades, analytics.ClosedTrade{NetPnl: net, GrossPnl: gross, FeesTotal: fees, OpenedAt: row.OpenedAt, ClosedAt: row.ClosedAt.Time})
	}
	return out, trades, nil
}

func (s *Server) summary(ctx context.Context, uid string, f Filters, requested string) (summaryResponse, error) {
	meta, trades, err := s.normalizedTrades(ctx, uid, f, requested)
	if err != nil {
		return summaryResponse{}, err
	}
	return summaryResponse{Summary: analytics.Summarize(trades), currencyMetadata: meta}, nil
}

func (s *Server) analyticsRate(ctx context.Context, meta *currencyMetadata, rates map[string]float64, source string) (float64, error) {
	source = strings.ToUpper(strings.TrimSpace(source))
	if !validCurrency(source) {
		return 0, failLoad(errUnknownCurrency, "monetary input currency could not be resolved")
	}
	if rate, ok := rates[source]; ok {
		return rate, nil
	}
	if s.deps.Market == nil {
		return 0, Fail(http.StatusBadGateway, "fx_unavailable", "analytics FX is unavailable", nil)
	}
	fx, err := s.deps.Market.GetFxRate(ctx, source, meta.Currency)
	if err != nil || fx.Rate <= 0 || math.IsNaN(fx.Rate) || math.IsInf(fx.Rate, 0) {
		return 0, Fail(http.StatusBadGateway, "fx_unavailable", "analytics FX is unavailable", nil)
	}
	rates[source] = fx.Rate
	meta.FXRates = append(meta.FXRates, fx)
	return fx.Rate, nil
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
