package api

import (
	"context"
	"math"
	"net/http"
	"strings"

	"github.com/tradermemos/api/internal/analytics"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
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

// analyticsCurrency resolves the shared target contract without changing stored money.
func (s *Server) analyticsCurrency(ctx context.Context, uid string, f Filters, requested string) (out currencyMetadata, err error) {
	target := strings.ToUpper(strings.TrimSpace(requested))
	if target != "" && !validCurrency(target) {
		return out, Fail(http.StatusBadRequest, "bad_request", "target_currency must be a three-letter currency code", nil)
	}
	currencies, err := s.portfolioCurrencies(ctx, uid, f)
	if err != nil {
		return out, failLoad(err, "could not resolve analytics currency")
	}
	if target == "" {
		if len(currencies) > 1 {
			return out, Fail(http.StatusBadRequest, "mixed_currencies", "target_currency is required for mixed-currency analytics", nil)
		}
		if len(currencies) == 1 {
			target = currencies[0]
		}
	}
	target = strings.ToUpper(strings.TrimSpace(target))
	if target != "" && !validCurrency(target) {
		return out, failLoad(errUnknownCurrency, "could not resolve analytics currency")
	}
	out.Currency, out.TargetCurrency = target, strings.ToUpper(strings.TrimSpace(requested))
	out.FXPolicy = "latest"
	out.FXRates = []marketdata.FxRateResponse{}
	return out, nil
}

func (s *Server) normalizedClosedRows(ctx context.Context, uid string, f Filters, requested string) (currencyMetadata, []store.Trade, error) {
	meta, err := s.analyticsCurrency(ctx, uid, f, requested)
	if err != nil {
		return meta, nil, err
	}
	rows, err := s.loadClosedTradeRows(ctx, uid, f)
	if err != nil {
		return meta, nil, failLoad(err, "could not load trades")
	}
	rates := map[string]float64{meta.Currency: 1}
	for i := range rows {
		if !rows[i].NetPnl.Valid || !rows[i].ClosedAt.Valid {
			continue
		}
		_, err := s.normalizeTrade(ctx, &meta, rates, &rows[i])
		if err != nil {
			return meta, nil, err
		}
	}
	return meta, rows, nil
}

func (s *Server) normalizeTrade(ctx context.Context, meta *currencyMetadata, rates map[string]float64, row *store.Trade) (float64, error) {
	rate, err := s.analyticsRate(ctx, meta, rates, row.PnlCurrency)
	if err != nil {
		return 0, err
	}
	if row.NetPnl.Valid {
		row.NetPnl.Float64 *= rate
	}
	if row.GrossPnl.Valid {
		row.GrossPnl.Float64 *= rate
	}
	row.FeesTotal *= rate
	for _, value := range []float64{row.NetPnl.Float64, row.GrossPnl.Float64, row.FeesTotal} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, Fail(http.StatusBadGateway, "fx_unavailable", "invalid normalized monetary input", nil)
		}
	}
	return rate, nil
}

func (s *Server) normalizedTrades(ctx context.Context, uid string, f Filters, requested string) (currencyMetadata, []analytics.ClosedTrade, error) {
	meta, rows, err := s.normalizedClosedRows(ctx, uid, f, requested)
	if err != nil {
		return meta, nil, err
	}
	return meta, toClosedTrades(rows), nil
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
