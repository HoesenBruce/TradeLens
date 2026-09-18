package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/accountvalue"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/marketdata"
)

func TestAccountValueContractFiltersRangeAndPreservesOpeningState(t *testing.T) {
	s := testServerWithAccountValue(t, true, accountValueBars(map[string]map[string]float64{
		"1306": {"2026-09-01": 100, "2026-09-02": 101},
		"5401": {"2026-08-10": 10, "2026-09-01": 12, "2026-09-02": 15},
	}))
	token := registerAndLogin(t, s, "account-value@example.com")
	accountA := createAccount(t, s, token, "A", "JPY")
	accountB := createAccount(t, s, token, "B", "JPY")
	postCash(t, s, token, accountA, 1000, "2026-08-01T01:00:00Z")
	postCash(t, s, token, accountB, 200, "2026-08-01T01:00:00Z")
	rec := do(s, http.MethodPost, "/api/v1/executions", `{
		"account_id":"`+accountA+`","symbol":"5401","instrument_type":"stock",
		"side":"buy","quantity":10,"price":10,"executed_at":"2026-08-10T01:00:00Z",
		"details":{"lot":"sbi:cash"}}`, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	rec = do(s, http.MethodGet, "/api/v1/analytics/account-value?account_id="+accountA+"&from=2026-09-01&to=2026-09-02", "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	response := decodeAccountValue(t, rec.Body.Bytes())
	require.Equal(t, "JPY", response.Currency)
	require.Equal(t, []string{"2026-09-01", "2026-09-02"}, []string{response.Points[0].Date, response.Points[1].Date})
	require.Equal(t, 1020.0, *response.Points[0].EstimatedAccountValue)
	require.Equal(t, 1050.0, *response.Points[1].EstimatedAccountValue)
	require.Equal(t, 1000.0, response.Points[0].ContributedCapital)

	rec = do(s, http.MethodGet, "/api/v1/analytics/account-value?from=2026-09-01&to=2026-09-02", "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	response = decodeAccountValue(t, rec.Body.Bytes())
	require.Equal(t, 1220.0, *response.Points[0].EstimatedAccountValue)

	rec = do(s, http.MethodGet, "/api/v1/analytics/equity-curve?account_id="+accountA, "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestAccountValueEmptyValidationAndMixedCurrencies(t *testing.T) {
	s := testServerWithAccountValue(t, true, accountValueBars(map[string]map[string]float64{}))
	token := registerAndLogin(t, s, "account-value-empty@example.com")
	jpy := createAccount(t, s, token, "JPY", "JPY")
	_ = createAccount(t, s, token, "USD", "USD")

	rec := do(s, http.MethodGet, "/api/v1/analytics/account-value?account_id="+jpy+"&from=2026-09-01&to=2026-09-02", "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Empty(t, decodeAccountValue(t, rec.Body.Bytes()).Points)

	rec = do(s, http.MethodGet, "/api/v1/analytics/account-value?from=bad&to=2026-09-02", "", token)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = do(s, http.MethodGet, "/api/v1/analytics/account-value?from=2026-09-03&to=2026-09-02", "", token)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = do(s, http.MethodGet, "/api/v1/analytics/account-value?from=2026-09-01&to=2026-09-02", "", token)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountValuePropagatesMissingAndCorporateActionWarnings(t *testing.T) {
	loader := func(_ context.Context, req marketdata.Request) (marketdata.Response, error) {
		switch req.Symbol {
		case "1306":
			return marketResponse("1306", map[string]float64{"2026-09-02": 100}), nil
		case "MISS":
			return marketResponse("MISS", map[string]float64{}), nil
		case "SPLT":
			return marketResponse("SPLT", map[string]float64{"2026-09-01": 100, "2026-09-02": 50}), nil
		default:
			return marketdata.Response{}, errors.New("unexpected symbol")
		}
	}
	s := testServerWithAccountValue(t, true, loader)
	token := registerAndLogin(t, s, "account-value-warning@example.com")
	account := createAccount(t, s, token, "A", "JPY")
	postCash(t, s, token, account, 1000, "2026-08-01T01:00:00Z")
	for _, symbol := range []string{"MISS", "SPLT"} {
		rec := do(s, http.MethodPost, "/api/v1/executions", `{
			"account_id":"`+account+`","symbol":"`+symbol+`","instrument_type":"stock",
			"side":"buy","quantity":1,"price":100,"executed_at":"2026-09-01T01:00:00Z",
			"details":{"lot":"sbi:cash"}}`, token)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}

	rec := do(s, http.MethodGet, "/api/v1/analytics/account-value?account_id="+account+"&from=2026-09-02&to=2026-09-02", "", token)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	point := decodeAccountValue(t, rec.Body.Bytes()).Points[0]
	require.Equal(t, "unsupported_corporate_action", point.Status)
	require.Nil(t, point.EstimatedAccountValue)
	codes := make([]string, len(point.Warnings))
	for i, warning := range point.Warnings {
		codes[i] = warning.Code
	}
	require.Contains(t, codes, "missing_price")
	require.Contains(t, codes, "unsupported_corporate_action")
}

func TestAccountValueCarriesPreviousCloseOnlyAfterIgnore(t *testing.T) {
	s := testServerWithAccountValue(t, true, accountValueBars(map[string]map[string]float64{
		"1306": {"2026-09-01": 100, "2026-09-02": 100},
		"1328": {"2026-09-01": 100},
	}))
	token := registerAndLogin(t, s, "account-value-ignore@example.com")
	account := createAccount(t, s, token, "A", "JPY")
	postCash(t, s, token, account, 1000, "2026-08-01T01:00:00Z")
	rec := do(s, http.MethodPost, "/api/v1/executions", `{
		"account_id":"`+account+`","symbol":"1328","instrument_type":"stock",
		"side":"buy","quantity":1,"price":100,"executed_at":"2026-09-01T01:00:00Z",
		"details":{"lot":"sbi:cash"}}`, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	path := "/api/v1/analytics/account-value?account_id=" + account + "&from=2026-09-02&to=2026-09-02"
	rec = do(s, http.MethodGet, path, "", token)
	require.Equal(t, "incomplete_missing_price", decodeAccountValue(t, rec.Body.Bytes()).Points[0].Status)

	rec = do(s, http.MethodGet, path+"&ignored_missing_prices=1328:2026-09-02,bad,1328:not-a-date", "", token)
	point := decodeAccountValue(t, rec.Body.Bytes()).Points[0]
	require.Equal(t, "complete", point.Status)
	require.Equal(t, 1000.0, *point.EstimatedAccountValue)
	require.Equal(t, "carried_forward_suspension_price", point.Warnings[0].Code)
}

type accountValueAPIResponse struct {
	Currency string               `json:"currency"`
	Points   []accountvalue.Point `json:"points"`
}

func decodeAccountValue(t *testing.T, body []byte) accountValueAPIResponse {
	t.Helper()
	var response accountValueAPIResponse
	require.NoError(t, json.Unmarshal(body, &response))
	return response
}

func createAccount(t *testing.T, s *api.Server, token, name, currency string) string {
	t.Helper()
	rec := do(s, http.MethodPost, "/api/v1/accounts", `{"name":"`+name+`","base_currency":"`+currency+`"}`, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var account struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &account))
	return account.ID
}

func postCash(t *testing.T, s *api.Server, token, account string, amount float64, occurredAt string) {
	t.Helper()
	rec := do(s, http.MethodPost, "/api/v1/cash-transactions", `{"account_id":"`+account+`","type":"deposit","amount":`+strconv.FormatFloat(amount, 'f', -1, 64)+`,"currency":"JPY","occurred_at":"`+occurredAt+`"}`, token)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

func accountValueBars(series map[string]map[string]float64) func(context.Context, marketdata.Request) (marketdata.Response, error) {
	return func(_ context.Context, req marketdata.Request) (marketdata.Response, error) {
		return marketResponse(req.Symbol, series[req.Symbol]), nil
	}
}

func marketResponse(symbol string, closes map[string]float64) marketdata.Response {
	response := marketdata.Response{Instrument: symbol, Source: "test", AdjustmentStatus: "unadjusted"}
	for date, close := range closes {
		at, _ := time.Parse(time.DateOnly, date)
		response.Bars = append(response.Bars, marketdata.Bar{Time: at.Unix(), MarketDate: date, Open: close, High: close, Low: close, Close: close})
	}
	return response
}
